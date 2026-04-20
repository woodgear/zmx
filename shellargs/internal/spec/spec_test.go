package spec

import "testing"

func TestParseSpec(t *testing.T) {
	raw := `
name: repo-sync
summary: Sync repos

flag dry_run | short=n | long=dry-run | desc=Dry run only
option branch | short=b | default=main | desc=Branch name
option retry | short=r | type=int | default=3 | desc=Retry count
arg repo | required | desc=Repository name
arg path | repeatable | desc=Extra paths
`

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got.Name != "repo-sync" {
		t.Fatalf("unexpected name %q", got.Name)
	}
	if len(got.Fields) != 5 {
		t.Fatalf("unexpected field count %d", len(got.Fields))
	}
	if got.Fields[4].Kind != FieldArg || !got.Fields[4].Repeatable {
		t.Fatalf("expected last field to be repeatable arg: %#v", got.Fields[4])
	}
}

func TestRepeatableArgMustBeLast(t *testing.T) {
	raw := `
arg items | repeatable
arg tail
`

	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
