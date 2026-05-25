package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildActionGraph(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sourceA := filepath.Join(tmpDir, "a.actions.zsh")
	sourceB := filepath.Join(tmpDir, "b.actions.zsh")
	sourceC := filepath.Join(tmpDir, "c.actions.zsh")
	actionsDB := filepath.Join(tmpDir, "actions.db")
	output := filepath.Join(tmpDir, "graph", "actions.json")

	if err := os.WriteFile(sourceA, []byte(`function root-action() {
  leaf-action
  duplicate-action
  zmx-call.sh leaf-action
  zmx-call "$runtime_action"
  "$dynamic_action"
  eval "$cmd"
}

function leaf-action() {
  :
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceB, []byte(`function duplicate-action() {
  :
}

function broken-action() {
  export BAD@NAME=value
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceC, []byte(`function duplicate-action() {
  :
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	actionsDBText := strings.Join([]string{
		"root-action " + sourceA + " 1",
		"leaf-action " + sourceA + " 10",
		"duplicate-action " + sourceB + " 1",
		"broken-action " + sourceB + " 5",
		"duplicate-action " + sourceC + " 1",
		"",
	}, "\n")
	if err := os.WriteFile(actionsDB, []byte(actionsDBText), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Build(context.Background(), BuildConfig{
		Base:      tmpDir,
		ActionsDB: actionsDB,
		Output:    output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Nodes != 5 {
		t.Fatalf("expected 5 nodes, got %d", result.Nodes)
	}
	if result.DirectEdges != 2 {
		t.Fatalf("expected 2 direct edges, got %d", result.DirectEdges)
	}
	if result.AmbiguousCalls != 1 {
		t.Fatalf("expected 1 ambiguous call, got %d", result.AmbiguousCalls)
	}
	if result.DynamicCalls != 3 {
		t.Fatalf("expected 3 dynamic calls, got %d", result.DynamicCalls)
	}
	if result.ParseErrors != 1 {
		t.Fatalf("expected 1 parse error, got %d", result.ParseErrors)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var graph Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		t.Fatal(err)
	}

	root := nodeByName(t, graph, "root-action")
	leaf := nodeByName(t, graph, "leaf-action")
	if len(graph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(graph.Edges))
	}
	for _, edge := range graph.Edges {
		if edge.From != root.ID || edge.To != leaf.ID {
			t.Fatalf("unexpected edge: %#v", edge)
		}
	}
	if graph.Edges[0].Kind != edgeKindDirect && graph.Edges[1].Kind != edgeKindDirect {
		t.Fatalf("expected one direct edge: %#v", graph.Edges)
	}
	if graph.Edges[0].Kind != edgeKindZMXCall && graph.Edges[1].Kind != edgeKindZMXCall {
		t.Fatalf("expected one zmx-call edge: %#v", graph.Edges)
	}

	if len(graph.AmbiguousCalls) != 1 {
		t.Fatalf("expected 1 ambiguous call, got %d", len(graph.AmbiguousCalls))
	}
	ambiguous := graph.AmbiguousCalls[0]
	if ambiguous.From != root.ID || ambiguous.Name != "duplicate-action" || len(ambiguous.Candidates) != 2 {
		t.Fatalf("unexpected ambiguous call: %#v", ambiguous)
	}

	var duplicateNodes int
	for _, node := range graph.Nodes {
		if node.Name == "duplicate-action" {
			duplicateNodes++
			if !node.DuplicateName {
				t.Fatalf("duplicate node not marked: %#v", node)
			}
		}
	}
	if duplicateNodes != 2 {
		t.Fatalf("expected 2 duplicate-action nodes, got %d", duplicateNodes)
	}

	if len(graph.ParseErrors) != 1 {
		t.Fatalf("expected 1 parse error, got %d", len(graph.ParseErrors))
	}
	if !strings.Contains(graph.ParseErrors[0].Error, "invalid var name") {
		t.Fatalf("unexpected parse error: %#v", graph.ParseErrors[0])
	}
}

func TestWriteViewer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "graph", "actions.json")
	outputPath := filepath.Join(tmpDir, "graph", "index.html")
	graph := Graph{
		Version: 1,
		Nodes: []Node{{
			ID:          "demo@123",
			Name:        "demo",
			Source:      "/tmp/demo.zsh",
			RealSource:  "/tmp/demo.zsh",
			Line:        1,
			SourceGroup: "tmp",
		}},
		Stats: Stats{Nodes: 1},
	}
	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := WriteViewer(ViewConfig{
		Base:   tmpDir,
		Data:   dataPath,
		Output: outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != outputPath {
		t.Fatalf("viewer output = %q, want %q", result.Output, outputPath)
	}
	html, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	htmlText := string(html)
	if !strings.Contains(htmlText, "const GRAPH = JSON.parse(new TextDecoder().decode(graphBytes));") {
		t.Fatalf("viewer did not embed graph data")
	}
	if !strings.Contains(htmlText, "function stepSimulation") {
		t.Fatalf("viewer missing force simulation")
	}
	if !strings.Contains(htmlText, "Reheat") {
		t.Fatalf("viewer missing reheat control")
	}
	if !strings.Contains(htmlText, "selected: null") || !strings.Contains(htmlText, "radius: \"all\"") {
		t.Fatalf("viewer should default to the global graph")
	}
	if !strings.Contains(htmlText, "function assignComponents") || !strings.Contains(htmlText, "function seedGlobalLayout") {
		t.Fatalf("viewer missing global component layout")
	}
	if !strings.Contains(htmlText, "function seedIsolateGrid") {
		t.Fatalf("viewer missing isolated-node grid layout")
	}
	if !strings.Contains(htmlText, "class=\"no-selection\"") || !strings.Contains(htmlText, "ctx.fillText(label") {
		t.Fatalf("viewer missing global display refinements")
	}
	if !strings.Contains(htmlText, "nodeDrag") || !strings.Contains(htmlText, "node.fixed") {
		t.Fatalf("viewer missing node dragging")
	}
	if !strings.Contains(htmlText, "zmx graph") {
		t.Fatalf("viewer missing title")
	}
}

func nodeByName(t *testing.T, graph Graph, name string) Node {
	t.Helper()
	for _, node := range graph.Nodes {
		if node.Name == name {
			return node
		}
	}
	t.Fatalf("node %q not found", name)
	return Node{}
}
