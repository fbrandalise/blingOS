package team

// broker_human.go owns the HTTP surface for per-human wiki authoring
// (v1.5 hardening of PR #192). Two endpoints:
//
//	POST /wiki/write-human   — save a human edit, optimistic concurrency
//	GET  /humans             — list registered human identities
//
// Both flow through the existing broker bearer-token middleware, same
// as every other /wiki/* endpoint.

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// humanIdentityRegistry is the process-wide registry the broker uses to
// resolve per-human git identities at `/wiki/write-human` time. It is
// lazy-initialized on first use to avoid probing `git config` before
// the broker is actually serving requests.
var (
	humanIdentityRegistryOnce sync.Once
	humanIdentityRegistrySvc  *HumanIdentityRegistry
)

// brokerHumanIdentityRegistry returns the shared registry. Exported only
// through this helper so tests can swap it by calling
// setHumanIdentityRegistry before a test run.
func brokerHumanIdentityRegistry() *HumanIdentityRegistry {
	humanIdentityRegistryOnce.Do(func() {
		humanIdentityRegistrySvc = NewHumanIdentityRegistry()
	})
	return humanIdentityRegistrySvc
}

// setHumanIdentityRegistry is the test hook. Safe to call from TestMain
// or individual tests — subsequent sync.Once runs are no-ops.
func setHumanIdentityRegistry(r *HumanIdentityRegistry) {
	humanIdentityRegistryOnce.Do(func() {})
	humanIdentityRegistrySvc = r
}

