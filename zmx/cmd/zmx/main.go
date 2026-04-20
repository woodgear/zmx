package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/woodgear/zmx/internal/reload"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return 2
	}

	switch args[0] {
	case "reload":
		return runReload(args[1:])
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", args[0])
		printUsage(os.Stderr)
		return 2
	}
}

func runReload(args []string) int {
	fs := flag.NewFlagSet("reload", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	defaultBase := os.Getenv("ZMX_BASE")
	if defaultBase == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultBase = filepath.Join(homeDir, ".zmx")
		}
	}

	var cfg reload.Config
	fs.StringVar(&cfg.Base, "base", defaultBase, "runtime directory")
	fs.StringVar(&cfg.ActionsPath, "actions-path", os.Getenv("SHELL_ACTIONS_PATH"), "colon-separated action source paths")
	fs.StringVar(&cfg.GenPath, "gen-path", os.Getenv("ZMX_GEN_PATH"), "colon-separated generator commands")
	fs.StringVar(&cfg.CallTarget, "call-target", os.Getenv("ZMX_CALL_TARGET"), "path to zmx-call.sh used for generated tools")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "reload does not accept positional args: %v\n", fs.Args())
		return 2
	}

	cfg.Stdout = os.Stdout
	cfg.Stderr = os.Stderr
	if _, err := reload.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  zmx reload [--base PATH] [--actions-path LIST] [--gen-path LIST] [--call-target PATH]")
}
