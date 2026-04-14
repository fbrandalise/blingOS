# YouTube Factory GTM Kit

Date: 2026-04-14
Owner: GTM

This folder turns the faceless-channel strategy into day-one operating assets the repo can use.

## Included Assets

- `channel-operating-manual.md`
  - niche rationale
  - audience thesis
  - content pillars
  - packaging rubric
  - publishing cadence
  - launch KPIs
- `launch-collateral.md`
  - channel description
  - banner and about copy
  - pinned comment and description templates
  - lead magnet copy
  - audit offer copy
  - sponsor one-sheet starter
- `launch-week-operating-brief.md`
  - locked week-one wedge and audience promise
  - first 10 upload slate
  - CTA ladder and launch-week monetization order
  - top 3 factory KPIs
  - missing business inputs the generator still needs
- `wuphf-operator-channel-package.md`
  - product-led channel recommendation for the live WUPHF GTM motion
  - wedge, positioning, monetization ladder, KPI targets, and first 30-video slate
- `wuphf-operator-channel-pack.yaml`
  - machine-readable WUPHF operator channel seed using the existing channel-pack schema
  - brand, audience, CTA, playlist, QA, and approval defaults for the product-led wedge
- `channel-strategy.yaml`
  - machine-readable brand, ICP, content pillar, cadence, SEO, and distribution defaults
- `default-channel-pack.yaml`
  - machine-readable default channel seed for the Studio and publish generator
  - brand, render, CTA, playlist, QA, and approval defaults in one pack
- `content-backlog.yaml`
  - first 30-video backlog with priorities, CTAs, search intent, and thumbnail angles
- `episode-launch-packets/vid_01-inbox-operator.yaml`
  - publish-ready dry-run launch packet for the flagship inbox-operator episode
  - title, thumbnail brief, CTA map, description, pinned comment, and QA checklist
- `episode-launch-packets/wuphf-vid_01-steer-mid-flight.yaml`
  - publish-ready dry-run launch packet for the WUPHF flagship see-and-steer episode
  - title, thumbnail brief, CTA map, description, pinned comment, chapters, and QA checklist
- `seo-distribution-playbook.md`
  - search-intent model
  - playlist and metadata rules
  - repurposing workflow
  - per-upload distribution checklist
- `monetization-registry.yaml`
  - structured ladder of offers
  - CTA routing rules
  - approved affiliate categories
  - digital products and service offers
  - UTM defaults and disclosure copy
- `partner-matrix.yaml`
  - workflow-to-affiliate and sponsor mapping
  - disclosure defaults
  - commercialization guardrails
- `automation-sops.yaml`
  - stage-by-stage operating procedures
  - pass gates, stop conditions, and handoff artifacts
- `factory-system-design.md`
  - end-to-end automation architecture
  - stack recommendation for this repo
  - QA boundaries, failure handling, and monetization instrumentation

## Strategic Call

- Channel thesis: `AI Back Office for Small Teams`
- Launch wedge: founder-led service businesses and agencies with 2-20 people
- Brand call: `Back Office AI`
- Format bias: long-form first, cutdowns second
- Revenue model: lead magnet -> affiliate -> low-ticket templates -> audit/sprint -> sponsors -> ads

## Live GTM Note

The assets above package the generic faceless-channel business. The additive `wuphf-operator-channel-package.md` captures the sharper product-led recommendation for the current WUPHF-vs-Paperclip launch window: lead with the WUPHF operator channel first, keep `Back Office AI` as a later expansion lane.

That recommendation is now machine-readable too in `wuphf-operator-channel-pack.yaml` and `episode-launch-packets/wuphf-vid_01-steer-mid-flight.yaml`, so the same generator path can run against the WUPHF wedge without reworking the schema.

## Why This Folder Exists

The repo already had a sound business-plan direction. What it lacked was reusable GTM collateral that can be handed to the Studio, the publish payload generator, and future sales/sponsor workflows without rethinking the business each run.

## New Operating Assets

- The brand system now includes multiple naming paths, visual direction, and packaging patterns in `channel-strategy.yaml`.
- The default channel seed now lives in `default-channel-pack.yaml`, so Studio state and future automation can start from a concrete bundle instead of stitching together prose docs at runtime.
- The WUPHF operator wedge now has its own publish-ready seed in `wuphf-operator-channel-pack.yaml`, using the same schema as the default channel pack.
- The backlog is now structured for automation in `content-backlog.yaml`, not just listed in prose.
- The first flagship launch packet now lives in `episode-launch-packets/vid_01-inbox-operator.yaml`, giving the factory a dry-run publish payload to wire up immediately.
- The WUPHF flagship packet now lives in `episode-launch-packets/wuphf-vid_01-steer-mid-flight.yaml`, so the product-led launch path can run in parallel with the faceless channel path.
- Commercial routing is now explicit in `partner-matrix.yaml`, so sponsor and affiliate choices follow the workflow instead of guesswork.
- Automation-ready SOPs now live in `automation-sops.yaml`, giving the control plane stage gates instead of vague GTM advice.