// handleWikiWriteHuman is the broker HTTP endpoint the web UI posts to
// when a human saves a wiki edit. Shape:
//
//	POST /wiki/write-human
//	{
//	  "path": "team/people/nazz.md",
//	  "content": "...",
//	  "commit_message": "human: fix typo",
//	  "expected_sha": "abc123"
//	}
//
// expected_sha MUST be the article's current SHA as last seen by the
// client. When HEAD has moved, the handler returns 409 with the current
// SHA and the current article bytes so the editor can prompt re-apply.
//
// Agents never reach this endpoint — it is HTTP-only (not exposed via
// MCP) and gated by the existing broker bearer token (held by the web
// UI). The identity stamped on the commit is resolved server-side from
// the HumanIdentityRegistry; clients cannot forge attribution.
//
// Responses:
//
//	200 { "path":..., "commit_sha":..., "bytes_written":..., "author_slug":... }
//	400 { "error":"..." } on malformed JSON / bad path / empty content
//	409 { "error":"...", "current_sha":..., "current_content":"..." }
//	429 { "error":"wiki queue saturated, retry on next turn" }
//	500 { "error":"..." }
//	503 { "error":"wiki backend is not active" }
func (b *Broker) handleWikiWriteHuman(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		http.Error(w, `{"error":"wiki backend is not active"}`, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Path          string `json:"path"`
		Content       string `json:"content"`
		CommitMessage string `json:"commit_message"`
		ExpectedSHA   string `json:"expected_sha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	// Pre-validate inputs BEFORE enqueueing so a rejection never touches
	// the working tree. Mirrors reviewApprove's CanApprove pre-check.
	if err := validateArticlePath(body.Path); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}
	// Big-bet gate: locked articles go through the amendment review queue.
	if isBigBetPath(body.Path) {
		b.handleBigBetAmendment(w, r, body.Path, body.Content, body.CommitMessage)
		return
	}

	identity := brokerHumanIdentityRegistry().Local()
	sha, n, err := worker.EnqueueHumanAs(
		r.Context(), identity, body.Path, body.Content, body.CommitMessage, body.ExpectedSHA,
	)
	if err != nil {
		if errors.Is(err, ErrWikiSHAMismatch) {
			// Return the current article bytes so the editor can show a
			// three-pane reload prompt without a second round trip.
			current, _ := readArticle(worker.Repo(), body.Path)
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":           err.Error(),
				"current_sha":     sha,
				"current_content": string(current),
			})
			return
		}
		if errors.Is(err, ErrQueueSaturated) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":          body.Path,
		"commit_sha":    sha,
		"bytes_written": n,
		"author_slug":   identity.Slug,
	})
}

// isBigBetPath reports whether a wiki article path lives under team/big-bets/.
func isBigBetPath(p string) bool {
	return strings.HasPrefix(filepath.ToSlash(p), "team/big-bets/")
}

// handleBigBetAmendment intercepts a write to a locked big-bet article and
// routes it to the review queue instead of applying it directly.
func (b *Broker) handleBigBetAmendment(w http.ResponseWriter, r *http.Request, path, content, rationale string) {
	worker := b.WikiWorker()
	if worker == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "wiki backend is not active"})
		return
	}
	current, err := readArticle(worker.Repo(), path)
	if err != nil {
		// Article doesn't exist yet — must be created via POST /wiki/big-bets.
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "big-bet articles must be created via POST /wiki/big-bets",
		})
		return
	}
	isLocked, _ := parseLockFrontmatter(string(current))
	if !isLocked {
		// Exists but not locked — treat as normal write (unlocked big-bet is an edge case).
		identity := brokerHumanIdentityRegistry().Local()
		sha, n, err := worker.EnqueueHumanAs(r.Context(), identity, path, content, rationale, "")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"path": path, "commit_sha": sha, "bytes_written": n})
		return
	}

	rl := b.ReviewLog()
	if rl == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "review backend is not active"})
		return
	}
	identity := brokerHumanIdentityRegistry().Local()
	promotion, err := rl.SubmitAmendment(SubmitAmendmentRequest{
		SubmitterSlug:   identity.Slug,
		TargetPath:      path,
		ProposedContent: content,
		Rationale:       rationale,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	b.publishReviewEvent(ReviewStateChangeEvent{
		ID:        promotion.ID,
		OldState:  "",
		NewState:  promotion.State,
		ActorSlug: identity.Slug,
		Timestamp: promotion.CreatedAt.Format(time.RFC3339),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"promotion_id":  promotion.ID,
		"reviewer_slug": promotion.ReviewerSlug,
		"state":         promotion.State,
		"message":       "amendment submitted for review",
	})
}

// bigBetSlugRE validates big-bet slugs: lowercase alphanumeric and dashes.
var bigBetSlugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// handleBigBetCreate creates a new big-bet article with locked frontmatter.
//
//	POST /wiki/big-bets
//	{ "slug": "...", "title": "...", "area": "...", "content": "..." }
func (b *Broker) handleBigBetCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "wiki backend is not active"})
		return
	}
	var body struct {
		Slug    string `json:"slug"`
		Title   string `json:"title"`
		Area    string `json:"area"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if !bigBetSlugRE.MatchString(body.Slug) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be lowercase alphanumeric with dashes"})
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	path := "team/big-bets/" + body.Slug + ".md"
	if _, err := readArticle(worker.Repo(), path); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "a big bet with this slug already exists"})
		return
	}

	identity := brokerHumanIdentityRegistry().Local()
	now := time.Now().UTC().Format(time.RFC3339)
	area := strings.TrimSpace(body.Area)
	if area == "" {
		area = "general"
	}
	frontmatter := "---\nkind: big_bet\ncanonical_slug: " + body.Slug +
		"\narea: " + area +
		"\nlocked: true\nlocked_at: " + now +
		"\nlock_version: 1\ncreated_by: " + identity.Slug +
		"\ncreated_at: " + now + "\n---\n"

	title := strings.TrimSpace(body.Title)
	bodyText := strings.TrimSpace(body.Content)
	var fullContent string
	if bodyText != "" {
		fullContent = frontmatter + "\n# " + title + "\n\n" + bodyText
	} else {
		fullContent = frontmatter + "\n# " + title + "\n"
	}

	sha, n, err := worker.EnqueueHumanAs(r.Context(), identity, path, fullContent, "feat(big-bet): "+title, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":          path,
		"commit_sha":    sha,
		"bytes_written": n,
		"author_slug":   identity.Slug,
	})
}

// humanIdentityResponse is the wire shape for GET /humans. We deliberately
// expose name + slug (UI byline lookups) and email (for `mailto:` links
// and git-log reconciliation) but not CreatedAt — clients have no use
// for it today, and leaving it out keeps the response stable.
type humanIdentityResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Slug  string `json:"slug"`
}

// handleHumans returns the list of registered human identities. The
// registry is merge-on-read, so this endpoint only exposes identities
// that have been observed by at least one commit or probed from the
// local shell's `git config --global`.
//
//	GET /humans
//	  200 { "humans": [ {name, email, slug}, ... ] }
//
// No query params, no pagination — team-scale only (handful of humans).
func (b *Broker) handleHumans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	reg := brokerHumanIdentityRegistry()
	// Ensure the local identity has been probed at least once so a
	// fresh-install `GET /humans` doesn't return an empty list.
	_ = reg.Local()
	list := reg.List()
	out := make([]humanIdentityResponse, 0, len(list))
	for _, id := range list {
		out = append(out, humanIdentityResponse{
			Name:  id.Name,
			Email: id.Email,
			Slug:  id.Slug,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"humans": out})
}
