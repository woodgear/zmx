package reload

import (
	"bufio"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var actionPattern = regexp.MustCompile(`^function\s*([^\s()_]+).*[\(\{][^\}\)]*$`)

type Config struct {
	Base        string
	ActionsPath string
	GenPath     string
	CallTarget  string
	Stdout      io.Writer
	Stderr      io.Writer
}

type Result struct {
	Actions int
	Files   int
}

type runner struct {
	base   string
	stdout io.Writer
	stderr io.Writer
}

type actionRecord struct {
	Name       string
	SourcePath string
	Line       int
}

type scanTarget struct {
	DisplayPath string
	RealPath    string
}

type md5Result struct {
	SourcePath string
	Value      string
}

func Run(ctx context.Context, cfg Config) (Result, error) {
	if cfg.Base == "" {
		return Result{}, errors.New("zmx reload: --base is required")
	}

	r := runner{
		base:   cfg.Base,
		stdout: cfg.Stdout,
		stderr: cfg.Stderr,
	}
	if r.stdout == nil {
		r.stdout = io.Discard
	}
	if r.stderr == nil {
		r.stderr = io.Discard
	}

	if err := os.MkdirAll(cfg.Base, 0o755); err != nil {
		return Result{}, fmt.Errorf("zmx reload: create base dir: %w", err)
	}
	if cfg.CallTarget != "" {
		if err := ensureRuntimeTools(cfg.Base, cfg.CallTarget); err != nil {
			return Result{}, fmt.Errorf("zmx reload: ensure runtime tools: %w", err)
		}
	}

	actionPaths := splitAndSortPaths(cfg.ActionsPath)
	genPaths := splitAndSortPaths(cfg.GenPath)

	r.printf("reload step: index\n")
	indexStart := time.Now()
	if err := rebuildIndex(cfg.Base, actionPaths, r.stdout); err != nil {
		_ = r.appendRecord("reload step failed: index")
		return Result{}, fmt.Errorf("zmx reload: rebuild index: %w", err)
	}
	if err := r.appendRecord(fmt.Sprintf("index over, spend %s.", formatDuration(time.Since(indexStart)))); err != nil {
		return Result{}, err
	}

	r.printf("reload step: autogen\n")
	autogenStart := time.Now()
	if err := runAutogen(ctx, genPaths, r.stdout, r.stderr); err != nil {
		_ = r.appendRecord("reload step failed: autogen")
		return Result{}, fmt.Errorf("zmx reload: autogen: %w", err)
	}
	if len(genPaths) > 0 {
		if err := r.appendRecord(fmt.Sprintf("autogen over, spend %s.", formatDuration(time.Since(autogenStart)))); err != nil {
			return Result{}, err
		}
	}

	r.printf("reload step: build-db\n")
	buildStart := time.Now()
	records, err := buildActionDB(cfg.Base, r.stdout)
	if err != nil {
		_ = r.appendRecord("reload step failed: build-db")
		return Result{}, fmt.Errorf("zmx reload: build actions db: %w", err)
	}
	if err := writeActionDB(cfg.Base, records); err != nil {
		return Result{}, fmt.Errorf("zmx reload: write actions db: %w", err)
	}
	if err := r.appendRecord(fmt.Sprintf("build over, spend %s.", formatDuration(time.Since(buildStart)))); err != nil {
		return Result{}, err
	}

	r.printf("reload step: gen-import\n")
	importStart := time.Now()
	sourceFiles := uniqueSourceFiles(records)
	if err := writeImportFile(cfg.Base, sourceFiles); err != nil {
		_ = r.appendRecord("reload step failed: gen-import")
		return Result{}, fmt.Errorf("zmx reload: write import.sh: %w", err)
	}
	if err := r.appendRecord(fmt.Sprintf("gen-import over, spend %s.", formatDuration(time.Since(importStart)))); err != nil {
		return Result{}, err
	}

	r.printf("reload step: gen-md5\n")
	md5Start := time.Now()
	if err := writeMD5Files(cfg.Base, sourceFiles); err != nil {
		_ = r.appendRecord("reload step failed: gen-md5")
		return Result{}, fmt.Errorf("zmx reload: write md5 files: %w", err)
	}
	if err := r.appendRecord(fmt.Sprintf("gen-md5 over, spend %s.", formatDuration(time.Since(md5Start)))); err != nil {
		return Result{}, err
	}

	return Result{Actions: len(records), Files: len(sourceFiles)}, nil
}

func (r runner) printf(format string, args ...any) {
	fmt.Fprintf(r.stdout, format, args...)
}

func (r runner) appendRecord(record string) error {
	r.printf("%s\n", record)
	recordPath := filepath.Join(r.base, "record")
	file, err := os.OpenFile(recordPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("zmx reload: append record: %w", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, record); err != nil {
		return fmt.Errorf("zmx reload: append record: %w", err)
	}
	return nil
}

func ensureRuntimeTools(base string, callTarget string) error {
	toolsDir := filepath.Join(base, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return err
	}

	callPath := filepath.Join(toolsDir, "zmx-call")
	callScript := fmt.Sprintf("#!/bin/bash\nexec %s \"$@\"\n", shellQuote(callTarget))
	if err := os.WriteFile(callPath, []byte(callScript), 0o755); err != nil {
		return err
	}

	linkPath := filepath.Join(toolsDir, "zmx-call.sh")
	if err := os.RemoveAll(linkPath); err != nil {
		return err
	}
	return os.Symlink(callPath, linkPath)
}

func rebuildIndex(base string, actionPaths []string, stdout io.Writer) error {
	indexDir := filepath.Join(base, "index")
	if err := os.RemoveAll(indexDir); err != nil {
		return err
	}
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "start index")
	for _, path := range actionPaths {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(stdout, "%s not exist\n", path)
				continue
			}
			return err
		}

		linkPath := filepath.Join(indexDir, strings.ReplaceAll(path, "/", "_"))
		fmt.Fprintf(stdout, "index %s %s\n", path, linkPath)
		if err := os.RemoveAll(linkPath); err != nil {
			return err
		}
		if err := os.Symlink(path, linkPath); err != nil {
			return err
		}
	}
	return nil
}

