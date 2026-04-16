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
- `~/.zmx/index/`
  - symlinked view of action source paths
- `~/.zmx/actions.record`
  - execution records

## Install

```zsh
zplug "woodgear/zmx"
export SHELL_ACTIONS_PATH=/path/to/actions:/another/path
zmx-reload-shell-actions
```

After the initial build, normal startup can use:

```zsh
zmx-load-shell-actions
```

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

1. `zmx` scans directories from `SHELL_ACTIONS_PATH`
2. it symlinks them into `~/.zmx/index/`
3. it scans shell functions and writes `~/.zmx/actions.db`
4. it generates `~/.zmx/import.sh`
5. it loads those action files and exposes discovery and execution commands

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

- `zmx.plugin.zsh`
  - main implementation
- `zmx-call.sh`
  - non-interactive executor
- `runner/lua/zmx.lua.sh`
  - experimental runner
- `legacy.sh`
  - older implementation notes and experiments
