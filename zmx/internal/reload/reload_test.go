package reload

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReload(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	actionsDir := filepath.Join(tmpDir, "actions")
	callTarget := filepath.Join(tmpDir, "zmx-call.sh")

	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(callTarget, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(actionsDir, "demo.zsh"), []byte(`function demo-action() {
  echo demo
}

function _private-action() {
  echo private
}

function demo_second() {
  echo no
}

function another-action() {
  echo another
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout strings.Builder
	result, err := Run(context.Background(), Config{
		Base:        baseDir,
		ActionsPath: actionsDir,
		CallTarget:  callTarget,
		Stdout:      &stdout,
		Stderr:      &stdout,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Actions != 3 {
		t.Fatalf("expected 3 actions, got %d", result.Actions)
	}
	if result.Files != 1 {
		t.Fatalf("expected 1 source file, got %d", result.Files)
	}

	actionsDB, err := os.ReadFile(filepath.Join(baseDir, "actions.db"))
	if err != nil {
		t.Fatal(err)
	}
	actionsText := string(actionsDB)
	if !strings.Contains(actionsText, "demo-action") {
		t.Fatalf("actions.db missing demo-action:\n%s", actionsText)
	}
	if !strings.Contains(actionsText, "another-action") {
		t.Fatalf("actions.db missing another-action:\n%s", actionsText)
	}
	if strings.Contains(actionsText, "_private-action") {
		t.Fatalf("actions.db should not include private action:\n%s", actionsText)
	}
	if strings.Contains(actionsText, "demo_second") {
		t.Fatalf("actions.db should preserve current no-underscore behavior:\n%s", actionsText)
	}

	importFile, err := os.ReadFile(filepath.Join(baseDir, "import.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(importFile), "source ") {
		t.Fatalf("import.sh missing source lines:\n%s", string(importFile))
	}

	toolsScript, err := os.ReadFile(filepath.Join(baseDir, "tools", "zmx-call"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(toolsScript), callTarget) {
		t.Fatalf("generated tool missing call target:\n%s", string(toolsScript))
	}

	md5DirEntries, err := os.ReadDir(filepath.Join(baseDir, "md5"))
	if err != nil {
		t.Fatal(err)
	}
	if len(md5DirEntries) != 1 {
		t.Fatalf("expected 1 md5 file, got %d", len(md5DirEntries))
	}

	recordFile, err := os.ReadFile(filepath.Join(baseDir, "record"))
	if err != nil {
		t.Fatal(err)
	}
	recordText := string(recordFile)
	if !strings.Contains(recordText, "index over") {
		t.Fatalf("record missing index step:\n%s", recordText)
	}
	if !strings.Contains(recordText, "build over") {
		t.Fatalf("record missing build step:\n%s", recordText)
	}
	if !strings.Contains(recordText, "gen-md5 over") {
		t.Fatalf("record missing gen-md5 step:\n%s", recordText)
	}
}

func TestSplitAndSortPaths(t *testing.T) {
	t.Parallel()

	paths := splitAndSortPaths(" /b::/a:/b:/c ")
	got := strings.Join(paths, ",")
	want := "/a,/b,/c"
	if got != want {
		t.Fatalf("splitAndSortPaths() = %q, want %q", got, want)
	}
}
