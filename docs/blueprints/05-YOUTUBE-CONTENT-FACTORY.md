# YouTube Content Factory Architecture

Date: 2026-04-14
Owner: Eng

## Current UI Audit

The web UI already had real product intent for this use case:

- onboarding ships a `Faceless AI Workflows` starter
- the starter seeds specialist roles, YouTube execution channels, and initial tasks
- the sidebar already exposes a `Studio` app
- the design system already includes a YouTube/control-plane visual language

The gap was that the Studio was still mostly theater:

- the sidebar linked to `Studio`, but `renderAppContent()` never rendered that panel
- Studio queue state lived only in `localStorage`
- there was no durable broker-backed workspace state for the YouTube business
- the UI hinted at research/script/voice/edit/thumbnail/publish stages, but those were not tied to the real office state

## Target System

The end-to-end system should be split into a stable control plane plus replaceable execution adapters.

### Control plane

- channel thesis and audience
- publishing cadence and content mix
- topic backlog and scorecards
- queue state for each content run
- monetization defaults: offers, CTA blocks, pinned comments, sponsor slots
- approval points for anything that touches real accounts or spend

### Build now

- broker-backed Studio state and queue
- research packet schema: topic, proof, search intent, monetization fit, source links
- script brief schema: hook, beats, CTA, repurposing notes, citations
- publish payload template: title, description, pinned comment, links, UTM tags, CTA copy
- monetization registry: affiliate offers, lead magnets, digital products, sponsor-safe slots

### Stub first

- TTS vendor execution
- thumbnail/image generation vendor execution
- video compositing/render farm
- YouTube upload + scheduling adapter
- YouTube Analytics ingest
- sponsor CRM or payment workflows

The rule is simple: business logic first, vendor adapters second.

## Recommended Delivery Order

1. Ship the Studio as a real control plane backed by broker memory.
2. Persist topic packets, script briefs, and monetization defaults as first-class records/artifacts.
3. Add dry-run publish payload generation with approvals.
4. Add external adapters one by one: voice, thumbnails, video render, YouTube upload, analytics ingest.

## First Slice Implemented

This repo now has the first concrete product slice:

- the `Studio` sidebar app now renders instead of opening a blank panel
- Studio state is persisted through broker `/memory` in live mode
- the faceless YouTube starter now seeds Studio state during onboarding
- the Studio panel shows:
  - workspace thesis and cadence
  - build-now vs stub architecture modules
  - live YouTube lanes and open lane tasks
  - durable queue state for runs in flight
  - monetization ladder
  - adapter backlog for the still-stubbed integrations

## Next Slice

The next implementation should add durable records for:

- topic packets
- script briefs
- publish payloads
- monetization offers

Once those records exist, agent work can stop being freeform chat and start producing reusable factory artifacts.
