# YouTube Business E2E Eval

Date: 2026-04-14
Scenario: Post in `#general` asking the team to build a faceless YouTube channel business with automated content creation and monetization, from the Web UI, starting from a fresh worktree and reset office state.

## Run 1

### Issue 1: stale CEO turn after delegation
- Symptom: CEO delegated to `@eng` and `@gtm`, but stayed active in the same turn instead of stopping and waiting for pushed specialist completions. Specialist replies landed in the broker, but the office stalled before CEO synthesis.
- Evidence:
  - `~/.wuphf/team/broker-state.json` showed specialist replies without a follow-up CEO message.
  - `~/.wuphf/logs/headless-codex-ceo.log` showed the runtime relying on stale-turn cancellation to recover.
- Fix:
  - Tightened CEO and specialist prompts to explicitly end the turn after delegation, completion, handoff, or a blocking human question.
  - Added the same stop-after-reply instruction to queued work packets and task notifications.
- Status: fixed in code, pending rerun validation.

### Issue 2: live-output panes leaked opaque `item.completed` noise
- Symptom: specialist DM live-output panes showed repeated `item.completed` cards with no human-usable content.
- Evidence:
  - Web UI live-output stream showed `item.completed` rows instead of actionable summaries.
- Fix:
  - Suppressed unknown `item.completed` events in the live-output renderer after recognized tool/message cases are handled.
- Status: fixed in code, pending rerun validation.

### Issue 3: Nex context queries failed despite health showing connected
- Symptom: agents attempted `query_context`, but live output showed session/auth expiry errors.
- Evidence:
  - Specialist and CEO live-output panes showed Nex auth/session-expired failures.
  - `/health` still reported `nex_connected: true`.
- Fix:
  - No code fix yet. This appears to be environment/auth state, not a deterministic repo bug.
- Status: open.

### Issue 4: automation did not create tasks, channels, or skills in the first pass
- Symptom: the team produced planning replies, but no task records, new channels, or skills were created for the business build.
- Evidence:
  - Broker state remained at `task_count = 0`, `channel_count = 1`.
- Fix:
  - No code fix yet. First rerun will confirm whether the stale-turn fix unblocks the expected CEO follow-through.
- Status: open.

## Run 2

### Result: Codex now creates durable task state
- Symptom change:
  - After the leadership prompt update, the CEO created a top-level project task immediately instead of leaving the office in pure discussion state.
- Evidence:
  - `task-4` was created from the initial CEO reply and showed up in the UI and `/tasks`.
- Fix:
  - Prompt change only. No broker change required for this part.
- Status: improved, but still incomplete.

### Issue 5: CEO could tag a non-existent agent and the broker accepted it
- Symptom:
  - The CEO tagged `studio` in-channel, but `studio` did not exist in `/office-members`.
  - The broker silently accepted the tag, creating a false appearance of extra staffing.
- Evidence:
  - Message `msg-3` included `studio` in `tagged`.
  - `/office-members` still listed only `ceo`, `eng`, and `gtm`.
- Fix:
  - Tightened broker message validation so explicit tagged slugs must be real office members (except `you`/`human`/`system`).
  - Thread auto-tagging now filters out stale/non-member participants instead of re-tagging ghosts.
- Status: fixed in code, rerun validated no dead-agent tag leakage.

### Issue 6: non-general channel composer kept the wrong aria-label
- Symptom:
  - In `#lab`, the composer placeholder updated correctly, but the aria-label still read `Message general channel`.
  - This is an accessibility bug and it broke automation targeting.
- Evidence:
  - DOM inspection in `#lab` showed placeholder `Message #lab — type / for commands, @ to mention` with aria-label `Message general channel`.
- Fix:
  - Updated the web UI to synchronize `aria-label` with composer context for normal channels and direct messages.
- Status: fixed in code.

## Run 3

### Result: Codex first-turn behavior improved materially
- Evidence:
  - Fresh rerun from the Web UI produced:
    - immediate CEO reply
    - explicit top-level task creation by CEO
    - specialist routing without dead-agent tags
- Current state:
  - Codex now reliably creates at least one durable task for the initiative.
  - Codex still did **not** create a dedicated execution channel or propose any skills in the validated reruns.
  - The business-build loop remains better coordinated, but it still stops short of a full autonomous “build and run from scratch” operating setup.
- Status: partially fixed; still open at the product-behavior level.

## Approval Judgement

- No approval requests were raised in the validated Codex reruns.
- This was acceptable for the work that actually happened:
  - internal planning
  - routing
  - task creation
  - channel setup intent
- Approvals that **should** still be required later, once the workflow reaches them:
  - spending money on vendors or subscriptions
  - creating or linking real external publishing/monetization accounts
  - publishing content to a real YouTube channel
  - accepting legal/commercial commitments such as sponsorship terms
- Net:
  - no unnecessary approvals observed
  - but the eval never progressed far enough to exercise the approvals that should exist for real-world execution

## Claude Code Smoke

Goal: verify Claude Code works for message exchange across normal channels and direct agent DMs after the Codex-focused fixes.

### Verified working
- `#general`
  - human: `Claude smoke test in #general. Reply with GENERAL-OK and nothing else.`
  - reply: `GENERAL-OK`
- `#lab`
  - human: `Claude smoke test in #lab. Reply with LAB-OK and nothing else.`
  - reply: `LAB-OK`
- DM with CEO
  - human: `Claude DM smoke to CEO. Reply with CEO-DM-OK and nothing else.`
  - reply: `CEO-DM-OK`
- DM with Eng
  - human: `Claude DM smoke to ENG. Reply with ENG-DM-OK and nothing else.`
  - reply: `ENG-DM-OK`

### Notes
- DM channels are persisted as deterministic pair slugs such as `ceo__human` and `eng__human`, even though the UI route is `#/dm/<agent>`.
- In the Eng DM, the CEO also posted a follow-up confirmation after Eng replied. This did not break the DM flow, but it is extra chatter worth noting for future DM-polish work.
