# shellargs

## 项目目的

`shellargs` 是一个基于 Go 的命令行工具，用来把一段声明式参数 spec 和一组 `argv` 交给 `go-flags` 解析，最后输出 JSON。

这个项目的目标是服务 shell 函数场景：

- 在 bash/zsh 函数里用一个字符串描述参数
- 不自己重写参数解析器
- 让 shell 代码通过 `jq` 等工具消费解析结果
- 可选自动处理 `--help`
- 生成 bash completion 脚本
- 为 Go 调用方提供 zsh completion 生成库函数

## 当前实现了什么

当前版本已经支持：

- `parse`
  - 输入 spec 和 argv
  - 用 `go-flags` 解析参数
  - 输出 JSON
  - 可通过 `--auto-help` 让底层库接管 `--help`
- `help`
  - 根据 spec 渲染帮助信息
- `is-help`
  - 判断目标 `argv` 是否请求帮助
- `completion`
  - 生成 bash completion 脚本
  - 通过 `GO_FLAGS_COMPLETION=1` 走 `go-flags` 内建 completion
- Go library
  - `shellargs.ParseSpec`
  - `shellargs.ZshCompletionScript`
  - 当前由 zmx reload 直接调用，用 action 的 `@@@ ... @@@` spec 生成 zsh completion 文件，不经过 `shellargs` CLI
- spec 装载时会自动识别 `@@@ ... @@@` 包裹块并裁掉首尾标记行

当前 spec 支持三类字段：

- `flag`
- `option`
- `arg`

支持的属性：

- `short=x`
- `long=name`
- `type=string|int|int64|uint|float64|duration|file`
- `default=...`
- `desc=...`
- `required`
- `required=yes|2|1-3`
- `repeatable`
- `placeholder=NAME`

## 核心设计

这个项目没有自己实现命令行解析逻辑，而是做了一层很薄的适配：

1. 把 line-based spec 解析成内部 `Spec`
2. 运行时通过 `reflect.StructOf` 动态生成 struct
3. 给字段打上 `go-flags` tag
4. 调用 `go-flags` 完成解析、help 和 completion
5. 再把结果展平成 JSON map 输出

位置参数使用 `go-flags` 的 `positional-args` 能力，所以支持：

```bash
xx 1 2 3
```

而不要求全部写成：

```bash
xx --a 1 --b 2
```

## 目录结构

```text
project/shellargs/
├── cmd/shellargs/main.go         # CLI 入口，包含 parse/help/completion 子命令
├── internal/spec/spec.go         # spec 文本解析与校验
├── internal/engine/engine.go     # 动态 struct、go-flags 绑定、JSON/help/completion
├── internal/spec/spec_test.go    # spec 解析测试
├── internal/engine/engine_test.go
├── README.md
└── go.mod
```

## 常用命令

在项目目录下：

```bash
GOPROXY=https://goproxy.cn,direct GOSUMDB=off go test ./...
GOPROXY=https://goproxy.cn,direct GOSUMDB=off go build -o bin/shellargs ./cmd/shellargs
```

示例：

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

./bin/shellargs parse --auto-help --spec "$SPEC" -- --branch dev demo p1 p2
./bin/shellargs is-help -- "$@"
./bin/shellargs help --spec "$SPEC"
./bin/shellargs help --spec "$SPEC_DOC"
./bin/shellargs completion --shell bash --runner ./bin/shellargs --spec "$SPEC"
```

## 当前边界

当前还没有做这些能力：

- subcommand DSL
- CLI zsh/fish completion
- 描述中的 `|` 转义
- repeatable flag
- 更复杂的类型系统
- 面向 bash 的更短别名封装

## 修改时的约定

- 优先继续复用 `go-flags`，不要把项目演化成自研解析器
- shell 场景是第一优先级，DSL 不要为了“通用性”变得难写
- 如果扩展 spec，先保证：
  - shell heredoc 中容易写
  - 帮助信息仍然清晰
  - JSON 输出字段稳定
- 如果改 completion，保持当前 `GO_FLAGS_COMPLETION=1` 的接入模型
