# Tasks: AI PR Reviewer

## Phase 1: Project Scaffold & GitHub Integration

- [x] **1.1** Initialize Go module and project structure
  - `go mod init`, create directory layout `cmd/`, `internal/`
  - Add dependencies: `google/go-github`, `anthropic-sdk-go`

- [x] **1.2** Implement config loading
  - Load from env vars: `GITHUB_APP_ID`, `GITHUB_APP_PRIVATE_KEY`, `GITHUB_WEBHOOK_SECRET`, `ANTHROPIC_API_KEY`, `PORT`
  - Validate required vars on startup

- [x] **1.3** Implement GitHub App auth & client
  - JWT generation from App ID + private key
  - Installation access token exchange
  - GitHub API client with auto token refresh

- [x] **1.4** Implement webhook handler
  - POST `/webhook` endpoint
  - Signature verification (HMAC-SHA256)
  - Parse event type, extract PR metadata
  - 202 Accepted immediate response

## Phase 2: Context Building

- [x] **2.1** Implement L0 context: diff fetching
  - Compare API: `GET /repos/{owner}/{repo}/compare/{base}...{head}`
  - Extract file list, patch data, stats
  - Skip binary/uninteresting files

- [x] **2.2** Implement L1 context: full file content fetching
  - Contents API for each changed file at head ref
  - Size/filter limits (skip >500KB, max 20 files)
  - Handle rate limit with retry

- [x] **2.3** Implement context assembler
  - Build structured context for each analysis stage
  - Calculate token estimates
  - Decide Stage 3 eligibility (files <= 5 & diff < 500 lines)

## Phase 3: AI Analysis Pipeline

- [x] **3.1** Implement Claude API client wrapper
  - Init client with API key
  - Generic `Chat(prompt)` method
  - Error handling & retry

- [x] **3.2** Implement Stage 1: PR Summary (Haiku)
  - Prompt: summarize what this PR does, impact scope
  - Model: `claude-haiku-4-5`
  - Output: 3-5 sentence Chinese summary + affected paths

- [x] **3.3** Implement Stage 2: Risk Identification (Sonnet)
  - Prompt: identify security bugs, logic errors, concurrency issues, breaking changes
  - Model: `claude-sonnet-4-6`
  - Output: structured risk list (title, file:line, severity, confidence, description, fix suggestion)

- [x] **3.4** Implement Stage 3: Review Suggestions (conditional, Sonnet)
  - Prompt: detailed code improvement suggestions at line level
  - Model: `claude-sonnet-4-6`
  - Condition: small PR only (see 2.3)
  - Output: line-specific improvement suggestions

- [x] **3.5** Implement pipeline orchestrator
  - Run Stage 1 + 2 sequentially (Stage 2 may use Stage 1 output)
  - Run Stage 3 conditionally
  - Collect all results into structured format
  - Handle partial failures (e.g., Stage 2 fails → still post Stage 1)

## Phase 4: Comment Publishing

- [x] **4.1** Implement comment formatter
  - Markdown template with severity-colored sections
  - Code blocks with language hints
  - Links to source lines on GitHub
  - Timestamp and mode indicator

- [x] **4.2** Implement comment publisher
  - POST to PR issues/comments API
  - Handle comment body size limits
  - Add "AI Reviewer" identifier footer

- [x] **4.3** Handle PR update (synchronize) behavior
  - Post new comment (don't edit old one)
  - Include incremental diff context in new comment

## Phase 5: Integration & Testing

- [x] **5.1** Wire up full flow in main.go
  - HTTP server with `/webhook` endpoint
  - Background processing pipeline
  - Graceful shutdown

- [x] **5.2** Create ngrok setup guide
  - `ngrok http 8080`
  - GitHub App webhook URL configuration steps

- [x] **5.3** End-to-end smoke test
  - Create test PR, verify comment appears
  - Push update, verify new comment appears
  - Test large PR behavior (20+ files)

- [x] **5.4** Error handling polish
  - Logging for all failure modes
  - Meaningful error comments on PR when analysis fails
  - Rate limit handling
