# VibeMux 项目上下文 (Project Context)

## 项目概述
**VibeMux** 是一个基于 TUI (终端用户界面) 的 AI 智能体编排终端，专为 "Vibe Coding" 设计。它允许开发者并行管理、监控和操作多个 AI 智能体会话（特别是 **Claude Code** 和 **Codex**），类似于 `lazydocker` 或 `k9s` 的操作体验。

## 技术栈 (Tech Stack)
- **开发语言**: Go 1.25+
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (基于 Elm 架构)
- **样式定制**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **交互组件**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **终端模拟**: [creack/pty](https://github.com/creack/pty)
- **数据存储**: 标准 `encoding/json`

## 系统架构
VibeMux 遵循 **Model-View-Update (MVU)** 架构：
1.  **数据层 (Data Layer)** (`internal/store`): 管理项目 (Projects)、配置方案 (Profiles) 及持久化。
2.  **运行时层 (Runtime Layer)** (`internal/runtime`): 处理 PTY 生成、进程管理和 I/O 转发，是程序的核心引擎。
3.  **表现层 (Presentation Layer)** (`internal/ui`): 处理 TUI 渲染和用户输入路由。

### 核心概念
- **项目 (Project)**: 代表受智能体管理的本地代码仓库。
- **配置方案 (Profile)**: 定义运行环境（驱动程序、API 密钥、环境变量）。通过 `CLAUDE_CONFIG_DIR` 实现会话隔离的关键。
- **驱动 (Driver)**: 执行策略（`native` 直接调用 CLI，`ccr` 调用代理/封装器，`custom` 执行自定义命令）。
- **自动确认 (Auto-Approve)**: 可配置的自动化级别（`none`、`safe`、`vibe`、`yolo`），用于自动处理智能体的交互提示。

## 目录结构
- `main.go`: 程序入口。
- `internal/`: 私有业务逻辑。
    - `app/`: 全局配置与启动逻辑。
    - `runtime/`: PTY 与进程管理逻辑（核心引擎）。
    - `store/`: JSON 持久化逻辑。
    - `ui/`: Bubble Tea 模型与视图。
- `docs/`: 设计文档与规范 (`DESIGN.md`)。

## 构建与运行
项目使用 `Makefile` 进行常用任务管理：
- **编译**: `make build` (输出至 `bin/vibemux`)
- **开发运行**: `make run` 或 `go run .`
- **测试**: `make test`
- **代码规范检查**: `make lint`
- **清理**: `make clean`

## 配置信息
配置文件存储在 `~/.config/vibemux/`：
- `config.json`: 通用设置（网格大小等）。
- `projects.json`: 受控项目列表。
- `profiles.json`: 配置方案定义。

## 开发原则
1.  **非侵入性**: 不修改全局用户配置，通过环境变量注入实现隔离。
2.  **异步 UI**: UI 渲染必须与 PTY I/O 解耦，确保大量输出时界面不卡顿。
3.  **基于 Profile 的隔离**: 严格遵循 Profile 设置以确保环境独立。
4.  **编码规范**: 遵循 Go 语言惯例及 Bubble Tea 的 MVU 模式。

## 相关文档
- 详细架构规格及功能规划请参考 `docs/DESIGN.md`。
- 快速上下文摘要请参考 `CLAUDE.md`。