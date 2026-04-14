package team

import "testing"

func TestInferTaskTypeTreatsAuditWorkAsResearch(t *testing.T) {
	got := inferTaskType("eng", "Audit repo and define the fastest path to a working web UI", "Working plan for the faceless YouTube business build.")
	if got != "research" {
		t.Fatalf("inferTaskType returned %q, want research", got)
	}
}

func TestTaskDefaultExecutionModeTreatsEngineeringFeatureWorkAsLocalWorktree(t *testing.T) {
	if got := taskDefaultExecutionMode("eng", "feature"); got != "local_worktree" {
		t.Fatalf("taskDefaultExecutionMode returned %q, want local_worktree", got)
	}
	if got := taskDefaultExecutionMode("eng", "bugfix"); got != "local_worktree" {
		t.Fatalf("taskDefaultExecutionMode returned %q, want local_worktree", got)
	}
	if got := taskDefaultExecutionMode("gtm", "launch"); got != "office" {
		t.Fatalf("taskDefaultExecutionMode returned %q, want office", got)
	}
}
