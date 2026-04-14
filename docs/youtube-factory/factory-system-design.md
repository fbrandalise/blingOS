# YouTube Factory System Design

Date: 2026-04-14
Owner: Engineering

## Goal

Turn the existing strategy assets in this repo into a mostly automated production system that can:

1. discover viable niches and topics
2. generate a research packet and script package
3. assemble voice, visuals, captions, and thumbnail assets
4. generate a publish-ready payload with monetization metadata
5. schedule uploads and ingest performance data
6. re-score the backlog from actual outcomes

The key constraint is speed. This repo already has structured GTM inputs in `docs/youtube-factory/*.yaml` and an existing Remotion video package in [`video/`](../../video). The fastest viable move is to keep the stack TypeScript-first, keep state simple, and add one control plane instead of a zoo of automation tools.

## Design Principles

- Keep one workflow per episode all the way through the pipeline.
- Treat the YAML files in `docs/youtube-factory/` as the source of truth for channel strategy, backlog rules, CTA routing, and pass gates.
- Make every pipeline stage idempotent and restartable from artifacts on disk plus one small database.
- Prefer templates over free-form generation for thumbnails, descriptions, CTAs, and scene composition.
- Push human review to the last responsible moment instead of sprinkling approvals everywhere.

## Recommended V1 Stack

| Layer | Choice | Why this is the fastest viable option here |
| --- | --- | --- |
| Language/runtime | Node 22 + TypeScript | Matches the existing `video/` package and keeps the whole factory in one language. |
| Package layout | `pnpm` workspace with `apps/factory` and `packages/config` | Light structure, no monorepo theater. |
| Orchestration | A single worker CLI with stage runners | Easier than standing up Temporal, Airflow, or BullMQ for one channel. |
| Persistence | SQLite via `better-sqlite3` | Good enough for one channel and one operator; zero extra infra. |
| Config/schema | `zod` + `yaml` | Validates the existing strategy assets before jobs run. |
| Video rendering | Remotion in the existing `video/` package | Already in repo and good at deterministic templated video output. |
| Voice | ElevenLabs adapter behind a `tts` interface | Proven fit for the current repo workflow and high enough quality for faceless narration. |
| Thumbnails | Remotion or HTML/SVG template renders first, image model optional second | Keeps brand consistency and avoids unstable slop thumbnails. |
| Scheduling | GitHub Actions or a simple cron runner invoking CLI commands | Enough for daily discovery and analytics sync. |
| Upload + analytics | `googleapis` client for YouTube Data API + YouTube Analytics API | Direct, scriptable, and keeps publishing state in our system. |
| Asset storage | Local `artifacts/` in dev, S3-compatible bucket in prod | Local speed now, easy lift later. |

## Suggested Repo Shape

```text
apps/
  factory/
    src/
      cli.ts
      db.ts
      stages/
      services/
      templates/
packages/
  config/
    src/
      strategy.ts
      backlog.ts
      monetization.ts
video/
  src/
  public/
docs/
  youtube-factory/
artifacts/
  episodes/<episode-id>/
```

## Core Data Model

| Entity | Purpose | Key fields |
| --- | --- | --- |
| `niche_candidates` | Candidate search lanes and clusters | `name`, `query_set`, `fit_score`, `status` |
| `topic_candidates` | Raw topic ideas before greenlighting | `workflow`, `pain`, `search_intent`, `proof_asset`, `score_breakdown` |
| `episodes` | Canonical record for one planned/published video | `episode_id`, `topic_id`, `pillar`, `cta_offer_id`, `stage`, `owner` |
| `episode_runs` | Retryable execution history per stage | `episode_id`, `stage`, `attempt`, `status`, `error`, `started_at`, `finished_at` |
| `assets` | Produced files and manifests | `episode_id`, `kind`, `path`, `checksum`, `version` |
| `publish_jobs` | Upload state and YouTube identifiers | `episode_id`, `youtube_video_id`, `scheduled_at`, `publish_status` |
| `analytics_daily` | Daily performance snapshots | `video_id`, `date`, `views`, `ctr`, `avg_view_duration`, `watch_time`, `subs` |
| `offer_events` | Monetization attribution | `video_id`, `offer_id`, `slot`, `clicks`, `leads`, `revenue` |

## End-to-End Pipeline

