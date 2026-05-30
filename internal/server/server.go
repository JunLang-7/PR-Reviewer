package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/junlang/PRReviewer/internal/analyzer"
	"github.com/junlang/PRReviewer/internal/comment"
	prcontext "github.com/junlang/PRReviewer/internal/context"
	ghclient "github.com/junlang/PRReviewer/internal/github"
)

type Server struct {
	appClient      *ghclient.Client
	pipeline       *analyzer.Pipeline
	webhookHandler *ghclient.WebhookHandler
}

func New(appClient *ghclient.Client, pipeline *analyzer.Pipeline, webhookHandler *ghclient.WebhookHandler) *Server {
	return &Server{
		appClient:      appClient,
		pipeline:       pipeline,
		webhookHandler: webhookHandler,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	_, info, err := s.webhookHandler.Handle(w, r)
	if err != nil {
		log.Printf("webhook error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if info == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic] %s/%s #%d: %v", info.Owner, info.Repo, info.PRNumber, r)
			}
		}()
		s.processPR(info)
	}()
}

func (s *Server) processPR(info *ghclient.PRInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	mode := "auto"
	if info.Action == "comment" {
		mode = "manual"
	}

	log.Printf("[%s/%s #%d] 开始 Review (mode=%s)", info.Owner, info.Repo, info.PRNumber, mode)

	ghClient, err := s.appClient.NewInstallationClient(ctx, info.InstallationID)
	if err != nil {
		log.Printf("[%s/%s #%d] 创建客户端失败: %v", info.Owner, info.Repo, info.PRNumber, err)
		return
	}

	// Comment triggers don't have SHA info; fetch PR details first
	if info.BaseSHA == "" || info.HeadSHA == "" {
		pr, _, err := ghClient.PullRequests.Get(ctx, info.Owner, info.Repo, info.PRNumber)
		if err != nil {
			log.Printf("[%s/%s #%d] 获取 PR 信息失败: %v", info.Owner, info.Repo, info.PRNumber, err)
			return
		}
		info.BaseSHA = pr.Base.GetSHA()
		info.HeadSHA = pr.Head.GetSHA()
	}

	builder := prcontext.NewBuilder(ghClient.Repositories)
	prCtx, err := builder.Build(ctx, info.Owner, info.Repo, info.BaseSHA, info.HeadSHA)
	if err != nil {
		log.Printf("[%s/%s #%d] 获取上下文失败: %v", info.Owner, info.Repo, info.PRNumber, err)
		return
	}

	// Manual trigger forces Stage 3 depth
	stage3 := prCtx.Stage3Eligible
	if info.Action == "comment" {
		stage3 = true
	}

	log.Printf("[%s/%s #%d] 已获取 %d 个文件 (%d 行 diff), 开始 AI 分析...",
		info.Owner, info.Repo, info.PRNumber, prCtx.TotalFiles, prCtx.TotalDiffLines)

	result, err := s.pipeline.Run(ctx, analyzer.PipelineInput{
		Diff:           buildDiffString(prCtx),
		FileContents:   prCtx.FileContents,
		Stage3Eligible: stage3,
	})
	if err != nil {
		log.Printf("[%s/%s #%d] AI 分析异常: %v", info.Owner, info.Repo, info.PRNumber, err)
	}

	publisher := comment.NewPublisher(ghClient.PullRequests)
	if err := publisher.Publish(ctx, info.Owner, info.Repo, info.PRNumber, result); err != nil {
		log.Printf("[%s/%s #%d] 发布评论失败: %v", info.Owner, info.Repo, info.PRNumber, err)
		return
	}

	log.Printf("[%s/%s #%d] Review 完成 (文件=%d, 风险=%d)",
		info.Owner, info.Repo, info.PRNumber, prCtx.TotalFiles, riskCount(result))
}

func riskCount(result *analyzer.AnalysisResult) int {
	if result.Risks != nil {
		return len(result.Risks.Risks)
	}
	return 0
}

func buildDiffString(prCtx *prcontext.PRContext) string {
	var result string
	for _, f := range prCtx.DiffFiles {
		result += "--- a/" + f.Path + "\n+++ b/" + f.Path + "\n" + f.Patch + "\n"
	}
	return result
}
