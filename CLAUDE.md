# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

zmx is a zsh plugin that brings Emacs M-x style command execution to zsh. It allows users to discover and execute shell functions through an interactive fzf interface, similar to Emacs' M-x command palette.

## Core Architecture

### Action Discovery System

The plugin scans directories specified in `SHELL_ACTIONS_PATH` for shell functions and builds an indexed database:

1. **Indexing** (`_zmx_index_all_actions`): Creates symlinks in `~/.zmx/index/` for all action paths
2. **Database Building** (`_zmx_build_db`): Scans indexed files using ripgrep to find all functions matching `^function\s*([^\s()_]+)` and stores them in `~/.zmx/actions.db` with format: `function_name source_file line_number`
3. **Import Generation** (`_zmx_gen_import`): Creates `~/.zmx/import.sh` with source statements for all unique files
4. **MD5 Tracking** (`_zmx_gen_md5`): Generates MD5 checksums for each source file to track changes

### Key Bindings

- `,xm` - Opens fzf to select and execute a global action (from `SHELL_ACTIONS_PATH`)
- `,xx` - Opens fzf to select and execute a local action (from `./actions.sh` in current directory)

### Action Execution Flow

1. User presses `,xm` or `,xx`
2. fzf displays available actions from the database
3. User selects an action
4. Plugin checks if action requires arguments (looks for `arg-len` annotation in function definition)
5. If no args needed: executes immediately
6. If args needed: inserts function name into command line for user to add arguments

### Preexec Hook

`zmx_preexec` runs before each command execution. If the command is a zmx action, it automatically sources the action's file to ensure the latest version is loaded.

## Installation & Setup

Users install via zplug:
```zsh
zplug "woodgear/zmx"
export SHELL_ACTIONS_PATH=/path/to/actions
load_shell_actions $SHELL_ACTIONS_PATH
```

## Key Functions

- `zmx-reload-shell-actions` - Full rebuild of action database (slow, use when adding new action directories)
- `zmx-load-shell-actions` - Load actions from existing database (fast)
- `zmx-list` - List all available actions
- `zmx-help [function]` - Show which file defines a function
- `zmx-add-path <path>` - Add new action directory to SHELL_ACTIONS_PATH
- `mx-without-zle` - Execute action without zle (for non-interactive contexts)

## File Structure

- `zmx.plugin.zsh` - Main plugin file with all core functionality
- `zmx-call` / `zmx-call.sh` - Wrapper scripts that source user environment and execute actions
- `runner/lua/zmx.lua` - Lua-based action runner (experimental)
- `runner/ts/zmx.ts` - TypeScript-based action runner (experimental)
- `legacy.sh` - Commented-out legacy code for MD5-based dirty checking

## Development Notes

- Actions are shell functions defined in files under `SHELL_ACTIONS_PATH`
- Functions starting with `_` are considered private and not exposed
- Add `# arg-len` comment after function definition to indicate it requires arguments
- The plugin uses ripgrep for fast function discovery
- Database is stored in `~/.zmx/` directory
