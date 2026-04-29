package shellargs

import (
	"strings"
	"testing"
)

func TestZshCompletionScriptFromSpec(t *testing.T) {
	sp, err := ParseSpec(`
name: repo-sync
summary: Sync repos

flag dry_run | short=n | long=dry-run | desc=Dry run only
option branch | short=b | default=main | desc=Branch name
option config | type=file | desc=Config path
arg repo | required | desc=Repository name
arg path | repeatable | desc=Extra paths
`)
	if err != nil {
		t.Fatalf("ParseSpec returned error: %v", err)
	}

	script, err := ZshCompletionScript(sp, "repo-sync")
	if err != nil {
		t.Fatalf("ZshCompletionScript returned error: %v", err)
	}

	for _, want := range []string{
		"#compdef repo-sync",
		"'-n[Dry run only]'",
		"'--dry-run[Dry run only]'",
		"'--branch[Branch name]:BRANCH:'",
		"'--config[Config path]:CONFIG:_files'",
		"'1:REPO:'",
		"'*:PATH:'",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("completion script missing %q:\n%s", want, script)
		}
	}
}

func TestParseSpecTrimsWrappedDoc(t *testing.T) {
	sp, err := ParseSpec(`
@@@
name: demo
arg target | required
@@@
`)
	if err != nil {
		t.Fatalf("ParseSpec returned error: %v", err)
	}
	if sp.Name != "demo" {
		t.Fatalf("unexpected spec name %q", sp.Name)
	}
}
