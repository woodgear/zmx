# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

`zmx` is the runtime and discovery layer for shell actions in the local environment.

Its job is not just to provide an Emacs `M-x` style launcher for zsh. More importantly, it is the mechanism that:
- discovers action definitions from multiple directories
- builds and maintains an index of available actions
- lets users inspect, find, and manage those action objects
- provides both interactive and non-interactive ways to run them

In the broader personal setup, repos such as `gnote`, `maid`, and other action repositories define shell functions, while `zmx` is responsible for turning them into a usable action registry.

## Core Model

An action object is a shell function that:
- is defined in a file under `SHELL_ACTIONS_PATH`
- does not start with `_`
- can be indexed into the zmx database

`zmx` treats these functions as discoverable objects with metadata:
- function name
- source file
- line number
- whether it expects arguments

The runtime state lives in `~/.zmx/`.

## How Discovery Works

### 1. Reload orchestration

`zmx-reload-shell-actions` is now a shell wrapper around the Go binary `cmd/zmx`.

The binary handles the generation pipeline. The shell wrapper still performs the final `source ~/.zmx/import.sh` step because a child process cannot mutate the parent shell.

### 2. Index action paths

`zmx reload` creates symlinks under `~/.zmx/index/` for every directory in `SHELL_ACTIONS_PATH`.

This gives zmx a stable local view of all configured action sources.

### 3. Build the action database

`zmx reload` scans indexed shell files and writes `~/.zmx/actions.db`.

Database format:
```text
function_name source_file line_number
```

This file is the main registry for discovering and locating actions.

The scan keeps the current ripgrep-compatible behavior, including its name-truncation edge cases for function names containing `_`.

### 4. Generate imports and hashes

`zmx reload` builds `~/.zmx/import.sh`, which sources all unique action files, and stores checksums in `~/.zmx/md5/`.

The scan and md5 phases use worker pools, so those stages can process files concurrently.

## Managing Action Objects

The most important commands for discovering and managing actions are:

- `zmx-list`
  - list all indexed action names
- `zmx-help [function]`
  - show help or source info for an action
- `zmx-actions-info <function>`
  - inspect the raw database entry for an action
- `zmx-find-path-of-action [function]`
  - find the source file of an action
- `zmx-find-base-of-action [function]`
  - find the base directory of an action source
- `zmx-list-path`
  - show all configured action source paths
- `zmx-add-path <path>`
  - append a new path into the action source set
- `zmx-reload-shell-actions`
  - rebuild index, database, imports, and reload
- `zmx-load-shell-actions`
  - load from the current generated import file

If the user is trying to understand what actions exist in the environment, start with:

```zsh
zmx-list
zmx-list-path
zmx-help <action>
zmx-find-path-of-action <action>
```

## Interactive Usage

Key bindings:
- `,xm`
  - choose a global action from the indexed database
- `,xx`
  - choose a local action from `./actions.sh`

Execution flow:
1. User opens the selector
2. zmx reads action names from `~/.zmx/actions.db`
3. User selects an action through `fzf`
4. zmx checks whether the function has an `arg-len` annotation
5. If no arguments are needed, it executes immediately
6. If arguments are needed, it inserts the function name into the command line

## Non-Interactive Usage

`zmx-call.sh` is the main non-interactive entrypoint.

It:
1. sources `~/.loadhome.sh`
2. sources `~/.zmx/import.sh` by default
3. if `ZMX_ENABLE_COMPILED_CACHE=1`, it may prefer `~/.zmx/aio.sh.zwc` when the cache is fresh
4. evaluates the requested action command

This is what other repos often call when they want access to the environment's action objects without recreating the indexing logic themselves.

## Preexec Behavior

`zmx_preexec` checks whether the command being executed is an indexed action.

If it is, zmx re-sources the action's source file before execution so the latest definition is used.

This matters because action files are edited frequently and are shared across multiple repositories.

## Installation & Setup

Typical setup:

```bash
go build -o bin/zmx ./cmd/zmx
(cd shellargs && go build -o /usr/local/bin/shellargs ./cmd/shellargs)
```

```zsh
zplug "woodgear/zmx"
export SHELL_ACTIONS_PATH=/path/to/actions:/another/path
zmx-reload-shell-actions
```

`zmx-reload-shell-actions` calls `zmx` from `PATH`. Make sure the built binary is installed or the build output directory is exported into `PATH`.

The shell helper commands use `shellargs` from `PATH`. Keep `/usr/local/bin/shellargs` on the latest version.

Compiled cache is disabled by default. Enable it with `export ZMX_ENABLE_COMPILED_CACHE=1` if you want zsh to build and load `~/.zmx/aio.sh.zwc`.

After the initial build, normal shell startup can use:

```zsh
zmx-load-shell-actions
```

## File Structure

- `cmd/zmx`
  - Go CLI entrypoint for reload
- `internal/reload`
  - Go reload pipeline and tests
- `zmx.plugin.zsh`
  - shell integration and runtime loading
- `zmx-call.sh`
  - non-interactive action executor
- `runner/lua/zmx.lua.sh`
  - experimental runner
- `legacy.sh`
  - older code and experiments

## Code Style

- in bash/zsh, prefer single-line local assignment when the value is available immediately
- do not split declaration and assignment into two lines such as:

```zsh
local value
value=$(some-command)
```

- prefer:

```zsh
local value=$(some-command)
```

- apply the same rule to simple parameter expansion and command substitution; finish the assignment on the same line when readability does not suffer

## Development Notes

- action discovery is based on shell function definitions, not a separate manifest
- functions starting with `_` are private and not exposed, but the current rg-compatible matcher still truncates names containing `_`
- `arg-len` annotations are used to mark actions that require arguments
- `~/.zmx/actions.db` is the central action registry
- `~/.zmx/import.sh` is the generated import layer
- `~/.zmx/aio.sh` and `~/.zmx/aio.sh.zwc` are the optional compiled load cache
- `~/.zmx/completions/_zmx_actions` is generated during `zmx reload` from action `shellargs` specs; `zmx-load-shell-actions` adds `~/.zmx/completions` to `fpath` and registers the generated `_zmx_actions` function if `compinit` has already run
- build `bin/zmx` with `go build -o bin/zmx ./cmd/zmx` and keep `/usr/local/bin/shellargs` on the latest version before testing shell integration
- when documenting zmx, focus on action discovery, inspection, path management, and execution flow
