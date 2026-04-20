package engine

import (
	"bytes"
	"strings"
	"testing"

	"shellargs/internal/spec"
)

func TestParseToJSONMap(t *testing.T) {
	sp, err := spec.Parse(`
name: repo-sync
summary: Sync repos

flag dry_run | short=n | long=dry-run | desc=Dry run only
option branch | short=b | default=main | desc=Branch name
option retry | short=r | type=int | default=3 | desc=Retry count
arg repo | required | desc=Repository name
arg path | repeatable | desc=Extra paths
`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	engine, err := New(sp)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	var out bytes.Buffer
	result, err := engine.Parse(ParseOptions{
		Args:     []string{"--branch", "dev", "--retry", "5", "--dry-run", "demo", "p1", "p2"},
		AutoHelp: true,
		Stdout:   &out,
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result.Values["branch"] != "dev" {
		t.Fatalf("unexpected branch %#v", result.Values["branch"])
	}
	if result.Values["retry"] != 5 {
		t.Fatalf("unexpected retry %#v", result.Values["retry"])
	}
	if result.Values["repo"] != "demo" {
		t.Fatalf("unexpected repo %#v", result.Values["repo"])
	}

	paths, ok := result.Values["path"].([]string)
	if !ok {
		t.Fatalf("unexpected path type %T", result.Values["path"])
	}
	if len(paths) != 2 || paths[0] != "p1" || paths[1] != "p2" {
		t.Fatalf("unexpected paths %#v", paths)
	}
}

func TestWriteHelp(t *testing.T) {
	sp, err := spec.Parse(`
name: repo-sync
summary: Sync repos
option branch | short=b | default=main | desc=Branch name
arg repo | required | desc=Repository name
`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	engine, err := New(sp)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	var out bytes.Buffer
	if err := engine.WriteHelp(&out); err != nil {
		t.Fatalf("WriteHelp returned error: %v", err)
	}

	help := out.String()
	if !strings.Contains(help, "--branch") {
		t.Fatalf("help did not contain branch option: %s", help)
	}
	if !strings.Contains(help, "repo") {
		t.Fatalf("help did not contain positional arg: %s", help)
	}
}

func TestBashCompletionScript(t *testing.T) {
	script, err := BashCompletionScript(BashCompletionOptions{
		Runner:     "shellargs",
		Program:    "repo-sync",
		SpecBase64: "c3BlYw==",
		Shell:      "bash",
	})
	if err != nil {
		t.Fatalf("BashCompletionScript returned error: %v", err)
	}
	if !strings.Contains(script, "GO_FLAGS_COMPLETION=1") {
		t.Fatalf("missing GO_FLAGS_COMPLETION hook: %s", script)
	}
	if !strings.Contains(script, "repo-sync") {
		t.Fatalf("missing program name: %s", script)
	}
}
