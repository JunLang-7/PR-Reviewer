# Design: AI PR Reviewer

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         GitHub.com                              │
│  ┌──────────┐    ┌──────────┐    ┌─────────────────────────┐   │
│  │ PR Event │───▶│ Webhook  │───▶│  PR Comments            │   │
│  │ (opened/ │    │ (POST)   │    │  (Review output)        │   │
│  │  sync)   │    └──────────┘    └─────────────────────────┘   │
│  └──────────┘         │                  ▲                      │
└───────────────────────┼──────────────────┼──────────────────────┘
                        │                  │
                  ┌─────▼──────────────────┼──────┐
                  │     ngrok tunnel       │      │
                  └─────┬──────────────────┼──────┘
                        │                  │
┌───────────────────────▼──────────────────┼──────┐
│                   Go App Server                  │
│                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │ Webhook     │  │ Context     │  │ Comment  │ │
│  │ Handler     │  │ Builder     │  │ Publisher│ │
│  │             │  │             │  │          │ │
│  │ Validate    │  │ Fetch diff  │  │ Format   │ │
│  │ Route       │  │ Fetch files │  │ Post to  │ │
│  │ Ack fast    │  │ Assemble    │  │ PR       │ │
│  └──────┬──────┘  └──────┬──────┘  └────▲─────┘ │
│         │                │              │       │
│         │         ┌──────▼──────────────┼───┐   │
│         │         │    Analysis Pipeline    │   │
│         │         │                        │   │
│         │         │  Stage 1: Summary      │   │
│         │         │    ↓                   │   │
│         │         │  Stage 2: Risk Scan    │   │
│         │         │    ↓                   │   │
│         │         │  Stage 3: Suggestions  │   │
│         │         │    (conditional)       │   │
│         │         └────────────────────────┘   │
│         └──────────────────────────────────────┘
│                                                  │
│  ┌──────────────────────────────────────────┐   │
│  │           Claude API                      │   │
│  │  Haiku (summary) / Sonnet (risk)          │   │
│  │  Opus (suggestions, optional)             │   │
│  └──────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

## Request Flow

```
1. Webhook received
   │
2. Validate signature & event type
   │  - pull_request.opened → full incremental diff
   │  - pull_request.synchronize → latest commit diff only
   │  - Others → ignore (for v1)
   │
3. ACK immediately (202 Accepted) ─── GitHub needs fast response
   │
4. Context Building (async)
   │
   ├─ L0: Get diff via GitHub Compare API
   │     (base commit → head commit)
   │
   ├─ L1: For each changed file (< 20 files):
   │     Fetch full content via Contents API
   │     Skip binary / large files (> 500KB)
   │
   └─ L2: Conditional (skipped in v1 except explicit trigger)
   │
5. Analysis Pipeline
   │
   ├─ Stage 1: PR Summary (Claude Haiku)
   │     Input:  file list + diff summary
   │     Output: 3-5 sentence summary + impact scope
   │
   ├─ Stage 2: Risk Identification (Claude Sonnet)
   │     Input:  diff + changed files full content
   │     Output: risk list with severity/confidence/location
   │
   └─ Stage 3: Review Suggestions (Claude Sonnet/Opus)
   │     Condition: changed files <= 5 OR small diff
   │     Input:  full files + diff + Stage 2 results
   │     Output: line-level improvement suggestions
   │
6. Format & Publish
   │
   └─ Post as PR review comment (not inline comments for v1)
      Markdown formatted, severity-colored, code-positioned
```

## Context Building Strategy

### L0: Diff (always)

```
Source:  GET /repos/{owner}/{repo}/compare/{base}...{head}
Returns: file list, patch hunks, stats
Token est: ~2,500 for typical 500-line diff
```

### L1: Changed Files Full Content (always)

```
Source:  GET /repos/{owner}/{repo}/contents/{path}?ref={head}
Returns: base64-encoded file content
Filter:  skip files > 500KB, skip binary files
Limit:   max 20 files (above that, summary-only mode)
Token est: ~8,000 for 10 files averaging 200 lines each
```

### L2: Related Files (v1 skipped)

```
Reserved for future: same-directory files, caller lookup via static analysis
```

## Model Routing