| Stage | Automation | Inputs | Outputs | Human gate |
| --- | --- | --- | --- | --- |
| 1. Niche discovery | Scheduled fetcher + scorer | Strategy YAML, competitor/channel seeds, search suggestions, comments | Ranked niche and query clusters | No |
| 2. Topic selection | Rules engine + LLM scoring | `content-backlog.yaml`, analytics deltas, comment themes, niche clusters | Selected topic packet | No |
| 3. Research packet build | Retrieval + structured summarizer | Topic packet, source URLs/transcripts, workflow references | Research packet, proof plan, citations | No |
| 4. Script package generation | JSON-constrained writer | Research packet, channel strategy, monetization rules | Hook, beats, script, CTA placement, shot list | No in steady state |
| 5. Voice generation | TTS job | Locked script | Narration audio, timestamps | No |
| 6. Visual assembly | Template compiler + render jobs | Script package, scene manifest, reusable components | Remotion manifest, video render, captions | No |
| 7. Thumbnail generation | Template variants + scorer | Packaging brief, channel visual system | 3 thumbnail candidates plus backup | No in steady state |
| 8. Publish payload generation | Deterministic generator | Title, thumbnail choice, CTA rules, disclosures | Description, pinned comment, tags, playlist, UTM map | No |
| 9. Pre-publish QA | Automated validators + human review | All assets and payload | Approve, block, or request rerun | Yes, hard gate |
| 10. Upload scheduling | API job | Approved publish bundle | Scheduled or published video | No after QA |
| 11. Analytics ingestion | Nightly sync | YouTube APIs, redirect events, lead events | Performance snapshots and optimization notes | No |
| 12. Backlog optimization | Rules + model summary | Analytics snapshots, comments, CTA performance | Updated topic scores and packaging experiments | No |

## Stage Design

### 1. Niche Discovery

The system should not guess niches from scratch every day. It should stay inside the channel thesis and hunt for under-served workflow lanes.

Inputs:

- channel thesis and anti-positioning from `channel-strategy.yaml`
- backlog workflow patterns from `content-backlog.yaml`
- competitor seed channels and transcripts
- comment and search suggestion ingestion

Output format:

- `niche-clusters.json`
- `topic-candidate-feed.json`

Scoring model:

- pain clarity
- buyer intent
- repeatability
- proof asset availability
- fit to one of the five approved pillars

### 2. Topic Selection

This is mostly a deterministic ranking problem with a small model assist.

Process:

1. Start with the structured backlog.
2. Blend in fresh candidates from niche discovery and comment mining.
3. Reject anything that violates `automation-sops.yaml` pass gates.
4. Pick the best topic that preserves pillar mix and CTA mix.

Hard rule:

- the selected topic must already name a believable proof asset before script generation starts

### 3. Research Packet Build

Each episode gets a structured packet, not a prose blob. The packet should include:

- viewer problem in one sentence
- workflow before state
- workflow after state
- supporting claims with citations
- tools mentioned only where they map to a workflow step
- a screenshot and diagram wish list
- proof asset definition

The model should produce JSON. Markdown can be generated later for human readability.

### 4. Script Package Generation

The script system should generate four artifacts, not just narration text:

- `script.md`
- `script.json`
- `shot-list.json`
- `qa-claims.json`

The important move is to separate narrative structure from render instructions. That lets us rerender scenes or voice without regenerating the whole concept.

### 5. Voice and Video Assembly

This repo already has a usable Remotion base. V1 should extend it into a generic episode renderer instead of building a bespoke composition for every video.

Recommended pattern:

- `video/src/compositions/LongformEpisode.tsx`
- `video/src/templates/` for reusable scene blocks
- `video/src/data/episode-manifest.ts` or JSON manifests generated by the factory

Scene inventory:

- cold open
- problem framing
- system diagram
- step walkthrough
- proof asset reveal
- CTA screen
- end card

Assembly rules:

- captions generated from TTS timestamps
- scene durations derived from narration timestamps with safe padding
- B-roll and UI mockups pulled from a deterministic asset manifest
- if a scene-specific asset is missing, fall back to a generic diagram scene instead of failing the whole episode

### 6. Thumbnail Generation

V1 should be template-first. The model can propose compositions and copy, but final assets should be rendered from house templates using the palette and composition rules in `channel-strategy.yaml`.

Generate:

- three text variants
- two layout families
- one safe backup

Auto-reject:

- logo spam
- robot faces
- duplicate title copy
- more than one focal object

### 7. Publish Payload Generation

This stage should be deterministic given the episode metadata and monetization registry.

Produced artifacts:

- title
- description
- pinned comment
- playlist selection
- tags
- disclosures
- UTM-decorated CTA links
- sponsor slot map if applicable

One important constraint from the existing strategy docs must stay enforced:

- one primary CTA per video

### 8. Upload Scheduling

Scheduling should only accept a fully green bundle:

- final video render
- selected thumbnail
- approved metadata bundle
- disclosure checks passed
- CTA link checks passed

The YouTube uploader should support:

- draft upload
- scheduled publish
- publish-now
- thumbnail upload
- metadata patch

### 9. Analytics Ingestion and Optimization

Nightly jobs should ingest:

