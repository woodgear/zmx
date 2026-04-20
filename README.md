# zmx

`zmx` is the discovery and runtime layer for shell actions in the local environment.

It turns shell functions from multiple repositories into a manageable action registry, then provides interactive and non-interactive ways to inspect and run those action objects.

## What zmx does

`zmx` is responsible for:
- discovering action definitions from `SHELL_ACTIONS_PATH`
- indexing action source paths
- building an action database
- generating a unified import layer
- letting users inspect, find, and run actions

In practice, repositories such as `gnote`, `maid`, and other action repositories define shell functions, while `zmx` is what makes them discoverable and usable as environment-level action objects.

## Runtime Files

Runtime state is stored under `~/.zmx/`.

Important files:
- `~/.zmx/actions.db`
  - action registry in the form `function_name source_file line_number`
- `~/.zmx/import.sh`
  - generated source list for all indexed action files
- `~/.zmx/aio.sh` and `~/.zmx/aio.sh.zwc`
  - optional concatenated action source and its compiled zsh cache
- `~/.zmx/index/`
  - symlinked view of action source paths
- `~/.zmx/actions.record`
  - execution records

## Install

Build the Go reload binary first:

```bash
go build -o bin/zmx ./cmd/zmx
(cd shellargs && go build -o /usr/local/bin/shellargs ./cmd/shellargs)
```

Then load the plugin as usual:

```zsh
zplug "woodgear/zmx"
export SHELL_ACTIONS_PATH=/path/to/actions:/another/path
zmx-reload-shell-actions
```

`zmx-reload-shell-actions` calls `zmx` from `PATH`. Make sure the built binary is installed or the build output directory is exported into `PATH`.

The shell helper commands use `shellargs` from `PATH`. Keep `/usr/local/bin/shellargs` on the latest version.

Compiled cache is disabled by default. Enable it explicitly with:

```zsh
export ZMX_ENABLE_COMPILED_CACHE=1
```

After the initial build, normal startup can use:

```zsh
zmx-load-shell-actions
```

`zmx-load-shell-actions` uses `~/.zmx/import.sh` by default. If `ZMX_ENABLE_COMPILED_CACHE=1`, it prefers `~/.zmx/aio.sh.zwc` when the cache is fresh and falls back to `~/.zmx/import.sh` if the cache is missing or cannot be rebuilt.

## Common Usage

### Discover actions

```zsh
zmx-list
zmx-list-path
zmx-help <action>
zmx-find-path-of-action <action>
zmx-actions-info <action>
```

### Manage action sources

```zsh
zmx-add-path /path/to/new/actions
zmx-reload-shell-actions
```

### Run actions

Interactive:
- `,xm`
  - choose a global action from the indexed database
- `,xx`
  - choose a local action from `./actions.sh`

Non-interactive:

```bash
zmx-call.sh some-action
zmx-call.sh some-action arg1 arg2
```

## How it works

1. `zmx-reload-shell-actions` calls the Go binary `zmx reload`
2. `zmx reload` rebuilds `~/.zmx/index/`, `~/.zmx/actions.db`, `~/.zmx/import.sh`, and `~/.zmx/md5/`
3. the binary scans shell files directly and hashes source files with worker pools, so build-db and md5 generation run concurrently across files
4. after generation finishes, the shell wrapper loads actions into the current shell; when `ZMX_ENABLE_COMPILED_CACHE=1`, it also builds and uses `~/.zmx/aio.sh.zwc`

Functions beginning with `_` are treated as private and not indexed as public actions.

If an action needs arguments, zmx uses the `arg-len` annotation to decide whether to execute directly or insert the command into the shell for completion.

## Main Commands

- `zmx-reload-shell-actions`
  - rebuild index, action database, imports, and reload
- `zmx-load-shell-actions`
  - load actions from generated imports
- `zmx-list`
  - list all indexed actions
- `zmx-help`
  - show help or source information for an action
- `zmx-list-path`
  - list configured action paths
- `zmx-add-path`
  - add a new action path into the environment
- `zmx-find-path-of-action`
  - find which file defines an action
- `zmx-find-base-of-action`
  - find the base directory of the action source

## Files

- `cmd/zmx`
  - Go CLI entrypoint for reload
- `internal/reload`
  - Go reload pipeline and tests
- `zmx.plugin.zsh`
  - shell integration and runtime loading
- `zmx-call.sh`
  - non-interactive executor
- `runner/lua/zmx.lua.sh`
  - experimental runner
- `legacy.sh`
  - older implementation notes and experiments
