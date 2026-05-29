package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v69/github"

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
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	log.Printf("received request at / — webhook URL should end with /webhook")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PR Reviewer is running. Webhook endpoint: POST /webhook"))
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

	if info.Action != "opened" && info.Action != "synchronize" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic] PR #%d: %v", info.PRNumber, r)
			}
		}()
		s.processPR(info)
	}()
}

func (s *Server) processPR(info *ghclient.PRInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("[%s/%s #%d] ====== 开始处理 ======", info.Owner, info.Repo, info.PRNumber)
	log.Printf("[%s/%s #%d] action=%s base=%.8s head=%.8s install=%d",
		info.Owner, info.Repo, info.PRNumber, info.Action, info.BaseSHA, info.HeadSHA, info.InstallationID)

	// Step 1: Create installation client
	log.Printf("[%s/%s #%d] 步骤1: 创建 installation client...", info.Owner, info.Repo, info.PRNumber)
	ghClient, err := s.appClient.NewInstallationClient(ctx, info.InstallationID)
	if err != nil {
		log.Printf("[%s/%s #%d] 步骤1 失败: %v", info.Owner, info.Repo, info.PRNumber, err)
		return
	}
	log.Printf("[%s/%s #%d] 步骤1: 完成", info.Owner, info.Repo, info.PRNumber)

	// Step 2: Build context
	log.Printf("[%s/%s #%d] 步骤2: 获取 PR diff 和文件内容...", info.Owner, info.Repo, info.PRNumber)
	builder := prcontext.NewBuilder(ghClient.Repositories)
	prCtx, err := builder.Build(ctx, info.Owner, info.Repo, info.BaseSHA, info.HeadSHA)
	if err != nil {
		log.Printf("[%s/%s #%d] 步骤2 失败: %v", info.Owner, info.Repo, info.PRNumber, err)
		s.postErrorComment(ctx, ghClient, info, "获取 PR 上下文失败: "+err.Error())
		return
	}
	log.Printf("[%s/%s #%d] 步骤2: 完成 (%d 个文件, %d 行 diff)", info.Owner, info.Repo, info.PRNumber, prCtx.TotalFiles, prCtx.TotalDiffLines)

	// Step 3: AI pipeline
	log.Printf("[%s/%s #%d] 步骤3: AI 分析中...", info.Owner, info.Repo, info.PRNumber)
	result, err := s.pipeline.Run(ctx, analyzer.PipelineInput{
		Diff:           buildDiffString(prCtx),
		FileContents:   prCtx.FileContents,
		Stage3Eligible: prCtx.Stage3Eligible,
	})
	if err != nil {
		log.Printf("[%s/%s #%d] 步骤3 失败: %v", info.Owner, info.Repo, info.PRNumber, err)
	}
	log.Printf("[%s/%s #%d] 步骤3: 完成 (summary=%v, risks=%d)",
		info.Owner, info.Repo, info.PRNumber,
		result.Summary != nil && result.Summary.Error == nil,
		riskCount(result))

	// Diagnostic: check what permissions the installation token actually has
	_, resp, _ := ghClient.Issues.CreateComment(ctx, info.Owner, info.Repo, info.PRNumber,
		&github.IssueComment{Body: github.Ptr("test")})
	if resp != nil {
		log.Printf("[%s/%s #%d] 诊断: 测试评论 HTTP %d", info.Owner, info.Repo, info.PRNumber, resp.StatusCode)
		log.Printf("[%s/%s #%d] 诊断: X-Accepted-Github-Permissions = %s",
			info.Owner, info.Repo, info.PRNumber, resp.Header.Get("X-Accepted-Github-Permissions"))
		log.Printf("[%s/%s #%d] 诊断: X-Github-Media-Type = %s",
			info.Owner, info.Repo, info.PRNumber, resp.Header.Get("X-Github-Media-Type"))
	}

	// Step 4: Publish comment
	log.Printf("[%s/%s #%d] 步骤4: 发布评论...", info.Owner, info.Repo, info.PRNumber)
	publisher := comment.NewPublisher(ghClient.Issues)
	err = publisher.Publish(ctx, info.Owner, info.Repo, info.PRNumber, result)
	if err != nil {
		log.Printf("[%s/%s #%d] 步骤4 失败: %v", info.Owner, info.Repo, info.PRNumber, err)
		return
	}
	log.Printf("[%s/%s #%d] 步骤4: 评论已发布", info.Owner, info.Repo, info.PRNumber)

	log.Printf("[%s/%s #%d] ====== 处理完成 ======", info.Owner, info.Repo, info.PRNumber)
}

func riskCount(result *analyzer.AnalysisResult) int {
	if result.Risks != nil {
		return len(result.Risks.Risks)
	}
	return 0
}

func (s *Server) postErrorComment(ctx context.Context, ghClient *github.Client, info *ghclient.PRInfo, msg string) {
	body := "## AI Review Error\n\n> " + msg + "\n\n---\n> AI Reviewer v1"
	comment := &github.IssueComment{Body: &body}
	_, _, err := ghClient.Issues.CreateComment(ctx, info.Owner, info.Repo, info.PRNumber, comment)
	if err != nil {
		log.Printf("post error comment failed: %v", err)
	}
}

func buildDiffString(prCtx *prcontext.PRContext) string {
	var result string
	for _, f := range prCtx.DiffFiles {
		result += "--- a/" + f.Path + "\n+++ b/" + f.Path + "\n" + f.Patch + "\n"
	}
	return result
}
