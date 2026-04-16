package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListRecentTasks_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, err := ListRecentTasks(dir, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(got))
	}
}

func TestListRecentTasks_OrdersByMtimeDesc(t *testing.T) {
	dir := t.TempDir()
	mustWriteLog(t, filepath.Join(dir, "eng-100", "output.log"), `{"tool_name":"grep_search"}`+"\n")
	time.Sleep(10 * time.Millisecond)
	mustWriteLog(t, filepath.Join(dir, "ceo-200", "output.log"), `{"tool_name":"send_message"}`+"\n")

	got, err := ListRecentTasks(dir, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	if got[0].TaskID != "ceo-200" {
		t.Fatalf("expected ceo-200 first (newest), got %s", got[0].TaskID)
	}
	if got[0].AgentSlug != "ceo" {
		t.Fatalf("expected agent slug ceo, got %s", got[0].AgentSlug)
	}
	if got[0].ToolCallCount != 1 {
		t.Fatalf("expected 1 tool call, got %d", got[0].ToolCallCount)
	}
}

func TestReadTaskLog_ParsesJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eng-100", "output.log")
	mustWriteLog(t, path, `{"tool_name":"grep_search","agent_slug":"eng","started_at":1700000000000,"params":{"pattern":"svg"}}`+"\n"+
		`{"tool_name":"write_file","agent_slug":"eng","started_at":1700000001000,"params":{"path":"/tmp/x"}}`+"\n")

	entries, err := ReadTaskLog(dir, "eng-100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ToolName != "grep_search" {
		t.Fatalf("first entry: want grep_search, got %s", entries[0].ToolName)
	}
	if entries[1].ToolName != "write_file" {
		t.Fatalf("second entry: want write_file, got %s", entries[1].ToolName)
	}
}

func TestReadTaskLog_SkipsCorruptLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eng-100", "output.log")
	mustWriteLog(t, path, `{"tool_name":"grep_search"}`+"\n"+
		"this is not json\n"+
		`{"tool_name":"write_file"}`+"\n")

	entries, err := ReadTaskLog(dir, "eng-100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 valid entries (corrupt line skipped), got %d", len(entries))
	}
}

func mustWriteLog(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
