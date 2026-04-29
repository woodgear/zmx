package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"shellargs/internal/engine"
	"shellargs/internal/spec"
)

var version = "dev"
var commit = ""

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return ""
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			if exitErr.err != nil && exitErr.err.Error() != "" {
				fmt.Fprintln(os.Stderr, exitErr.err)
			}
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		printRootHelp(stderr)
		return errors.New("missing subcommand")
	}

	switch args[0] {
	case "parse":
		return runParse(args[1:], stdout, stderr)
	case "help":
		return runHelp(args[1:], stdout, stderr)
	case "is-help":
		return runIsHelp(args[1:], stdout, stderr)
	case "completion":
		return runCompletion(args[1:], stdout, stderr)
	case "-h", "--help", "help-self":
		printRootHelp(stdout)
		return nil
	case "-v", "--version", "version":
		printVersion(stdout)
		return nil
	default:
		printRootHelp(stderr)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runParse(args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	specText := new(string)
	specFile := new(string)
	specBase64 := new(string)
	bindSpecFlags(fs, specText, specFile, specBase64)
	prog := fs.String("prog", "", "program name override")
	autoHelp := fs.Bool("auto-help", false, "enable go-flags builtin --help handling")
	pretty := fs.Bool("pretty", true, "pretty-print json")

	if err := fs.Parse(args); err != nil {
		printParseHelp(stderr)
		return err
	}

	rawSpec, err := loadSpec(*specText, *specFile, *specBase64)
	if err != nil {
		return err
	}

	parsedSpec, err := spec.Parse(rawSpec)
	if err != nil {
		return err
	}
	if *prog != "" {
		parsedSpec.Name = *prog
	}

	parser, err := engine.New(parsedSpec)
	if err != nil {
		return err
	}

	result, err := parser.Parse(engine.ParseOptions{
		Args:     fs.Args(),
		AutoHelp: *autoHelp,
		Stdout:   stdout,
	})
	if err != nil {
		if errors.Is(err, engine.ErrHelpShown) {
			return nil
		}
		return err
	}
	if result.CompletionHandled {
		return nil
	}

	enc := json.NewEncoder(stdout)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(result.Values)
}

func runHelp(args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	specText := new(string)
	specFile := new(string)
	specBase64 := new(string)
	bindSpecFlags(fs, specText, specFile, specBase64)
	prog := fs.String("prog", "", "program name override")

	if err := fs.Parse(args); err != nil {
		printHelpHelp(stderr)
		return err
	}

	rawSpec, err := loadSpec(*specText, *specFile, *specBase64)
	if err != nil {
		return err
	}

	parsedSpec, err := spec.Parse(rawSpec)
	if err != nil {
		return err
	}
	if *prog != "" {
		parsedSpec.Name = *prog
	}

	parser, err := engine.New(parsedSpec)
	if err != nil {
		return err
	}

	return parser.WriteHelp(stdout)
}

func runIsHelp(args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("is-help", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		printIsHelpHelp(stderr)
		return err
	}

	if targetWantsHelp(fs.Args()) {
		return nil
	}
	return &exitError{code: 1}
}

func runCompletion(args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("completion", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	specText := new(string)
	specFile := new(string)
	specBase64 := new(string)
	bindSpecFlags(fs, specText, specFile, specBase64)
	prog := fs.String("prog", "", "target function/command name")
	shell := fs.String("shell", "bash", "target shell")
	runner := fs.String("runner", "shellargs", "shellargs executable used inside the completion script")

	if err := fs.Parse(args); err != nil {
		printCompletionHelp(stderr)
		return err
	}

	rawSpec, err := loadSpec(*specText, *specFile, *specBase64)
	if err != nil {
		return err
	}

	parsedSpec, err := spec.Parse(rawSpec)
	if err != nil {
		return err
	}
	if *prog != "" {
		parsedSpec.Name = *prog
	}
	if parsedSpec.Name == "" {
		return errors.New("spec name is required for completion generation; set `name:` or pass --prog")
	}

	script, err := engine.BashCompletionScript(engine.BashCompletionOptions{
		Runner:     *runner,
		Program:    parsedSpec.Name,
		SpecBase64: base64.StdEncoding.EncodeToString([]byte(rawSpec)),
		Shell:      *shell,
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(stdout, script)
	return err
}

func bindSpecFlags(fs *flag.FlagSet, specText, specFile, specBase64 *string) {
	fs.StringVar(specText, "spec", "", "inline spec text")
	fs.StringVar(specFile, "spec-file", "", "path to spec file")
	fs.StringVar(specBase64, "spec-base64", "", "base64 encoded spec text")
}

func loadSpec(specText, specFile, specBase64 string) (string, error) {
	sources := 0
	if specText != "" {
		sources++
	}
	if specFile != "" {
		sources++
	}
	if specBase64 != "" {
		sources++
	}
	if sources == 0 {
		return "", errors.New("one of --spec, --spec-file, or --spec-base64 is required")
	}
	if sources > 1 {
		return "", errors.New("only one of --spec, --spec-file, or --spec-base64 may be used")
	}

	var rawSpec string
	switch {
	case specText != "":
		rawSpec = specText
	case specFile != "":
		data, err := os.ReadFile(specFile)
		if err != nil {
			return "", err
		}
		rawSpec = string(data)
	default:
		data, err := base64.StdEncoding.DecodeString(specBase64)
		if err != nil {
			return "", fmt.Errorf("decode spec base64: %w", err)
		}
		rawSpec = string(data)
	}

	return autoTrimSpecDoc(rawSpec), nil
}

func autoTrimSpecDoc(input string) string {
	first, ok := firstNonEmptyLine(input)
	if !ok {
		return input
	}
	last, ok := lastNonEmptyLine(input)
	if !ok {
		return input
	}
	if first.start == last.start {
		return input
	}
	if strings.TrimSpace(lineMarker(input[first.start:first.end])) != "@@@" {
		return input
	}
	if strings.TrimSpace(lineMarker(input[last.start:last.end])) != "@@@" {
		return input
	}
	return input[first.nextStart:last.start]
}

type lineBounds struct {
	start     int
	end       int
	nextStart int
}

func firstNonEmptyLine(input string) (lineBounds, bool) {
	start := 0
	for start <= len(input) {
		lineStart, lineEnd, nextStart, ok := nextLineBounds(input, start)
		if !ok {
			return lineBounds{}, false
		}
		if strings.TrimSpace(lineMarker(input[lineStart:lineEnd])) != "" {
			return lineBounds{start: lineStart, end: lineEnd, nextStart: nextStart}, true
		}
		if nextStart <= start {
			break
		}
		start = nextStart
	}
	return lineBounds{}, false
}

func lastNonEmptyLine(input string) (lineBounds, bool) {
	start := 0
	var last lineBounds
	found := false
	for start <= len(input) {
		lineStart, lineEnd, nextStart, ok := nextLineBounds(input, start)
		if !ok {
			break
		}
		if strings.TrimSpace(lineMarker(input[lineStart:lineEnd])) != "" {
			last = lineBounds{start: lineStart, end: lineEnd, nextStart: nextStart}
			found = true
		}
		if nextStart <= start {
			break
		}
		start = nextStart
	}
	return last, found
}

func nextLineBounds(input string, start int) (lineStart int, lineEnd int, nextStart int, ok bool) {
	if start >= len(input) {
		return 0, 0, 0, false
	}
	lineStart = start
	idx := strings.IndexByte(input[start:], '\n')
	if idx < 0 {
		lineEnd = len(input)
		nextStart = len(input)
		return lineStart, lineEnd, nextStart, true
	}
	lineEnd = start + idx
	nextStart = lineEnd + 1
	return lineStart, lineEnd, nextStart, true
}

func lineMarker(line string) string {
	return strings.TrimSuffix(line, "\r")
}

func targetWantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func printRootHelp(w io.Writer) {
	info := buildVersionInfo()
	fmt.Fprintf(w, "shellargs version %s\ncommit %s\n\n", info.version, info.commit)
	_, _ = fmt.Fprint(w, strings.TrimSpace(`
shellargs turns a declarative spec string plus argv into JSON using go-flags.

Subcommands:
  parse        Parse argv and print JSON
  help         Render generated help text from the spec
  is-help      Check whether target argv requests help
  completion   Render a bash completion script for the target command

Examples:
  shellargs parse --auto-help --spec "$SPEC" -- "$@"
  shellargs is-help -- "$@"
  shellargs help --spec-file repo-sync.args
  shellargs completion --shell bash --prog repo-sync --spec "$SPEC"
`)+"\n")
}

func printVersion(w io.Writer) {
	info := buildVersionInfo()
	fmt.Fprintf(w, "shellargs version %s\n", info.version)
	fmt.Fprintf(w, "commit %s\n", info.commit)
}

type versionInfo struct {
	version string
	commit  string
}

func buildVersionInfo() versionInfo {
	out := versionInfo{
		version: version,
		commit:  commit,
	}
	if out.version == "" {
		out.version = "dev"
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		if out.version == "dev" && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
			out.version = buildInfo.Main.Version
		}
		if out.commit == "" {
			out.commit = vcsRevision(buildInfo)
		}
		if out.commit == "" {
			out.commit = commitFromModuleVersion(buildInfo.Main.Version)
		}
	}

	if out.commit == "" {
		out.commit = "unknown"
	}
	return out
}

func vcsRevision(buildInfo *debug.BuildInfo) string {
	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}

func commitFromModuleVersion(value string) string {
	parts := strings.Split(value, "-")
	if len(parts) < 3 {
		return ""
	}
	candidate := parts[len(parts)-1]
	if len(candidate) < 7 {
		return ""
	}
	for _, r := range candidate {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return ""
	}
	return candidate
}

func printParseHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, strings.TrimSpace(`
Usage:
  shellargs parse [--spec TEXT | --spec-file PATH | --spec-base64 B64] [--prog NAME] [--auto-help] [--pretty] -- [argv...]
`)+"\n")
}

func printHelpHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, strings.TrimSpace(`
Usage:
  shellargs help [--spec TEXT | --spec-file PATH | --spec-base64 B64] [--prog NAME]
`)+"\n")
}

func printIsHelpHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, strings.TrimSpace(`
Usage:
  shellargs is-help -- [argv...]
`)+"\n")
}

func printCompletionHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, strings.TrimSpace(`
Usage:
  shellargs completion [--spec TEXT | --spec-file PATH | --spec-base64 B64] [--prog NAME] [--shell bash] [--runner shellargs]
`)+"\n")
}