func runAutogen(ctx context.Context, genPaths []string, stdout io.Writer, stderr io.Writer) error {
	for _, path := range genPaths {
		fmt.Fprintf(stdout, "gen %s\n", path)
		cmd := exec.CommandContext(ctx, path)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func buildActionDB(base string, stdout io.Writer) ([]actionRecord, error) {
	fmt.Fprintln(stdout, "start build")
	targets, err := collectScanTargets(filepath.Join(base, "index"))
	if err != nil {
		return nil, err
	}

	records, err := scanTargets(targets)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		fmt.Fprintf(stdout, "%s   %s   %d\n", record.Name, record.SourcePath, record.Line)
	}
	return records, nil
}

func collectScanTargets(indexDir string) ([]scanTarget, error) {
	entries, err := os.ReadDir(indexDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var targets []scanTarget
	for _, entry := range entries {
		rootPath := filepath.Join(indexDir, entry.Name())
		entryTargets, err := collectScanTargetsFromPath(rootPath, rootPath, map[string]struct{}{})
		if err != nil {
			return nil, err
		}
		targets = append(targets, entryTargets...)
	}

	sort.Slice(targets, func(i int, j int) bool {
		return targets[i].DisplayPath < targets[j].DisplayPath
	})
	return targets, nil
}

func collectScanTargetsFromPath(realPath string, displayPath string, stack map[string]struct{}) ([]scanTarget, error) {
	info, err := os.Stat(realPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		if !isShellFile(displayPath) {
			return nil, nil
		}

		resolvedPath := realPath
		if evaluated, err := filepath.EvalSymlinks(realPath); err == nil {
			resolvedPath = evaluated
		}
		return []scanTarget{{DisplayPath: displayPath, RealPath: resolvedPath}}, nil
	}

	resolvedDir := realPath
	if evaluated, err := filepath.EvalSymlinks(realPath); err == nil {
		resolvedDir = evaluated
	}
	if _, exists := stack[resolvedDir]; exists {
		return nil, nil
	}

	stack[resolvedDir] = struct{}{}
	defer delete(stack, resolvedDir)

	entries, err := os.ReadDir(resolvedDir)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var targets []scanTarget
	for _, entry := range entries {
		childReal := filepath.Join(resolvedDir, entry.Name())
		childDisplay := filepath.Join(displayPath, entry.Name())
		childTargets, err := collectScanTargetsFromPath(childReal, childDisplay, stack)
		if err != nil {
			return nil, err
		}
		targets = append(targets, childTargets...)
	}
	return targets, nil
}

func scanTargets(targets []scanTarget) ([]actionRecord, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	type scanResult struct {
		records []actionRecord
		err     error
	}

	workerCount := min(goruntime.GOMAXPROCS(0), len(targets))
	jobs := make(chan scanTarget)
	results := make(chan scanResult, len(targets))

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				records, err := scanActionFile(target)
				results <- scanResult{records: records, err: err}
			}
		}()
	}

	go func() {
		for _, target := range targets {
			jobs <- target
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var allRecords []actionRecord
	var firstErr error
	for result := range results {
		if firstErr == nil && result.err != nil {
			firstErr = result.err
		}
		allRecords = append(allRecords, result.records...)
	}
	if firstErr != nil {
		return nil, firstErr
	}

	sort.Slice(allRecords, func(i int, j int) bool {
		left := allRecords[i]
		right := allRecords[j]
		if left.SourcePath != right.SourcePath {
			return left.SourcePath < right.SourcePath
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		return left.Name < right.Name
	})
	return allRecords, nil
}

func scanActionFile(target scanTarget) ([]actionRecord, error) {
	file, err := os.Open(target.RealPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var records []actionRecord
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		matches := actionPattern.FindStringSubmatch(scanner.Text())
		if len(matches) != 2 {
			continue
		}
		records = append(records, actionRecord{Name: matches[1], SourcePath: target.DisplayPath, Line: lineNumber})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func writeActionDB(base string, records []actionRecord) error {
	var builder strings.Builder
	for _, record := range records {
		fmt.Fprintf(&builder, "%s   %s   %d\n", record.Name, record.SourcePath, record.Line)
	}
	return writeFileAtomically(filepath.Join(base, "actions.db"), []byte(builder.String()), 0o644)
}

func writeImportFile(base string, sourceFiles []string) error {
	var builder strings.Builder
	for _, sourceFile := range sourceFiles {
		fmt.Fprintf(&builder, "source %s || true\n", sourceFile)
	}
	return writeFileAtomically(filepath.Join(base, "import.sh"), []byte(builder.String()), 0o644)
}

func writeMD5Files(base string, sourceFiles []string) error {
	md5Dir := filepath.Join(base, "md5")
	if err := os.RemoveAll(md5Dir); err != nil {
		return err
	}
	if err := os.MkdirAll(md5Dir, 0o755); err != nil {
		return err
	}
	if len(sourceFiles) == 0 {
		return nil
	}

	type hashResult struct {
		result md5Result
		err    error
	}

	workerCount := min(goruntime.GOMAXPROCS(0), len(sourceFiles))
	jobs := make(chan string)
	results := make(chan hashResult, len(sourceFiles))

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sourceFile := range jobs {
				value, err := hashFile(sourceFile)
				results <- hashResult{result: md5Result{SourcePath: sourceFile, Value: value}, err: err}
			}
		}()
	}

	go func() {
		for _, sourceFile := range sourceFiles {
			jobs <- sourceFile
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var hashes []md5Result
	var firstErr error
	for result := range results {
		if firstErr == nil && result.err != nil {
			firstErr = result.err
		}
		hashes = append(hashes, result.result)
	}
	if firstErr != nil {
		return firstErr
	}

	sort.Slice(hashes, func(i int, j int) bool {
		return hashes[i].SourcePath < hashes[j].SourcePath
	})
	for _, hash := range hashes {
		fileName := strings.ReplaceAll(hash.SourcePath, "/", "_") + ".md5"
		if err := writeFileAtomically(filepath.Join(md5Dir, fileName), []byte(hash.Value+"\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func uniqueSourceFiles(records []actionRecord) []string {
	seen := make(map[string]struct{}, len(records))
	sourceFiles := make([]string, 0, len(records))
	for _, record := range records {
		if _, exists := seen[record.SourcePath]; exists {
			continue
		}
		seen[record.SourcePath] = struct{}{}
		sourceFiles = append(sourceFiles, record.SourcePath)
	}
	sort.Strings(sourceFiles)
	return sourceFiles
}

func splitAndSortPaths(raw string) []string {
	if raw == "" {
		return nil
	}

	seen := map[string]struct{}{}
	var paths []string
	for _, part := range strings.Split(raw, ":") {
		path := strings.TrimSpace(part)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func writeFileAtomically(path string, content []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(mode); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func isShellFile(path string) bool {
	switch filepath.Ext(path) {
	case ".sh", ".bash", ".zsh":
		return true
	default:
		return false
	}
}

func formatDuration(value time.Duration) string {
	if value < time.Millisecond {
		return value.Round(time.Microsecond).String()
	}
	return value.Round(time.Millisecond).String()
}
