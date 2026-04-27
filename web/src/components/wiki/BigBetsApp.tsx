import { useEffect, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import {
  type WikiCatalogEntry,
  createBigBet,
  fetchArticle,
  fetchCatalog,
  proposeAmendment,
  type WikiArticle,
} from "../../api/wiki";
import "../../styles/wiki.css";

function stripFrontmatter(raw: string): string {
  const m = raw.match(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/);
  return m ? raw.slice(m[0].length).trimStart() : raw;
}

function slugify(title: string): string {
  return title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 63);
}

function extractH1(md: string): string {
  const match = md.match(/^#\s+(.+)$/m);
  return match ? match[1].trim() : "";
}

// ── Creation form ────────────────────────────────────────────────

interface CreateFormProps {
  onCreated: (path: string) => void;
}

function CreateForm({ onCreated }: CreateFormProps) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [content, setContent] = useState("");
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [area, setArea] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const text = ev.target?.result as string;
      setContent(text);
      const h1 = extractH1(text);
      if (h1) {
        setTitle(h1);
        setSlug(slugify(h1));
      }
    };
    reader.readAsText(file);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!content) {
      setError("Selecione um arquivo .md.");
      return;
    }
    if (!slug) {
      setError("Informe o slug.");
      return;
    }
    setSubmitting(true);
    try {
      const controller = new AbortController();
      const timer = globalThis.setTimeout(() => controller.abort(), 10_000);
      const result = await createBigBet({ slug, title: title || slug, area, content });
      globalThis.clearTimeout(timer);
      onCreated(result.path);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg.includes("abort") ? "Servidor não respondeu. Tente dar Ctrl+Shift+R e submeter novamente." : msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="bb-create-form" onSubmit={handleSubmit}>
      <h2 className="bb-create-title">Nova Big Bet</h2>
      <p className="bb-create-hint">
        Envie o arquivo <code>.md</code> com a tese. Após criada, qualquer
        alteração passará por revisão e aprovação.
      </p>

      <label className="bb-field">
        <span>Arquivo (.md)</span>
        <div className="bb-file-row">
          <input
            ref={fileRef}
            type="file"
            accept=".md,text/markdown,text/plain"
            onChange={handleFile}
            style={{ display: "none" }}
          />
          <button
            type="button"
            className="bb-file-btn"
            onClick={() => fileRef.current?.click()}
          >
            Escolher arquivo
          </button>
          <span className="bb-file-name">
            {content ? `${content.length} caracteres carregados` : "Nenhum arquivo"}
          </span>
        </div>
      </label>

      <label className="bb-field">
        <span>Título</span>
        <input
          className="bb-input"
          value={title}
          onChange={(e) => {
            setTitle(e.target.value);
            setSlug(slugify(e.target.value));
          }}
          placeholder="ex: Expansão para PMEs"
        />
      </label>

      <label className="bb-field">
        <span>Slug (URL)</span>
        <input
          className="bb-input"
          value={slug}
          onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
          placeholder="ex: expansao-pmes"
        />
      </label>

      <label className="bb-field">
        <span>Área</span>
        <input
          className="bb-input"
          value={area}
          onChange={(e) => setArea(e.target.value)}
          placeholder="ex: produto, engenharia, marketing…"
        />
      </label>

      {error && <p className="bb-error">{error}</p>}

      <button
        type="submit"
        className="bb-submit-btn"
        disabled={submitting || !content}
      >
        {submitting ? "Enviando…" : "Criar Big Bet"}
      </button>
    </form>
  );
}

// ── Article detail ───────────────────────────────────────────────

interface DetailPanelProps {
  path: string;
  onBack: () => void;
}

