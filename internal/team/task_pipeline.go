package team

import "strings"

type taskPipelineTemplate struct {
	ID             string
	OpenStage      string
	ActiveStage    string
	ReviewStage    string
	DoneStage      string
	ReviewRequired bool
}

var taskPipelineTemplates = map[string]taskPipelineTemplate{
	"feature":   {ID: "feature", OpenStage: "triage", ActiveStage: "implement", ReviewStage: "review", DoneStage: "ship", ReviewRequired: true},
	"bugfix":    {ID: "bugfix", OpenStage: "triage", ActiveStage: "fix", ReviewStage: "review", DoneStage: "verify", ReviewRequired: true},
	"research":  {ID: "research", OpenStage: "question", ActiveStage: "investigate", ReviewStage: "synthesize", DoneStage: "recommend"},
	"launch":    {ID: "launch", OpenStage: "brief", ActiveStage: "execute", ReviewStage: "review", DoneStage: "ship"},
	"incident":  {ID: "incident", OpenStage: "assess", ActiveStage: "mitigate", ReviewStage: "verify", DoneStage: "postmortem"},
	"follow_up": {ID: "follow_up", OpenStage: "triage", ActiveStage: "act", ReviewStage: "verify", DoneStage: "done"},
}

func inferTaskType(owner, title, details string) string {
	text := strings.ToLower(strings.TrimSpace(owner + " " + title + " " + details))
	switch {
	case containsAnyTaskFragment(text, "bug", "fix", "regression", "broken", "error", "panic", "crash"):
		return "bugfix"
	case containsAnyTaskFragment(text, "incident", "outage", "sev", "mitigate", "hotfix"):
		return "incident"
	case containsAnyTaskFragment(text, "launch", "campaign", "announce", "rollout", "go to market"):
		return "launch"
	case containsAnyTaskFragment(text, "research", "investigate", "evaluate", "compare", "analyze"):
		return "research"
	case containsAnyTaskFragment(text, "feature", "build", "implement", "ship", "signup", "flow"):
		return "feature"
	default:
		return "follow_up"
	}
}

func pipelineTemplate(taskType string) taskPipelineTemplate {
	if template, ok := taskPipelineTemplates[strings.TrimSpace(taskType)]; ok {
		return template
	}
	return taskPipelineTemplates["follow_up"]
}

func taskNeedsStructuredReview(task *teamTask) bool {
	if task == nil {
		return false
	}
	template := pipelineTemplate(task.TaskType)
	return template.ReviewRequired || task.ExecutionMode == "local_worktree"
}

func taskDefaultExecutionMode(owner, taskType string) string {
	switch strings.TrimSpace(strings.ToLower(owner)) {
	case "fe", "be", "ai":
		if taskType == "feature" || taskType == "bugfix" || taskType == "incident" {
			return "local_worktree"
		}
	}
	return "office"
}

func taskStageForStatus(task *teamTask) string {
	template := pipelineTemplate(task.TaskType)
	switch strings.TrimSpace(task.Status) {
	case "in_progress":
		return template.ActiveStage
	case "review":
		return template.ReviewStage
	case "done":
		return template.DoneStage
	default:
		return template.OpenStage
	}
}

func normalizeTaskPlan(task *teamTask) {
	if task == nil {
		return
	}
	if strings.TrimSpace(task.TaskType) == "" {
		task.TaskType = inferTaskType(task.Owner, task.Title, task.Details)
	}
	if strings.TrimSpace(task.PipelineID) == "" {
		task.PipelineID = task.TaskType
	}
	if strings.TrimSpace(task.ExecutionMode) == "" {
		task.ExecutionMode = taskDefaultExecutionMode(task.Owner, task.TaskType)
	}
	if strings.TrimSpace(task.ReviewState) == "" {
		if taskNeedsStructuredReview(task) {
			task.ReviewState = "pending_review"
		} else {
			task.ReviewState = "not_required"
		}
	}
	if strings.TrimSpace(task.Status) == "review" {
		task.ReviewState = "ready_for_review"
	}
	if strings.TrimSpace(task.Status) == "done" && task.ReviewState == "pending_review" {
		task.ReviewState = "approved"
	}
	task.PipelineStage = taskStageForStatus(task)
}

func containsAnyTaskFragment(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
