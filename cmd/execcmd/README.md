# ccmodel exec Subsystem

`cmd/execcmd` 提供 `ccmodel exec` 命名空间下的全部子命令，实现 Claude / Codex CLI 的代理运行、tmux 托管、多实例恢复与日志监控。

## 架构总览

```
┌──────────────┐         tmux / direct          ┌──────────────┐
│ cobra root   │ ─────── run.go ─────────────→ │ target CLI   │
│ exec command │                               │ (claude/codex)│
└──────┬───────┘                                └─────┬─────────┘
       │                                             │
       │                session metadata             │ stdout/stderr
       ▼                                             ▼
┌──────────────┐    ┌────────────────────┐    ┌──────────────┐
│ run/resume   │ -> │ exec_sessions/*.json│ <- │ logging      │
│ watch/status │    │ + logs/             │    │ monitor.go   │
└──────────────┘    └────────────────────┘    └──────────────┘
```

- **run.go**：处理 `exec run` 及 legacy `exec <target>`，默认在 tmux 下创建窗口并写入日志；若 tmux 不可用则回落到直接执行。
- **session.go**：统一生成/保存 session JSON，记录目标、工作目录、tmux 窗口、日志文件等。
- **resume.go**：根据 session JSON 查找与目录匹配的 tmux 记录，按需重建窗口。
- **monitor.go**：实现 `watch` 与 `status`，聚合日志或输出 tmux/日志状态。
- **targets.go**：注册 Claude / Codex 目标名称与别名。

## Session 目录结构

所有记录存储于 `~/.claude/ccmodel/exec_sessions/`：

```bash
exec_sessions/
├── <session-id>.json    # 运行元数据
└── logs/
    └── <session-id>.log # tmux 管道输出
```

JSON 中新增字段：

| 字段          | 说明                                  |
|---------------|---------------------------------------|
| `run_mode`    | `tmux` 或 `direct`                    |
| `tmux_session`| 使用的 tmux 会话名（默认 `ccmodel-exec`） |
| `tmux_window` | 创建的窗口名                          |
| `log_file`    | 对应日志文件路径                      |

## 子命令说明

### `ccmodel exec run <target> [-- <args>]`

- 同目录窗口共用一个 session；不同目录自动使用不同 session（命名形如 `ccmodel-<basename>-<hash>`）。
- 目录已有 session 时会直接打开新窗口；若 session 尚未运行，会从历史记录里列出可恢复的窗口，交互式选择后再启动。
- 非交互终端会跳过恢复提示，直接新建窗口。
- 常用 flags：
  - `--dir DIR`：显式指定目录。
  - `--detach`：后台运行，不切换到新窗口。
  - `--tmux` / `--no-tmux`：显式开启/关闭 tmux（默认开启）。
  - `--name NAME`：自定义窗口名。

示例：

```bash
# 在当前目录的 session 中开新窗口（若无则自动创建）
ccmodel exec run codex -- --reset

# 指定目录运行并附带交互式恢复流程
ccmodel exec run --dir ~/workspace/foo codex
```

> 兼容路径：`ccmodel exec codex -- ...` 仍视为 `exec run codex`.

### `ccmodel exec resume`

- 仅恢复 `--dir`（默认当前目录）下的窗口，其他目录不受影响。恢复前无需手动停止现有 session，可直接运行。
- 输出类似 `docker-compose up` 的实时进度：`[.]` 启动中、`[✓]` 成功、`[-]` 已运行、`[!]` 失败。
- 恢复完成后给出汇总统计，并打印可复制的 `ccmodel exec attach <session>`；不会自动 attach（如需后台恢复可用 `--detach`）。
- 若该目录没有历史记录，会提示先通过 `exec run` 创建。

示例：

```bash
# 恢复当前目录下所有窗口，显示进度
ccmodel exec resume

# 指定目录恢复，并保持后台运行
ccmodel exec resume --dir ~/workspace/foo --detach
```

### `ccmodel exec watch`

聚合查看日志目录中的 `.log` 文件：

- `--mode auto|tail|multitail` 控制展示方式（默认 auto：优先 multitail）。
- `--simple` 为 tail 模式快捷方式。

示例：
```bash
# 自动选择输出模式
ccmodel exec watch
# 强制使用 tail
ccmodel exec watch --simple
```

### `ccmodel exec status [logs]`

- 默认：以树状视图列出所有已记录的 session。`●` 表示 `[running]`，`○` 表示 `[archived]`（仅存在历史记录）；窗口会显示 `[live]` 或 `[saved]`。
- `logs`：列出日志文件及最近修改时间。

示例：

```bash
ccmodel exec status
ccmodel exec status logs
```

### `ccmodel exec attach [session[:window]]`

- 支持 `session:window`、纯 session 名、纯窗口名或窗口索引；当 session 未运行时会拒绝并提示使用 `ccmodel exec resume --dir <path>` 先恢复。
- 仅展示正在运行的 session 供选择；非交互终端默认进入第一个窗口。
- 指定 `session:window` 时会直接切换至该窗口再附着。

## 环境变量

| 名称                           | 作用                                   |
|--------------------------------|----------------------------------------|
| `CCMODEL_EXEC_TMUX_SESSION`    | 覆盖默认 tmux 会话名 (`ccmodel-exec`) |
| `CCMODEL_EXEC_CLAUDE` / `_CODEX` | 覆盖对应 CLI 的二进制路径             |
| `TMUX`                         | 自动检测当前是否位于 tmux 内（决定 attach 行为） |

## 开发与测试提示

- 执行 `go test ./...` 验证直接模式等行为；tmux 集成依赖外部命令，测试中仅做路径注入与 JSON 验证。
- 新增 target（例如未来引入其他 CLI）时，只需在 `target_*.go` 中注册，`exec run` 等会自动识别。
- 如需进一步扩展（如 manifest、清理命令），可复用 `run.go` 与 `resume.go` 中的 session 读取与 tmux 帮助函数。
