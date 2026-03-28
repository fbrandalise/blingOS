# WUPHF Autonomy Acceptance Criteria

This document is the implementation contract for the current autonomy push. When the work is complete, WUPHF should satisfy every item here.

## 1. Visible Signal -> Decision -> Action Ledger

- Every major external or office trigger is persisted as an `office_signal`.
  - Includes at least: Nex insights, Nex notifications, scheduler/watchdog wakes, and explicit human directives.
- Every policy outcome is persisted as an `office_decision`.
  - Includes at least: summarize, create/update task, ask human, bridge channel, wake specialist, or ignore.
- Every resulting side effect is persisted as an `office_action`.
  - Actions point back to the originating decision and signal when available.
- Watchdog escalations are persisted as `watchdog_alert` records.
- The office UI exposes this visibly in `Insights` and related task metadata, not just as hidden state.

## 2. Self-Starting Owned Work

- When policy or the CEO creates a task with an owner, the owner is woken automatically.
- The wake carries concrete task context, not just a generic “check the channel” nudge.
- Task state is the source of truth for owned work.
- Task lifecycle is visible and must support at least:
  - `open`
  - `in_progress`
  - `review`
  - `blocked`
  - `done`
- Human-facing updates, recommendations, and completion reports use direct human-facing messages in the main chat.

## 3. Pipeline and Execution Metadata

- Every task carries structured metadata:
  - `task_type`
  - `pipeline_id`
  - `pipeline_stage`
  - `execution_mode`
  - `review_state`
  - `source_signal_id`
  - `source_decision_id`
- WUPHF supports pipeline defaults for:
  - `feature`
  - `bugfix`
  - `research`
  - `launch`
  - `incident`
  - `follow_up`
- Pipeline state is visible in the office UI, especially in `Tasks` and `Calendar`.
- Code-heavy tasks can be marked for local execution hygiene through `execution_mode`, even when planning and coordination remain in the office.

## 4. Watchdogs and Stalled Work

- Watchdogs detect and act on:
  - unclaimed or stagnant tasks
  - tasks stuck in progress too long
  - blocking human decisions waiting too long
  - repeated duplicate signals inside a suppression window
- Watchdog output is visible and concise.
- Watchdogs prefer:
  1. remind owner
  2. escalate to CEO
  3. interrupt the human only when needed
- Calendar shows watchdog reminders and follow-up timing.

## 5. Human-Visible Control Surface

- Important human decisions still interrupt in main chat; they are not buried in `Requests`.
- `Requests` is for backlog/history/non-blocking asks.
- `Insights` answers:
  - what changed
  - why it mattered
  - what decision was taken
  - who owns the follow-up
- `Tasks` answers:
  - who owns this
  - what stage it is in
  - whether review is required
  - what triggered it
- Cross-channel bridges visibly show:
  - source channel
  - target channel
  - whether the CEO bridged the context

## 6. Required Validation

- Nex insight -> signal -> decision -> task/request -> visible action trail
- Owned task creation immediately wakes the owner
- Blocking decision still pauses the office and lands in main chat
- Out-of-domain specialists remain suppressed unless tagged or assigned
- Pipeline metadata appears on tasks and calendar events
- Watchdog escalation produces visible office state without channel spam