function DetailPanel({ path, onBack }: DetailPanelProps) {
  const [article, setArticle] = useState<WikiArticle | null>(null);
  const [loading, setLoading] = useState(true);
  const [proposing, setProposing] = useState(false);
  const [submitted, setSubmitted] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setProposing(false);
    setSubmitted(null);
    fetchArticle(path)
      .then((a) => { if (!cancelled) setArticle(a); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [path]);

  if (loading) {
    return <div className="bb-detail-loading">Carregando…</div>;
  }
  if (!article) return null;

  const body = stripFrontmatter(article.content);

  return (
    <div className="bb-detail">
      <div className="bb-detail-header">
        <button type="button" className="bb-back-btn" onClick={onBack}>
          ← voltar
        </button>
        <div className="bb-detail-header-row">
          <h2 className="bb-detail-title">{article.title}</h2>
          {!proposing && !submitted && (
            <button
              type="button"
              className="bb-propose-btn"
              onClick={() => setProposing(true)}
            >
              Propor nova versão
            </button>
          )}
        </div>
        {article.locked && (
          <span className="bb-locked-badge">
            🔒 Imutável — alterações exigem revisão e aprovação
          </span>
        )}
      </div>

      {submitted && (
        <div className="wk-amendment-confirm" role="status">
          Proposta <code>{submitted}</code> enviada para revisão. Aguardando
          aprovação.{" "}
          <button
            type="button"
            className="wk-amendment-confirm-close"
            onClick={() => setSubmitted(null)}
          >
            ✕
          </button>
        </div>
      )}

      {proposing ? (
        <AmendmentForm
          path={path}
          initialBody={body}
          onSubmitted={(id) => { setSubmitted(id); setProposing(false); }}
          onCancel={() => setProposing(false)}
        />
      ) : (
        <>
          <div className="bb-detail-meta">
            <span>Última edição por <strong>{article.last_edited_by}</strong></span>
            <span>{new Date(article.last_edited_ts).toLocaleDateString("pt-BR")}</span>
            <span>{article.word_count} palavras</span>
          </div>
          <article className="bb-detail-body">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {body}
            </ReactMarkdown>
          </article>
        </>
      )}
    </div>
  );
}

// ── Amendment form ───────────────────────────────────────────────

interface AmendmentFormProps {
  path: string;
  initialBody: string;
  onSubmitted: (promotionId: string) => void;
  onCancel: () => void;
}

function AmendmentForm({ path, initialBody, onSubmitted, onCancel }: AmendmentFormProps) {
  const [content, setContent] = useState(initialBody);
  const [rationale, setRationale] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit() {
    if (!rationale.trim()) {
      setError("Descreva o motivo da alteração.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const result = await proposeAmendment({ path, content, rationale: rationale.trim() });
      onSubmitted(result.promotion_id);
    } catch (err) {
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

// ── Main BigBetsApp ──────────────────────────────────────────────

export default function BigBetsApp() {
  const [bets, setBets] = useState<WikiCatalogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<string | null>(null);
  const [creating, setCreating] = useState(true); // default to create form open

  useEffect(() => {
    let cancelled = false;
    const timer = globalThis.setTimeout(() => {
      if (!cancelled) setLoading(false);
    }, 5000);
    fetchCatalog().then((entries) => {
      if (cancelled) return;
      setBets(entries.filter((e) => e.group === "big-bets"));
      setLoading(false);
    }).catch(() => {
      if (!cancelled) setLoading(false);
    }).finally(() => globalThis.clearTimeout(timer));
    return () => {
      cancelled = true;
      globalThis.clearTimeout(timer);
    };
  }, []);

  function handleCreated(path: string) {
    setCreating(false);
    // Refresh list and open new article
    fetchCatalog().then((entries) => {
      setBets(entries.filter((e) => e.group === "big-bets"));
      setSelected(path);
    });
  }

  const showRight = selected !== null || creating;

  return (
    <div className="wiki-root bb-root">
      <div className="bb-layout">
        {/* Left: list */}
        <aside className="bb-sidebar">
          <div className="bb-sidebar-header">
            <span className="bb-sidebar-heading">Big Bets</span>
            <button
              type="button"
              className="bb-new-btn"
              onClick={() => { setCreating(true); setSelected(null); }}
            >
              + Nova
            </button>
          </div>

          {loading ? (
            <div className="bb-sidebar-loading">Carregando…</div>
          ) : bets.length === 0 ? (
            <div className="bb-sidebar-empty">
              Nenhuma big bet ainda.
              <br />
              <button
                type="button"
                className="bb-sidebar-empty-btn"
                onClick={() => setCreating(true)}
              >
                Criar a primeira →
              </button>
            </div>
          ) : (
            <ul className="bb-list">
              {bets.map((bet) => (
                <li key={bet.path}>
                  <button
                    type="button"
                    className={`bb-list-item${selected === bet.path ? " is-active" : ""}`}
                    onClick={() => { setSelected(bet.path); setCreating(false); }}
                  >
                    <span className="bb-list-title">{bet.title}</span>
                    <span className="bb-list-meta">
                      {new Date(bet.last_edited_ts).toLocaleDateString("pt-BR")}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </aside>

        {/* Right: detail or create form */}
        <main className="bb-main">
          {!showRight && (
            <div className="bb-empty-state">
              <p>Selecione uma big bet ou crie uma nova.</p>
            </div>
          )}
          {creating && <CreateForm onCreated={handleCreated} />}
          {selected && !creating && (
            <DetailPanel
              path={selected}
              onBack={() => setSelected(null)}
            />
          )}
        </main>
      </div>
    </div>
  );
}
