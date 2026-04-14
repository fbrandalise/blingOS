# YouTube Web UI Fast Path

Date: 2026-04-14
Owner: Eng

## Verdict

Do not re-platform this into React, Next, or a separate frontend app.

The fastest path to a working faceless YouTube business UI is to extend the existing WUPHF web shell:

- Go broker backend
- static web app in `web/index.html`
- existing `Studio` app panel
- existing starter template that seeds YouTube channels, specialists, and tasks
- existing broker surfaces for tasks, channels, memory, actions, scheduler, and skills

This repo is not greenfield. It already has the right product seam. The missing work is turning the Studio from a seeded control plane into a structured operating surface.

## What Already Exists

### Product scaffold

- `web/index.html` already includes a `Faceless AI Workflows` starter.
- The starter creates:
  - custom YouTube specialist agents
  - `yt-*` execution channels
  - seeded lane tasks
  - a kickoff packet for the CEO
- The sidebar already exposes a `Studio` app.

### Runtime scaffold

- `internal/team/broker.go` serves the web UI and proxies `/api/*` to the broker.
- `go build ./cmd/wuphf` passes.
- The web app already persists Studio state through broker `/memory`.
- The app already has Actions, Scheduler, Skills, Tasks, Requests, and Artifacts surfaces that can be reused for workflow visibility.

### YouTube-specific UI already shipped

- Studio hero, queue, architecture modules, monetization ladder, and integration backlog are already rendered.
- The starter seeds YouTube workspace defaults and queue state.
- The Studio panel reflects live `yt-*` channels and open tasks.

## Gaps

### 1. Studio state is still one coarse blob

Current state lives under:

- namespace: `youtube_factory`
- key: `studio_state`

That is enough for a single demo workspace, but not for a real operating system.

Problems:

- no per-workspace or per-channel partitioning
- no first-class records for content artifacts
- queue items are static seed objects, not linked entities
- `/memory` ignores the requested channel on read and returns the full shared store

### 2. Queue progression is UI theater

The queue can advance stages, but a run is not yet tied to:

- a topic packet
- a script brief
- a publish payload
- monetization defaults
- a retained artifact trail per run

### 3. Monetization is descriptive, not operational

The UI explains monetization well, but there is no editable registry for:

- offers
- CTA defaults
- pinned comment templates
- sponsor slots
- owned-audience destinations

### 4. External adapters are intentionally stubbed

Still missing:

- YouTube upload/scheduling adapter
- analytics ingest
- TTS vendor execution
- thumbnail/image generation
- render/compositing integration

That is fine for MVP. The mistake would be building those before the control plane works.

### 5. Frontend verification is thin

- no dedicated web smoke test for the YouTube starter + Studio flow
- `go test ./...` is not green in this environment

Observed test constraints:

- many tests fail because the sandbox blocks local listener binds
- there are also existing unrelated expectation failures in `cmd/wuphf`, `internal/config`, and `internal/tui`

## Fastest MVP Definition

The UI counts as working when a human can:

1. Launch WUPHF and choose the `Faceless AI Workflows` starter.
2. See `yt-*` channels, seeded tasks, and the Studio control plane.
3. Create and edit:
   - topic packets
   - script briefs
   - publish payloads
   - monetization offers
4. Move a run from backlog to ready-to-publish with durable state.
5. Generate a dry-run publish package with title, description, pinned comment, CTAs, and links.
6. Review an execution trail in the existing Artifacts surface.

No external credentials are required for MVP. Uploads and vendor generation stay dry-run.

## Recommended Delivery Order

### Milestone 1: Hardening the Studio control plane

Target: 1-2 days

Build:

- partition Studio state by workspace key instead of one global blob
- add a small Studio state schema:
  - workspace config
  - content runs
  - artifact references
  - monetization defaults
- keep using broker memory for speed
- wire explicit Studio sections for:
  - thesis and cadence
  - lane health
  - queue state
  - bottleneck state

Implementation note:

- fastest storage path is still broker memory, but split into multiple keys instead of one monolith
- example namespaces:
  - `youtube_factory/workspaces/<workspace-id>/config`
  - `youtube_factory/workspaces/<workspace-id>/runs`
  - `youtube_factory/workspaces/<workspace-id>/offers`

### Milestone 2: Content artifact layer

Target: 2-3 days

Build first-class structured objects for:

- topic packets
- script briefs
- publish payloads
- monetization offers

Each content run should reference those object IDs.

UI surfaces:

- topic backlog table with score, proof, search intent, monetization fit
- script brief drawer with hook, beats, CTA, repurposing notes, citations
- publish payload drawer with title, description, pinned comment, links, UTM tags
- offer registry editor

Fast-path storage:

- structured JSON collections in broker memory
- append action logs when records are created or updated so the Artifacts pane stays useful

### Milestone 3: MVP automation loop

Target: 2-3 days

Reuse existing broker surfaces instead of inventing a new automation engine:

- `Skills` for workflow definitions
- `Scheduler` for queued jobs
- `Actions` and `Artifacts` for history
- `Tasks` for human-visible execution ownership

MVP automations:

- generate topic packet draft
- generate script brief from approved topic packet
- generate publish payload from approved script brief + selected offer set
- move queue stage automatically when required artifacts exist

Keep all outputs dry-run and reversible.

### Milestone 4: Monetization surfaces

Target: 1-2 days

Add editable business surfaces for:

- affiliate offers
- lead magnets / owned-audience destinations
- CTA variants by content pillar
- sponsor-safe insertion slots
- default pinned comment blocks

The publish payload generator should consume these directly.

### Milestone 5: Adapter phase

Target: after MVP

Only after the above works, add adapters one by one:

- YouTube upload + schedule
- analytics ingest
- voice generation
- thumbnail generation
- render/export pipeline

Every adapter should sit behind:

- approval gates
- explicit cost controls
- reversible dry-run mode

## Suggested File-Level Approach

### Keep

- `web/index.html` as the main UI shell for this slice
- `internal/team/broker.go` as the persistence and workflow surface

### Add next

- small broker helpers for scoped Studio record read/write
- Studio artifact helpers in `web/index.html`
- a focused smoke test for:
  - starter selection
  - Studio render
  - record create/edit
  - queue persistence across reload

### Avoid for now

- React migration
- separate frontend build pipeline
- database introduction
- OAuth adapter work before the control plane is real

## Immediate Next Build

If we start now, the best first engineering slice is:

1. Split Studio memory into workspace config, runs, offers, and artifact collections.
2. Add Topic Packets and Script Briefs as editable structured records in the Studio.
3. Link queue cards to those records.
4. Generate a dry-run Publish Payload from a selected script brief + offer bundle.
5. Log each transition into Actions so the existing Artifacts view becomes the audit trail.

That gets us to a real control plane quickly without paying a rewrite tax.
