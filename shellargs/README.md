# shellargs

`shellargs` 把一段声明式参数 spec 和一组 `argv` 交给 `go-flags`，输出 JSON，适合在 bash/zsh 函数里作为统一的参数解析层。

## 目标

- 不自己重写命令行参数解析器
- 在 shell 函数里用一个字符串描述参数
- 输出 JSON，后续直接配合 `jq`
- 可选自动处理 `--help`
- 可判断目标 `argv` 是否是在请求帮助
- 生成 bash completion 脚本
- 为 Go 调用方提供 zsh completion 生成库函数

## Spec 语法

Spec 是一个多行字符串。

- 元数据行：`name: ...`、`summary: ...`、`description: ...`、`usage: ...`
- 字段行：`<kind> <name> | attr=value | attr | ...`
- `kind` 支持 `flag`、`option`、`arg`
- `name` 是 JSON key；如果是 `option/flag`，默认 long name 会从 `name` 推导
- 行内 `|` 当前不支持转义，描述里不要写 `|`
- 如果 spec 的首尾非空行都是 `@@@`，shellargs 会自动去掉这两行再解析

### 支持的属性

- `short=x`
- `long=dry-run`
- `type=string|int|int64|uint|float64|duration|file`
- `default=...`
- `desc=...`
- `required`
- `required=yes|2|1-3`
- `repeatable`
- `placeholder=NAME`

## 示例

```bash
SPEC='
name: repo-sync
summary: Sync git repositories

flag dry_run | short=n | long=dry-run | desc=Dry run only
option branch | short=b | default=main | desc=Branch name
option retry | short=r | type=int | default=3 | desc=Retry count
arg repo | required | desc=Repository name
arg path | repeatable | desc=Extra paths
'
```

### 解析成 JSON

```bash
shellargs parse --auto-help --spec "$SPEC" -- --branch dev --retry 5 --dry-run demo p1 p2
```

输出：

```json
{
  "branch": "dev",
  "dry_run": true,
  "path": ["p1", "p2"],
  "repo": "demo",
  "retry": 5
}
```

### 直接在函数里用

```bash
repo-sync() {
  local spec_doc='
@@@
name: repo-sync
summary: Sync git repositories

flag dry_run | short=n | long=dry-run | desc=Dry run only
option branch | short=b | default=main | desc=Branch name
option retry | short=r | type=int | default=3 | desc=Retry count
arg repo | required | desc=Repository name
arg path | repeatable | desc=Extra paths
@@@
'

  if shellargs is-help -- "$@"; then
    shellargs help --spec "$spec_doc"
    return
  fi

  local parsed="$(shellargs parse --spec "$spec_doc" -- "$@")" || return

  local repo="$(jq -r '.repo' <<<"$parsed")"
  local branch="$(jq -r '.branch' <<<"$parsed")"
  local retry="$(jq -r '.retry' <<<"$parsed")"
  local dry_run="$(jq -r '.dry_run' <<<"$parsed")"

  printf 'repo=%s branch=%s retry=%s dry_run=%s\n' "$repo" "$branch" "$retry" "$dry_run"
}
```

### 检查是否在请求帮助

```bash
shellargs is-help -- "$@"
```

### 生成帮助

```bash
shellargs help --spec "$SPEC"
shellargs help --spec "$SPEC_DOC"
```

### 生成 bash completion

```bash
shellargs completion --shell bash --prog repo-sync --runner shellargs --spec "$SPEC"
```

可以重定向成文件再 `source`：

```bash
shellargs completion --shell bash --prog repo-sync --runner shellargs --spec "$SPEC" > repo-sync.completion.bash
source ./repo-sync.completion.bash
```

### Go 库生成 zsh completion

`zmx reload` 直接调用 `shellargs.ParseSpec` 和 `shellargs.ZshCompletionScript`，在 reload 阶段从 action 的 `@@@ ... @@@` spec 生成 zsh completion 文件，不依赖 `shellargs` CLI。

## 当前边界

- CLI 当前只生成 bash completion
- zsh completion 目前作为 Go 库能力提供给 zmx reload 使用
- 不支持 subcommand DSL
- 不支持描述里的 `|`
- 不支持 repeatable flag
