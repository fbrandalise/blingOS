import { useEffect, useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";
import type { PluggableList } from "unified";

import type { EntityKind } from "../../api/entity";
import { detectPlaybook } from "../../api/playbook";
import {
  fetchArticle,
  fetchHistory,
  fetchHumans,
  type HumanIdentity,
  proposeAmendment,
  subscribeEditLog,
  type WikiArticle as WikiArticleT,
  type WikiCatalogEntry,
  type WikiHistoryCommit,
} from "../../api/wiki";
import { formatAgentName } from "../../lib/agentName";
import {
  buildMarkdownComponents,
  buildRehypePlugins,
  buildRemarkPlugins,
} from "../../lib/wikiMarkdownConfig";
import ArticleStatusBanner from "./ArticleStatusBanner";
import ArticleTitle from "./ArticleTitle";
import Byline from "./Byline";
import CategoriesFooter from "./CategoriesFooter";
import CiteThisPagePanel from "./CiteThisPagePanel";
import EntityBriefBar from "./EntityBriefBar";
import EntityRelatedPanel from "./EntityRelatedPanel";
import FactsOnFile from "./FactsOnFile";
import HatBar, { type HatBarTab } from "./HatBar";
import Hatnote from "./Hatnote";
import PageFooter from "./PageFooter";
import PageStatsPanel from "./PageStatsPanel";
import PlaybookExecutionLog from "./PlaybookExecutionLog";
import PlaybookSkillBadge from "./PlaybookSkillBadge";
import ReferencedBy from "./ReferencedBy";
import SeeAlso from "./SeeAlso";
import type { SourceItem } from "./Sources";
import Sources from "./Sources";
import TocBox, { type TocEntry } from "./TocBox";
import WikiEditor from "./WikiEditor";

// Real backend paths look like `team/people/nazz.md`. Mock/dev paths may
// drop the `team/` prefix or the `.md` suffix. Accept both so the entity
// surface lights up in demos without forcing every caller to normalize.
const ENTITY_PATH_RE =
  /^(?:team\/)?(people|companies|customers)\/([a-z0-9][a-z0-9-]*)(?:\.md)?$/;

function detectEntity(path: string): { kind: EntityKind; slug: string } | null {
  const m = path.match(ENTITY_PATH_RE);
  if (!m) return null;
  return { kind: m[1] as EntityKind, slug: m[2] };
}

interface WikiArticleProps {
  path: string;
  catalog: WikiCatalogEntry[];
  onNavigate: (path: string) => void;
  /**
   * Bumped by Pam (now hoisted to the Wiki shell) when an action completes,
   * so the article + history refetch without a navigation. Treated as an
   * additive trigger on top of the local refreshNonce used by inline edits.
   */
  externalRefreshNonce?: number;
}

export default function WikiArticle({
  path,
  catalog,
  onNavigate,
  externalRefreshNonce = 0,
}: WikiArticleProps) {
  const [article, setArticle] = useState<WikiArticleT | null>(null);
  const [tab, setTab] = useState<HatBarTab>("article");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [historyCommits, setHistoryCommits] = useState<
    WikiHistoryCommit[] | null
  >(null);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [historyError, setHistoryError] = useState(false);
  const [liveAgent, setLiveAgent] = useState<string | null>(null);
  const [_refreshNonce, setRefreshNonce] = useState(0);
  const [humans, setHumans] = useState<HumanIdentity[]>([]);
  const [amendmentSent, setAmendmentSent] = useState<string | null>(null);

  // Fetch the human registry once per mount. The list is small (a handful
  // of team members) and changes rarely, so we skip refetching on every
  // path change. Failure falls through to an empty list — Byline gracefully
  // shows the agent path when no human identity matches.
  useEffect(() => {
    let cancelled = false;
    fetchHumans()
      .then((list) => {
        if (!cancelled) setHumans(list);
      })
      .catch(() => {
        if (!cancelled) setHumans([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchArticle(path)
      .then((a) => {
        if (cancelled) return;
        setArticle(a);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : "Failed to load article");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [path]);

  useEffect(() => {
    let cancelled = false;
    setHistoryCommits(null);
    setHistoryLoading(true);
    setHistoryError(false);
    fetchHistory(path)
      .then((res) => {
        if (cancelled) return;
        setHistoryCommits(res.commits ?? []);
      })
      .catch(() => {
        if (cancelled) return;
        // Graceful degradation: missing history should not block the article read.
        setHistoryError(true);
        setHistoryCommits([]);
      })
      .finally(() => {
        if (!cancelled) setHistoryLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [path]);

  useEffect(() => {
    setLiveAgent(null);
    let clearTimer: ReturnType<typeof setTimeout> | null = null;
    const unsubscribe = subscribeEditLog((entry) => {
      if (entry.article_path !== path) return;
      setLiveAgent(entry.who);
      if (clearTimer) clearTimeout(clearTimer);
      clearTimer = setTimeout(() => setLiveAgent(null), 10_000);
    });
    return () => {
      if (clearTimer) clearTimeout(clearTimer);
      unsubscribe();
    };
  }, [path]);

  const sourceItems = useMemo<SourceItem[]>(() => {
    if (!historyCommits) return [];
    return historyCommits.map((c) => ({
      commitSha: c.sha,
      authorSlug: c.author_slug,
      authorName: formatAgentName(c.author_slug),
      msg: c.msg,
      date: c.date,
    }));
  }, [historyCommits]);

  const catalogSlugs = useMemo(
    () => new Set(catalog.map((c) => c.path)),
    [catalog],
  );
  const resolver = useMemo(
    () => (slug: string) => catalogSlugs.has(slug),
    [catalogSlugs],
  );

  const remarkPlugins: PluggableList = useMemo(
    () => buildRemarkPlugins(resolver),
    [resolver],
  );
  const rehypePlugins: PluggableList = useMemo(() => buildRehypePlugins(), []);
  const markdownComponents = useMemo(
    () => buildMarkdownComponents({ resolver, onNavigate }),
    [resolver, onNavigate],
  );

  if (loading) return <div className="wk-loading">Loading article…</div>;
  if (error) return <div className="wk-error">Error: {error}</div>;
  if (!article) return <div className="wk-error">Article not found.</div>;

  const toc = buildTocFromMarkdown(article.content);
  const entity = detectEntity(article.path);
  const playbook = detectPlaybook(article.path);
  const breadcrumbSegments = article.path.split("/").filter(Boolean);
  const isLocked = article.locked === true;
  const context = breadcrumbSegments[0] || "";
  const byline = (
    <Byline
      authorSlug={article.last_edited_by}
      authorName={formatAgentName(article.last_edited_by)}
      lastEditedTs={article.last_edited_ts}
      revisions={article.revisions}
      humans={humans}
    />
  );

  return (
    <>
      <main className="wk-article-col">
        {isLocked && (
          <div className="wk-lock-banner" role="status" aria-label="Locked article">
            <span className="wk-lock-banner-icon" aria-hidden="true">🔒</span>
            <span className="wk-lock-banner-text">
              <strong>Big Bet</strong> — Esta tese está travada.
              Alterações devem ser aprovadas antes de serem aplicadas.
            </span>
            {tab !== "edit" && (
              <button
                type="button"
                className="wk-lock-banner-propose"
                onClick={() => setTab("edit")}
              >
                Propor alteração →
              </button>
            )}
          </div>
        )}
        {amendmentSent && (
          <div className="wk-amendment-confirm" role="status">
            <strong>Proposta enviada para revisão</strong> — ID{" "}
            <code>{amendmentSent}</code>. Aguardando aprovação.{" "}
            <button
              type="button"
              className="wk-amendment-confirm-close"
              onClick={() => setAmendmentSent(null)}
            >
              ×
            </button>
          </div>
        )}
        {liveAgent && (
          <ArticleStatusBanner
            message={`${formatAgentName(liveAgent)} is editing this article right now.`}
            liveAgent={liveAgent}
            revisions={article.revisions}
            contributors={article.contributors.length}
            wordCount={article.word_count}
          />
        )}
        {entity && (
          <EntityBriefBar
            kind={entity.kind}
            slug={entity.slug}
            onSynthesized={() => setRefreshNonce((n) => n + 1)}
          />
        )}
        {playbook && <PlaybookSkillBadge slug={playbook.slug} />}
        <HatBar
          active={tab}
          onChange={setTab}
          rightRail={context ? [context] : undefined}
          disabledTabs={isLocked ? ["talk"] : ["talk"]}
        />
        <div className="wk-breadcrumb">
          <a
            href="#/wiki"
            onClick={(e) => {
              e.preventDefault();
              onNavigate("");
            }}
          >
            Team Wiki
          </a>
          {breadcrumbSegments.map((seg, i) => (
            <span key={`${seg}-${i}`} style={{ display: "contents" }}>
              <span className="sep">›</span>
              {i < breadcrumbSegments.length - 1 ? (
                <a href="#">{seg}</a>
              ) : (
                <span>{article.title}</span>
              )}
            </span>
          ))}
        </div>
        <ArticleTitle title={article.title} />
        {byline}
        <Hatnote>
          This article is auto-generated from team activity. See the commit
          history for the full trail.
        </Hatnote>
        {tab === "article" && (
          <div className="wk-article-body" data-testid="wk-article-body">
            <ReactMarkdown
              remarkPlugins={remarkPlugins}
              rehypePlugins={rehypePlugins}
              components={markdownComponents}
            >
              {article.content}
            </ReactMarkdown>
          </div>
        )}
        {tab === "edit" && !isLocked && (
          <WikiEditor
            path={article.path}
            initialContent={article.content}
            expectedSha={article.commit_sha ?? ""}
            serverLastEditedTs={article.last_edited_ts}
            catalog={catalog}
            onSaved={(newSha) => {
              void newSha;
              setRefreshNonce((n) => n + 1);
              setTab("article");
            }}
            onCancel={() => setTab("article")}
          />
        )}
        {tab === "edit" && isLocked && (
          <BigBetAmendmentEditor
            path={article.path}
            initialContent={article.content}
            catalog={catalog}
            onSubmitted={(id) => {
              setAmendmentSent(id);
              setTab("article");
            }}
            onCancel={() => setTab("article")}
          />
        )}
        {tab === "raw" && (
          <pre
            style={{
              fontFamily: "var(--wk-mono)",
              background: "var(--wk-code-bg)",
              padding: 16,
              border: "1px solid var(--wk-border)",
              overflowX: "auto",
              fontSize: 13,
              lineHeight: 1.5,
              whiteSpace: "pre-wrap",
            }}
          >
            {article.content}
          </pre>
        )}
        {tab === "history" && (
          <div className="wk-loading">
            History view streams from <code>git log</code>. Wiring pending Lane
            A.
          </div>
        )}
        {entity && tab === "article" && (
          <FactsOnFile kind={entity.kind} slug={entity.slug} />
        )}
        {entity && tab === "article" && (
          <EntityRelatedPanel kind={entity.kind} slug={entity.slug} />
        )}
        {playbook && tab === "article" && (
          <PlaybookExecutionLog slug={playbook.slug} />
        )}
        <SeeAlso
          items={article.backlinks.map((b) => ({
            slug: b.path,
            display: b.title,
          }))}
          onNavigate={onNavigate}
        />
        {historyError ? null : (
          <Sources items={sourceItems} loading={historyLoading} />
        )}
        <CategoriesFooter tags={article.categories} />
        <PageFooter
          lastEditedBy={formatAgentName(article.last_edited_by)}
          lastEditedTs={article.last_edited_ts}
          articlePath={article.path}
        />
      </main>
      <aside className="wk-right-sidebar">
        <TocBox entries={toc} />
        <PageStatsPanel
          revisions={article.revisions}
          contributors={article.contributors.length}
          wordCount={article.word_count}
          created={article.last_edited_ts}
          lastEdit={article.last_edited_ts}
        />
        <CiteThisPagePanel slug={article.path} />
        <ReferencedBy backlinks={article.backlinks} onNavigate={onNavigate} />
      </aside>
    </>
  );
}

function buildTocFromMarkdown(md: string): TocEntry[] {
  const out: TocEntry[] = [];
  const lines = md.split("\n");
  let h2Count = 0;
  let h3Count = 0;
  const h2Re = /^##\s+(.+)$/;
  const h3Re = /^###\s+(.+)$/;
  for (const line of lines) {
    const h3 = line.match(h3Re);
    if (h3) {
      h3Count += 1;
      const title = h3[1].trim();
      out.push({
        level: 2,
        num: `${h2Count}.${h3Count}`,
        anchor: slugify(title),
        title,
      });
      continue;
    }
    const h2 = line.match(h2Re);
    if (h2) {
      h2Count += 1;
      h3Count = 0;
      const title = h2[1].trim();
      out.push({
        level: 1,
        num: String(h2Count),
        anchor: slugify(title),
        title,
      });
    }
  }
  return out;
}

function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

// ── BigBetAmendmentEditor ─────────────────────────────────────────────────────

interface BigBetAmendmentEditorProps {
  path: string;
  initialContent: string;
  catalog: WikiCatalogEntry[];
  onSubmitted: (promotionId: string) => void;
  onCancel: () => void;
}

function BigBetAmendmentEditor({
  path,
  initialContent,
  onSubmitted,
  onCancel,
}: BigBetAmendmentEditorProps) {
  const [content, setContent] = useState(initialContent);
  const [rationale, setRationale] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit() {
    const trimmedRationale = rationale.trim();
    if (!trimmedRationale) {
      setError("Descreva o motivo da alteração.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const result = await proposeAmendment({ path, content, rationale: trimmedRationale });
      onSubmitted(result.promotion_id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Erro ao enviar proposta.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="wk-amendment-editor">
      <div className="wk-amendment-editor-notice">
        <strong>Modo de proposta</strong> — Sua edição será submetida para
        revisão e não será publicada até ser aprovada.
      </div>
      <textarea
        className="wk-amendment-editor-textarea"
        value={content}
        onChange={(e) => setContent(e.target.value)}
        rows={24}
        spellCheck={false}
        aria-label="Conteúdo proposto"
      />
      <div className="wk-amendment-editor-footer">
        <input
          type="text"
          className="wk-amendment-editor-rationale"
          placeholder="Motivo da alteração (obrigatório)"
          value={rationale}
          onChange={(e) => setRationale(e.target.value)}
        />
        {error && <span className="wk-amendment-editor-error">{error}</span>}
        <div className="wk-amendment-editor-actions">
          <button
            type="button"
            className="wk-amendment-editor-cancel"
            onClick={onCancel}
            disabled={saving}
          >
            Cancelar
          </button>
          <button
            type="button"
            className="wk-amendment-editor-submit"
            onClick={() => void handleSubmit()}
            disabled={saving}
          >
            {saving ? "Enviando…" : "Enviar para revisão"}
          </button>
        </div>
      </div>
    </div>
  );
}
