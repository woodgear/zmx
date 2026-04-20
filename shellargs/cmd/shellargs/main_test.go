package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestLoadSpecAutoTrimWrappedSpecDoc(t *testing.T) {
	raw, err := loadSpec("\n@@@\nname: demo\n@@@\n", "", "")
	if err != nil {
		t.Fatalf("loadSpec returned error: %v", err)
	}
	if raw != "name: demo\n" {
		t.Fatalf("unexpected spec after auto trim: %q", raw)
	}
}

func TestLoadSpecLeavesUnwrappedSpecAlone(t *testing.T) {
	raw, err := loadSpec("name: demo\narg action | required\n", "", "")
	if err != nil {
		t.Fatalf("loadSpec returned error: %v", err)
	}
	if raw != "name: demo\narg action | required\n" {
		t.Fatalf("unexpected unwrapped spec: %q", raw)
	}
}

func TestRunHelpWithWrappedSpecDoc(t *testing.T) {
	specDoc := "\n@@@\nname: demo\nsummary: demo help\n\narg action | required | desc=Action name\n@@@\n"
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"help", "--spec", specDoc}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("help output missing usage: %s", output)
	}
	if !strings.Contains(output, "ACTION") {
		t.Fatalf("help output missing positional arg: %s", output)
	}
}

func TestRunParseWithWrappedSpecDoc(t *testing.T) {
	specDoc := "\n@@@\nname: demo\nsummary: demo parse\n\narg action | required | desc=Action name\n@@@\n"
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"parse", "--spec", specDoc, "--", "hello"}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal parse output: %v\noutput=%s", err, stdout.String())
	}
	if got["action"] != "hello" {
		t.Fatalf("unexpected action value: %#v", got["action"])
	}
}

func TestRunIsHelpMatchesHelpFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"is-help", "--", "--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunIsHelpIgnoresLiteralAfterDoubleDash(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"is-help", "--", "--", "--help"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected non-help exit")
	}
	var exitErr *exitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exitError, got %T", err)
	}
	if exitErr.code != 1 {
		t.Fatalf("unexpected exit code %d", exitErr.code)
	}
}
