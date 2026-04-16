package agent

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TaskLogSummary is a one-line description of a task's log, produced by ListRecentTasks.
type TaskLogSummary struct {
	TaskID        string `json:"taskId"`
	AgentSlug     string `json:"agentSlug"`
	ToolCallCount int    `json:"toolCallCount"`
	FirstToolAt   int64  `json:"firstToolAt,omitempty"`
	LastToolAt    int64  `json:"lastToolAt,omitempty"`
	HasError      bool   `json:"hasError,omitempty"`
	SizeBytes     int64  `json:"sizeBytes"`
}

// TaskLogEntry is a single tool invocation parsed from a task's output.log.
// Fields mirror the record written by AgentLoop.logToolExecution in loop.go.
type TaskLogEntry struct {
	TaskID      string         `json:"task_id"`
	AgentSlug   string         `json:"agent_slug"`
	ToolName    string         `json:"tool_name"`
	Params      map[string]any `json:"params,omitempty"`
	Result      string         `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	StartedAt   int64          `json:"started_at,omitempty"`
	CompletedAt int64          `json:"completed_at,omitempty"`
}

// DefaultTaskLogRoot returns the directory where AgentLoop writes task logs.
// Exported so callers outside this package can match the on-disk layout.
func DefaultTaskLogRoot() string {
	return defaultTaskLogRoot()
}

// ListRecentTasks scans root for task directories and returns a summary for each,
// sorted newest-first by the modification time of each task's output.log.
// limit caps the number of returned tasks; pass 0 or negative for no cap.
func ListRecentTasks(root string, limit int) ([]TaskLogSummary, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []TaskLogSummary{}, nil
		}
		return nil, err
	}

	type taskWithMtime struct {
		summary TaskLogSummary
		mtime   int64
	}
	results := make([]taskWithMtime, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskID := entry.Name()
		logPath := filepath.Join(root, taskID, "output.log")
		info, err := os.Stat(logPath)
		if err != nil {
			continue
		}
		summary := summarizeTaskLog(logPath, taskID)
		summary.SizeBytes = info.Size()
		results = append(results, taskWithMtime{summary: summary, mtime: info.ModTime().UnixMilli()})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].mtime > results[j].mtime
	})

	out := make([]TaskLogSummary, 0, len(results))
	for i, r := range results {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, r.summary)
	}
	return out, nil
}

// ReadTaskLog parses root/{taskID}/output.log into TaskLogEntry records.
// Corrupt lines are silently skipped so one bad write does not poison the whole
// log. Callers would rather see 99 valid entries than a red error screen.
func ReadTaskLog(root, taskID string) ([]TaskLogEntry, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("taskID is required")
	}
	path := filepath.Join(root, taskID, "output.log")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []TaskLogEntry
	scanner := bufio.NewScanner(f)
	// Tool results can be large; give the scanner a generous buffer.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var entry TaskLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func summarizeTaskLog(path, taskID string) TaskLogSummary {
	s := TaskLogSummary{TaskID: taskID}
	f, err := os.Open(path)
	if err != nil {
		return s
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var entry TaskLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		s.ToolCallCount++
		if s.AgentSlug == "" && entry.AgentSlug != "" {
			s.AgentSlug = entry.AgentSlug
		}
		if entry.StartedAt > 0 {
			if s.FirstToolAt == 0 || entry.StartedAt < s.FirstToolAt {
				s.FirstToolAt = entry.StartedAt
			}
			if entry.StartedAt > s.LastToolAt {
				s.LastToolAt = entry.StartedAt
			}
		}
		if entry.Error != "" {
			s.HasError = true
		}
	}

	// Task IDs are "{slug}-{unixMillis}". Fall back to prefix parsing if JSONL
	// had no agent_slug (older logs).
	if s.AgentSlug == "" {
		if idx := strings.LastIndex(taskID, "-"); idx > 0 {
			s.AgentSlug = taskID[:idx]
		}
	}
	return s
}
