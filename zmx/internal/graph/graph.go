package graph

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mvdan.cc/sh/v3/syntax"
)

const (
	edgeKindDirect  = "direct"
	edgeKindZMXCall = "zmx-call"

	dynamicReasonCommandWord = "dynamic command word"
	dynamicReasonEval        = "eval"
	dynamicReasonZMXCall     = "dynamic zmx-call target"
)

var functionPattern = regexp.MustCompile(`^function\s+\S+`)

type BuildConfig struct {
	Base      string
	ActionsDB string
	Output    string
	Stdout    io.Writer
}

type BuildResult struct {
	Nodes            int
	DirectEdges      int
	AmbiguousCalls   int
	DynamicCalls     int
	ParseErrors      int
	ExternalCommands int
	Output           string
}

type Graph struct {
	Version          int               `json:"version"`
	GeneratedAt      string            `json:"generated_at"`
	ActionsDB        string            `json:"actions_db"`
	Nodes            []Node            `json:"nodes"`
	Edges            []Edge            `json:"edges"`
	AmbiguousCalls   []AmbiguousCall   `json:"ambiguous_calls,omitempty"`
	DynamicCalls     []DynamicCall     `json:"dynamic_calls,omitempty"`
	ParseErrors      []ParseError      `json:"parse_errors,omitempty"`
	ExternalCommands []ExternalCommand `json:"external_commands,omitempty"`
	Stats            Stats             `json:"stats"`
}

type Node struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Source        string `json:"source"`
	RealSource    string `json:"real_source"`
	Line          int    `json:"line"`
	SourceGroup   string `json:"source_group"`
	DuplicateName bool   `json:"duplicate_name"`
	ParseError    bool   `json:"parse_error"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Name string `json:"name"`
	Line int    `json:"line"`
	Kind string `json:"kind"`
}

type AmbiguousCall struct {
	From       string   `json:"from"`
	Name       string   `json:"name"`
	Line       int      `json:"line"`
	Candidates []string `json:"candidates"`
}

type DynamicCall struct {
	From   string `json:"from"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
	Reason string `json:"reason"`
}

