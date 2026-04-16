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

### 1. Index action paths

`_zmx_index_all_actions` creates symlinks under `~/.zmx/index/` for every directory in `SHELL_ACTIONS_PATH`.

This gives zmx a stable local view of all configured action sources.

### 2. Build the action database

`_zmx_build_db` scans indexed files with ripgrep and writes `~/.zmx/actions.db`.

Database format:
```text
function_name source_file line_number
```

This file is the main registry for discovering and locating actions.

### 3. Generate imports

`_zmx_gen_import` builds `~/.zmx/import.sh`, which sources all unique action files.

This allows zmx to load the action set into the current shell.

### 4. Track source changes

`_zmx_gen_md5` stores checksums in `~/.zmx/md5/`.

This is auxiliary state used to track source changes. The current execution path mainly relies on re-sourcing action files before execution.

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
2. sources `~/.zmx/import.sh`
3. evaluates the requested action command

This is what other repos often call when they want access to the environment's action objects without recreating the indexing logic themselves.

## Preexec Behavior

`zmx_preexec` checks whether the command being executed is an indexed action.

If it is, zmx re-sources the action's source file before execution so the latest definition is used.

This matters because action files are edited frequently and are shared across multiple repositories.

## Installation & Setup

Typical setup:

```zsh
zplug "woodgear/zmx"
export SHELL_ACTIONS_PATH=/path/to/actions:/another/path
zmx-reload-shell-actions
```

After the initial build, normal shell startup can use:

```zsh
zmx-load-shell-actions
```

## File Structure

- `zmx.plugin.zsh`
  - main implementation
- `zmx-call.sh`
  - non-interactive action executor
- `runner/lua/zmx.lua.sh`
  - experimental runner
- `legacy.sh`
  - older code and experiments

## Development Notes

- action discovery is based on shell function definitions, not a separate manifest
- functions starting with `_` are private and not exposed
- `arg-len` annotations are used to mark actions that require arguments
- `~/.zmx/actions.db` is the central action registry
- `~/.zmx/import.sh` is the generated import layer
- when documenting zmx, focus on action discovery, inspection, path management, and execution flow