```
┌──────────────┬──────────────┬─────────────────────────────┐
│    Stage     │    Model     │    Rationale                │
├──────────────┼──────────────┼─────────────────────────────┤
│ Stage 1      │ Haiku        │ Summarization is pattern    │
│ (Summary)    │ (cheap/fast) │ matching, not deep reasoning │
├──────────────┼──────────────┼─────────────────────────────┤
│ Stage 2      │ Sonnet       │ Risk scanning needs good    │
│ (Risk)       │ (balanced)   │ reasoning at moderate cost  │
├──────────────┼──────────────┼─────────────────────────────┤
│ Stage 3      │ Sonnet/Opus  │ Code suggestions need       │
│ (Suggestion) │ (powerful)   │ highest accuracy, only      │
│              │              │ triggered for small PRs     │
└──────────────┴──────────────┴─────────────────────────────┘
```

### Stage 3 trigger conditions

```
Run Stage 3 when ALL of:
  - changed files count <= 5
  - total diff lines < 500
  
Skip Stage 3 otherwise (cost control)
```

## Error Handling & Edge Cases

| Scenario | Handling |
|----------|----------|
| PR too large (>20 files) | Summary-only mode, skip per-file risk scanning |
| Binary files in diff | Skip from context, note in summary |
| API rate limit | Exponential backoff, retry up to 3 times |
| Claude API error | Post comment with error notice, don't crash |
| Webhook timeout | ACK before processing (202 Accepted) |
| Duplicate webhook | Idempotency via check_run external_id or comment dedup |

## Comment Output Format

```markdown
## AI Review Summary

**变更概述**: {Stage 1 summary}

**影响范围**: `{path1}`, `{path2}`, ...

**分析模式**: {basic / deep}

---

### Risk Scan (Stage 2)

#### Critical
- **{title}** `{file}:{line}`
  > {description}
  >
  > **建议修复**:
  > ```{lang}
  > {suggestion code}
  > ```
  > 置信度: {high/medium/low}

#### Warnings
...

#### Suggestions
...

---

### Detailed Suggestions (Stage 3, if triggered)

{line-level suggestions}

---

> AI Reviewer v1 · {timestamp}
```

Each risk item MUST include: title, file path with line number, description, severity level, and confidence level.

## Project Structure

```
PRReviewer/
├── cmd/
│   └── server/
│       └── main.go           # Entry point, server startup
├── internal/
│   ├── github/
│   │   ├── client.go         # GitHub API client wrapper
│   │   └── webhook.go        # Webhook signature verification & parsing
│   ├── context/
│   │   └── builder.go        # L0+L1 context assembly
│   ├── analyzer/
│   │   ├── pipeline.go       # Orchestrates stages 1-3
│   │   ├── summary.go        # Stage 1: PR summary via Haiku
│   │   ├── risk.go           # Stage 2: Risk identification via Sonnet
│   │   └── suggestion.go     # Stage 3: Review suggestions (conditional)
│   ├── comment/
│   │   └── publisher.go      # Format & post review comment to PR
│   └── config/
│       └── config.go         # App configuration (env vars)
├── go.mod
├── go.sum
└── Makefile
```

## Configuration (env vars)

```
GITHUB_APP_ID           # GitHub App ID
GITHUB_APP_PRIVATE_KEY   # Path to private key pem file
GITHUB_WEBHOOK_SECRET    # Webhook secret for signature verification
ANTHROPIC_API_KEY        # Claude API key
PORT                     # Server port (default: 8080)
```

## Future Extension Points

| Direction | v2+ Considerations |
|-----------|-------------------|
| Multi-model support | Abstract model interface, support OpenAI/本地模型 |
| Knowledge base | Store project conventions, historical review patterns per repo |
| Feedback loop | Collect 👍/👎, use for prompt tuning or fine-tuning |
| Inline comments | Post suggestions as GitHub review threads on specific lines |
| Static analysis integration | Run linters/security scanners before AI, use results as context |
| Custom rules | Per-repo review rules configured via `.ai-review.yml` |
| PR size control | Auto-label large PRs, suggest splitting |
| Support more events | `pull_request.reopened`, `check_run.requested_action` for manual trigger |