type ParseError struct {
	NodeID string `json:"node_id"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Line   int    `json:"line"`
	Error  string `json:"error"`
}

type ExternalCommand struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Stats struct {
	Nodes                  int `json:"nodes"`
	Edges                  int `json:"edges"`
	AmbiguousCalls         int `json:"ambiguous_calls"`
	DynamicCalls           int `json:"dynamic_calls"`
	ParseErrors            int `json:"parse_errors"`
	ExternalCommandNames   int `json:"external_command_names"`
	ExternalCommandUsages  int `json:"external_command_usages"`
	ActionsWithParseErrors int `json:"actions_with_parse_errors"`
}

type actionRecord struct {
	Name       string
	SourcePath string
	RealSource string
	Line       int
	ID         string
}

type callRecord struct {
	From string
	Name string
	Line int
	Kind string
}

func Build(ctx context.Context, cfg BuildConfig) (BuildResult, error) {
	if cfg.Base == "" {
		return BuildResult{}, fmt.Errorf("zmx graph build: --base is required")
	}
	if cfg.ActionsDB == "" {
		cfg.ActionsDB = filepath.Join(cfg.Base, "actions.db")
	}
	if cfg.Output == "" {
		cfg.Output = filepath.Join(cfg.Base, "graph", "actions.json")
	}
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	records, err := readActionRecords(cfg.ActionsDB)
	if err != nil {
		return BuildResult{}, fmt.Errorf("zmx graph build: read actions db: %w", err)
	}
	if len(records) == 0 {
		return BuildResult{}, fmt.Errorf("zmx graph build: no actions in %s", cfg.ActionsDB)
	}

	nodes, recordsByID, nameIndex := buildNodes(records)
	calls, dynamicCalls, parseErrors, err := collectCalls(ctx, recordsByID)
	if err != nil {
		return BuildResult{}, fmt.Errorf("zmx graph build: collect calls: %w", err)
	}

	var edges []Edge
	var ambiguousCalls []AmbiguousCall
	externalCounts := map[string]int{}
	for _, call := range calls {
		candidates := nameIndex[call.Name]
		switch len(candidates) {
		case 0:
			externalCounts[call.Name]++
		case 1:
			edges = append(edges, Edge{
				From: call.From,
				To:   candidates[0],
				Name: call.Name,
				Line: call.Line,
				Kind: call.Kind,
			})
		default:
			ambiguousCalls = append(ambiguousCalls, AmbiguousCall{
				From:       call.From,
				Name:       call.Name,
				Line:       call.Line,
				Candidates: append([]string(nil), candidates...),
			})
		}
	}

	parseErrorNodeIDs := map[string]struct{}{}
	for _, parseErr := range parseErrors {
		parseErrorNodeIDs[parseErr.NodeID] = struct{}{}
	}
	for idx := range nodes {
		if _, ok := parseErrorNodeIDs[nodes[idx].ID]; ok {
			nodes[idx].ParseError = true
		}
	}

	externalCommands, externalUsages := externalCommandsFromCounts(externalCounts)
	graph := Graph{
		Version:          1,
		GeneratedAt:      time.Now().Format(time.RFC3339),
		ActionsDB:        cfg.ActionsDB,
		Nodes:            nodes,
		Edges:            edges,
		AmbiguousCalls:   ambiguousCalls,
		DynamicCalls:     dynamicCalls,
		ParseErrors:      parseErrors,
		ExternalCommands: externalCommands,
		Stats: Stats{
			Nodes:                  len(nodes),
			Edges:                  len(edges),
			AmbiguousCalls:         len(ambiguousCalls),
			DynamicCalls:           len(dynamicCalls),
			ParseErrors:            len(parseErrors),
			ExternalCommandNames:   len(externalCommands),
			ExternalCommandUsages:  externalUsages,
			ActionsWithParseErrors: len(parseErrorNodeIDs),
		},
	}

	if err := writeJSON(cfg.Output, graph); err != nil {
		return BuildResult{}, fmt.Errorf("zmx graph build: write graph json: %w", err)
	}
	fmt.Fprintf(stdout, "graph build over, nodes %d edges %d ambiguous %d dynamic %d parse-errors %d\n",
		graph.Stats.Nodes,
		graph.Stats.Edges,
		graph.Stats.AmbiguousCalls,
		graph.Stats.DynamicCalls,
		graph.Stats.ParseErrors,
	)

	return BuildResult{
		Nodes:            graph.Stats.Nodes,
		DirectEdges:      graph.Stats.Edges,
		AmbiguousCalls:   graph.Stats.AmbiguousCalls,
		DynamicCalls:     graph.Stats.DynamicCalls,
		ParseErrors:      graph.Stats.ParseErrors,
		ExternalCommands: graph.Stats.ExternalCommandNames,
		Output:           cfg.Output,
	}, nil
}

func readActionRecords(path string) ([]actionRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []actionRecord
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s:%d: expected 3 fields, got %d", path, lineNumber, len(fields))
		}
		sourcePath := fields[1]
		lineValue, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: invalid action line %q: %w", path, lineNumber, fields[2], err)
		}
		realSource, err := filepath.EvalSymlinks(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: resolve source %s: %w", path, lineNumber, sourcePath, err)
		}
		records = append(records, actionRecord{
			Name:       fields[0],
			SourcePath: sourcePath,
			RealSource: realSource,
			Line:       lineValue,
			ID:         actionID(fields[0], realSource, lineValue),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func buildNodes(records []actionRecord) ([]Node, map[string]actionRecord, map[string][]string) {
	recordsByID := map[string]actionRecord{}
	nameIDs := map[string]map[string]struct{}{}
	for _, record := range records {
		if _, exists := recordsByID[record.ID]; exists {
			continue
		}
		recordsByID[record.ID] = record
		if nameIDs[record.Name] == nil {
			nameIDs[record.Name] = map[string]struct{}{}
		}
		nameIDs[record.Name][record.ID] = struct{}{}
	}

	nameIndex := make(map[string][]string, len(nameIDs))
	for name, ids := range nameIDs {
		for id := range ids {
			nameIndex[name] = append(nameIndex[name], id)
		}
		sort.Strings(nameIndex[name])
	}

	nodes := make([]Node, 0, len(recordsByID))
	for _, record := range recordsByID {
		nodes = append(nodes, Node{
			ID:            record.ID,
			Name:          record.Name,
			Source:        record.SourcePath,
			RealSource:    record.RealSource,
			Line:          record.Line,
			SourceGroup:   sourceGroup(record.RealSource),
			DuplicateName: len(nameIndex[record.Name]) > 1,
		})
	}
	sort.Slice(nodes, func(i int, j int) bool {
		if nodes[i].Name != nodes[j].Name {
			return nodes[i].Name < nodes[j].Name
		}
		if nodes[i].RealSource != nodes[j].RealSource {
			return nodes[i].RealSource < nodes[j].RealSource
		}
		return nodes[i].Line < nodes[j].Line
	})
	return nodes, recordsByID, nameIndex
}

func collectCalls(ctx context.Context, recordsByID map[string]actionRecord) ([]callRecord, []DynamicCall, []ParseError, error) {
	records := make([]actionRecord, 0, len(recordsByID))
	for _, record := range recordsByID {
		records = append(records, record)
	}
	sort.Slice(records, func(i int, j int) bool {
		if records[i].RealSource != records[j].RealSource {
			return records[i].RealSource < records[j].RealSource
		}
		if records[i].Line != records[j].Line {
			return records[i].Line < records[j].Line
		}
		return records[i].Name < records[j].Name
	})

	recordsBySource := map[string][]actionRecord{}
	for _, record := range records {
		recordsBySource[record.RealSource] = append(recordsBySource[record.RealSource], record)
	}

	var calls []callRecord
	var dynamicCalls []DynamicCall
	var parseErrors []ParseError
	for _, source := range sortedSourcePaths(recordsBySource) {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		sourceRecords := recordsBySource[source]
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read source %s: %w", source, err)
		}

		sourceCalls, sourceDynamicCalls, sourceParseErrors, wholeFileErr := parseSourceActions(source, data, sourceRecords)
		if wholeFileErr == nil {
			calls = append(calls, sourceCalls...)
			dynamicCalls = append(dynamicCalls, sourceDynamicCalls...)
			parseErrors = append(parseErrors, sourceParseErrors...)
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, record := range sourceRecords {
			block, err := functionBlock(lines, record.Line)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("%s:%d %s: %w", record.RealSource, record.Line, record.Name, err)
			}
			actionCalls, actionDynamicCalls, parseErr := parseActionCalls(record, block)
			if parseErr != nil {
				parseErrors = append(parseErrors, ParseError{
					NodeID: record.ID,
					Name:   record.Name,
					Source: record.RealSource,
					Line:   record.Line,
					Error:  parseErr.Error(),
				})
				continue
			}
			calls = append(calls, actionCalls...)
			dynamicCalls = append(dynamicCalls, actionDynamicCalls...)
		}
	}
	return calls, dynamicCalls, parseErrors, nil
}

func sortedSourcePaths(recordsBySource map[string][]actionRecord) []string {
	paths := make([]string, 0, len(recordsBySource))
	for path := range recordsBySource {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func parseSourceActions(source string, data []byte, records []actionRecord) ([]callRecord, []DynamicCall, []ParseError, error) {
	parser := syntax.NewParser(syntax.Variant(syntax.LangZsh))
	file, err := parser.Parse(strings.NewReader(string(data)), source)
	if err != nil {
		return nil, nil, nil, err
	}

	wanted := map[string]actionRecord{}
	for _, record := range records {
		wanted[actionKey(record.Name, record.Line)] = record
	}
	found := map[string]struct{}{}
	var calls []callRecord
	var dynamicCalls []DynamicCall
	syntax.Walk(file, func(node syntax.Node) bool {
		fn, ok := node.(*syntax.FuncDecl)
		if !ok {
			return true
		}
		for _, name := range functionNames(fn) {
			key := actionKey(name, int(fn.Pos().Line()))
			record, exists := wanted[key]
			if !exists {
				continue
			}
			found[key] = struct{}{}
			actionCalls, actionDynamicCalls := collectCallsFromNode(record, fn.Body, 1)
			calls = append(calls, actionCalls...)
			dynamicCalls = append(dynamicCalls, actionDynamicCalls...)
		}
		return false
	})

	var parseErrors []ParseError
	for key, record := range wanted {
		if _, exists := found[key]; exists {
			continue
		}
		parseErrors = append(parseErrors, ParseError{
			NodeID: record.ID,
			Name:   record.Name,
			Source: record.RealSource,
			Line:   record.Line,
			Error:  "action function not found in parsed AST",
		})
	}
	sort.Slice(parseErrors, func(i int, j int) bool {
		if parseErrors[i].Source != parseErrors[j].Source {
			return parseErrors[i].Source < parseErrors[j].Source
		}
		if parseErrors[i].Line != parseErrors[j].Line {
			return parseErrors[i].Line < parseErrors[j].Line
		}
		return parseErrors[i].Name < parseErrors[j].Name
	})
	return calls, dynamicCalls, parseErrors, nil
}

func parseActionCalls(record actionRecord, source string) ([]callRecord, []DynamicCall, error) {
	parser := syntax.NewParser(syntax.Variant(syntax.LangZsh))
	file, err := parser.Parse(strings.NewReader(source), record.RealSource)
	if err != nil {
		return nil, nil, err
	}

	var calls []callRecord
	var dynamicCalls []DynamicCall
	found := false
	syntax.Walk(file, func(node syntax.Node) bool {
		fn, ok := node.(*syntax.FuncDecl)
		if !ok {
			return true
		}
		found = true
		actionCalls, actionDynamicCalls := collectCallsFromNode(record, fn.Body, record.Line)
		calls = append(calls, actionCalls...)
		dynamicCalls = append(dynamicCalls, actionDynamicCalls...)
		return false
	})
	if !found {
		actionCalls, actionDynamicCalls := collectCallsFromNode(record, file, record.Line)
		calls = append(calls, actionCalls...)
		dynamicCalls = append(dynamicCalls, actionDynamicCalls...)
	}
	return calls, dynamicCalls, nil
}

func collectCallsFromNode(record actionRecord, node syntax.Node, baseLine int) ([]callRecord, []DynamicCall) {
	var calls []callRecord
	var dynamicCalls []DynamicCall
	syntax.Walk(node, func(node syntax.Node) bool {
		if _, ok := node.(*syntax.FuncDecl); ok {
			return false
		}
		call, ok := node.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		line := absoluteLine(baseLine, call.Pos())
		command, literal := staticWord(call.Args[0])
		if !literal {
			dynamicCalls = append(dynamicCalls, DynamicCall{
				From:   record.ID,
				Line:   line,
				Text:   printWord(call.Args[0]),
				Reason: dynamicReasonCommandWord,
			})
			return true
		}
		switch command {
		case "":
			return true
		case "eval":
			dynamicCalls = append(dynamicCalls, DynamicCall{
				From:   record.ID,
				Line:   line,
				Text:   printCall(call),
				Reason: dynamicReasonEval,
			})
			return true
		case "zmx-call", "zmx-call.sh":
			if len(call.Args) < 2 {
				return true
			}
			target, ok := staticWord(call.Args[1])
			if !ok {
				dynamicCalls = append(dynamicCalls, DynamicCall{
					From:   record.ID,
					Line:   line,
					Text:   printCall(call),
					Reason: dynamicReasonZMXCall,
				})
				return true
			}
			if target != "" {
				calls = append(calls, callRecord{From: record.ID, Name: target, Line: line, Kind: edgeKindZMXCall})
			}
			return true
		default:
			calls = append(calls, callRecord{From: record.ID, Name: command, Line: line, Kind: edgeKindDirect})
			return true
		}
	})
	return calls, dynamicCalls
}

func functionNames(fn *syntax.FuncDecl) []string {
	var names []string
	if fn.Name != nil {
		names = append(names, fn.Name.Value)
	}
	for _, name := range fn.Names {
		names = append(names, name.Value)
	}
	return names
}

func actionKey(name string, line int) string {
	return fmt.Sprintf("%s\x00%d", name, line)
}

func functionBlock(lines []string, lineNumber int) (string, error) {
	start := lineNumber - 1
	if start < 0 || start >= len(lines) {
		return "", fmt.Errorf("action start line outside source file")
	}
	end := len(lines)
	for idx := start + 1; idx < len(lines); idx++ {
		if functionPattern.MatchString(lines[idx]) {
			end = idx
			break
		}
	}
	return strings.Join(lines[start:end], "\n") + "\n", nil
}

func staticWord(word *syntax.Word) (string, bool) {
	if word == nil || len(word.Parts) == 0 {
		return "", false
	}
	var builder strings.Builder
	for _, part := range word.Parts {
		value, ok := staticWordPart(part)
		if !ok {
			return "", false
		}
		builder.WriteString(value)
	}
	return builder.String(), true
}

func staticWordPart(part syntax.WordPart) (string, bool) {
	switch part := part.(type) {
	case *syntax.Lit:
		return part.Value, true
	case *syntax.SglQuoted:
		return part.Value, true
	case *syntax.DblQuoted:
		var builder strings.Builder
		for _, nestedPart := range part.Parts {
			value, ok := staticWordPart(nestedPart)
			if !ok {
				return "", false
			}
			builder.WriteString(value)
		}
		return builder.String(), true
	default:
		return "", false
	}
}

func printWord(word *syntax.Word) string {
	var builder strings.Builder
	if err := syntax.NewPrinter().Print(&builder, word); err != nil {
		return ""
	}
	return builder.String()
}

func printCall(call *syntax.CallExpr) string {
	var builder strings.Builder
	if err := syntax.NewPrinter().Print(&builder, call); err != nil {
		return ""
	}
	return builder.String()
}

func absoluteLine(startLine int, pos syntax.Pos) int {
	if !pos.IsValid() {
		return startLine
	}
	return startLine + int(pos.Line()) - 1
}

func externalCommandsFromCounts(counts map[string]int) ([]ExternalCommand, int) {
	commands := make([]ExternalCommand, 0, len(counts))
	total := 0
	for name, count := range counts {
		total += count
		commands = append(commands, ExternalCommand{Name: name, Count: count})
	}
	sort.Slice(commands, func(i int, j int) bool {
		if commands[i].Count != commands[j].Count {
			return commands[i].Count > commands[j].Count
		}
		return commands[i].Name < commands[j].Name
	})
	return commands, total
}

func writeJSON(path string, graph Graph) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(graph); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func actionID(name string, realSource string, line int) string {
	hash := sha1.Sum([]byte(fmt.Sprintf("%s\x00%s\x00%d", name, realSource, line)))
	return fmt.Sprintf("%s@%s", name, hex.EncodeToString(hash[:])[:12])
}

func sourceGroup(realSource string) string {
	dir := filepath.Dir(realSource)
	parent := filepath.Base(filepath.Dir(dir))
	base := filepath.Base(dir)
	if parent == "." || parent == string(filepath.Separator) || parent == "" {
		return base
	}
	return parent + "/" + base
}