- views
- CTR
- average percentage viewed
- average view duration
- watch time
- subscribers gained
- comments
- description click-through
- redirect click and lead events per CTA slot

That data feeds:

- topic score adjustments
- title and thumbnail experiment suggestions
- CTA offer routing changes
- sponsor eligibility checks

## Where Human Approval Can Disappear

These stages should run without a person in the loop once the templates are stable:

- niche discovery
- topic scoring and selection
- research packet generation
- first script draft
- shot list generation
- narration render
- video render retries
- thumbnail variant generation
- publish payload drafting
- post-publish analytics summaries

The system should still notify humans with artifacts and diffs, but not wait for approval by default.

## Hard QA Checkpoints That Should Stay

There should be one required pre-publish checkpoint and two conditional hard stops.

### Required hard gate: final publish bundle QA

One human should review:

- title + thumbnail + first description lines as a single promise
- audio quality and obvious timing jank
- factual and claim-risk items in `qa-claims.json`
- CTA relevance and disclosure copy
- whether the promised proof asset actually exists

This is the last cheap place to catch brand damage.

### Conditional hard stop: sponsor or affiliate compliance risk

Require explicit human approval when:

- a sponsor slot is attached
- claims rely on performance numbers or vendor comparisons
- disclosures were inserted or changed automatically

### Conditional hard stop: template drift

If automated QA detects a large deviation from house style, block the run and reroute to human review.

Examples:

- thumbnail violates brand palette or focal-object rule
- narration length exceeds target window
- scene count or pacing drifts far from template norms

## Automated QA

Automated checks should be opinionated and binary where possible.

| Check | Stage | Block condition |
| --- | --- | --- |
| Config validation | topic selection | invalid YAML or missing CTA mapping |
| Proof asset presence | topic selection | no proof asset declared |
| Citation coverage | research packet | unsupported factual claims |
| Hook clarity | script package | no workflow pain/result in the first 20 seconds |
| Timing window | voice/video | render exceeds target duration band |
| Brand palette and layout | thumbnail | off-brand template or multiple focal objects |
| Link validation | publish payload | broken CTA URLs or missing UTMs |
| Disclosure coverage | publish payload | affiliate/sponsor mention without disclosure |
| Publish bundle completeness | scheduling | missing video, thumbnail, or metadata |

## Failure Handling

The system should behave like a small factory, not a brittle demo.

Rules:

- every stage writes a manifest and status row
- every output is content-addressed or checksum-tracked
- transient failures retry automatically with exponential backoff
- deterministic failures mark the episode blocked with a specific fix reason
- reruns happen from the failed stage forward, not from the top

Typical failure paths:

| Failure | Handling |
| --- | --- |
| source fetch failed | retry up to 3 times, then keep prior cache and mark research stale |
| TTS generation failed | switch to backup voice or rerun only narration stage |
| Remotion render failed | rerun the render stage with lower parallelism and attach logs |
| thumbnail scorer returned low confidence | fall back to safe template variant |
| upload failed after asset creation | keep publish bundle frozen and retry uploader only |
| analytics API quota issue | defer sync and avoid mutating optimization scores on partial data |

## Monetization Instrumentation

Monetization should be wired in as data, not added manually in the description field at the last minute.

Required fields on every episode:

- `offer_id`
- `offer_type`
- `cta_slot`
- `utm_source`
- `utm_medium`
- `utm_campaign`
- `disclosure_type`
- `sponsor_category`

Recommended instrumentation path:

1. Generate all CTA URLs from `monetization-registry.yaml`.
2. Route links through a simple redirect layer like `/go/:offer/:video/:slot`.
3. Log redirect clicks with `video_id`, `offer_id`, and slot.
4. Ingest lead or purchase events back into `offer_events`.
5. Join revenue and lead quality to YouTube analytics for per-video ROI.

This matters because the factory is not just making videos. It is learning which workflows create revenue, not just views.

## Fastest V1 Implementation Order

1. Convert the repo to a lightweight `pnpm` workspace.
2. Add `packages/config` to validate and load the existing YAML strategy assets.
3. Add `apps/factory` with a SQLite-backed stage runner and CLI commands.
4. Generalize the existing Remotion package into manifest-driven long-form episode templates.
5. Implement publish bundle generation and local artifact output.
6. Add YouTube upload and analytics sync commands.
7. Add redirect-link instrumentation for CTA attribution.

## Opinionated Recommendation

Do not build a giant autonomous studio first.

Build a deterministic control plane that turns the existing strategy YAML into:

- one chosen topic
- one research packet
- one script package
- one render bundle
- one publish bundle
- one analytics feedback record

That is enough to run a serious faceless channel. The only hard human checkpoint should be the final pre-publish bundle review, plus sponsor/compliance exceptions. Everything else should be generated, validated, and rerun by the system.
