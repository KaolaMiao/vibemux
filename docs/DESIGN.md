# Design Specification: lazyvibe

## 1. 系统架构概览 (Architecture Overview)

Lazyvibe 遵循典型的 **Model-View-Update (ELM)** 架构，基于 Bubble Tea 框架。核心逻辑分为三层：

1.  **数据层 (Data Layer)**: 管理项目列表、配置文件 (Profiles) 和持久化存储。
2.  **运行时层 (Runtime Layer)**: 负责 PTY (伪终端) 的生成、进程管理、信号处理和 I/O 转发。
3.  **表现层 (Presentation Layer)**: TUI 界面渲染，处理用户键盘输入并路由到对应的逻辑层。

## 2. 核心数据模型 (Data Models)

### 2.1 Project (项目)
代表一个被管理的本地代码仓库。

```go
type Project struct {
    ID        string `json:"id"`
    Name      string `json:"name"`      // 显示名称，默认为文件夹名
    Path      string `json:"path"`      // 绝对路径
    ProfileID string `json:"profile_id"`// 绑定的配置 ID
    LastUsed  int64  `json:"last_used"` // 时间戳
}

```

### 2.2 Profile (配置描述文件)

核心组件。定义了 Claude 运行时的“上下文环境”。通过组合不同的 Profile，用户可以在同一台机器上隔离出不同的“人格”或环境。

```go
type DriverType string

const (
    DriverNative DriverType = "native" // 直接调用 claude
    DriverCCR    DriverType = "ccr"    // 调用 ccr (Claude Code Runner)
    DriverCustom DriverType = "custom" // 自定义命令
)

type Profile struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`        // 例如: "Work-Strict", "Personal-Haiku"
    Driver      DriverType        `json:"driver"`      // 启动方式
    CommandArgs []string          `json:"command_args"`// 自定义参数 (如 ccr 的 flag)
    EnvVars     map[string]string `json:"env_vars"`    // 注入的环境变量
    AutoApprove  AutoApproveLevel   `json:"auto_approve"`
    Notification NotificationConfig `json:"notification"`
}
const (
ApproveNone  AutoApproveLevel = "none"   // 全手动 
ApproveSafe  AutoApproveLevel = "safe"   // 自动确认读文件、运行测试
ApproveVibe  AutoApproveLevel = "vibe"   // 自动确认写文件、执行安装 (默认)
ApproveYolo  AutoApproveLevel = "yolo"   // 自动确认所有 Shell 命令 (高危)
)

type NotificationConfig struct {
DesktopNotify bool   `json:"desktop_notify"` // 是否发送系统桌面通知 (默认)
SoundAlert    bool   `json:"sound_alert"`    // 是否播放提示音 (默认)
WebhookURL    string `json:"webhook_url"`    // 比如 Slack/飞书/钉钉 Hook
}

```

> **关键设计**: `EnvVars` 必须包含 `CLAUDE_CONFIG_DIR` 的支持，以便为每个 Profile 指定独立的配置文件夹（存放 session token 和 history），实现完全隔离。

### 2.3 Store (持久化)

数据存储在 `~/.config/lazyvibe/` 目录下：

* `projects.json`: 存储 `[]Project`
* `profiles.json`: 存储 `[]Profile`

---

## 3. 功能模块详解 (Feature Specs)

### 3.1 运行时引擎 (Runtime Engine)

这是 lazyvibe 的心脏。它不直接运行 shell 命令，而是通过 **PTY (Pseudo-Terminal)** 启动进程。

* **启动流程**:
1. 用户在 UI 选中 Project A。
2. 读取 Project A 关联的 Profile。
3. 构建 `exec.Cmd`:
* `Dir`: Project A 的 Path。
* `Env`: 系统 Env + Profile.EnvVars。
* `Command`: 根据 Driver 决定 (e.g., `claude` 或 `ccr`).


4. 使用 `pty.Start(cmd)` 启动进程。
5. 启动一个 Goroutine 持续读取 PTY 的 `stdout/stderr`，并通过 `tea.Msg` 发送给 UI 更新。


* **交互逻辑**:
* 当 Focus 在终端窗口时，用户的按键（包括 `Ctrl+C`, `Enter`）直接写入 PTY 的 `stdin`。
* 这保证了 Claude Code 的交互式提问（如 "Do you want to run this? [y/N]"）可以被用户正常操作。



### 3.2 多驱动支持 (Driver System)

* **Native Driver**:
* Command: `claude`
* 适用: 标准环境，用户已在全局安装了 `claude-code`。


* **CCR Driver**:
* Command: `ccr`
* 适用: 需要通过 API 代理、或者使用特定模型参数启动的场景。
* Lazyvibe 自动处理 `ccr` 需要的特殊环境变量。


* **Custom Driver**:
* 允许用户输入任意 Shell 字符串，例如 `docker run -it -v $(pwd):/app claude-container`。



### 3.3 配置文件注入 (Profile Injection)

为了实现“零配置”但“高隔离”，默认逻辑如下：

1. 如果用户未创建任何 Profile，使用 **Default Profile** (继承当前 Shell 所有变量)。
2. 当用户创建 Profile 时，UI 提供字段输入：
* `ANTHROPIC_API_KEY`: (Masked display)
* `CLAUDE_CONFIG_DIR`: 建议默认为 `~/.config/lazyvibe/claude_configs/<profile_name>`。这样可以确保不同 Profile 的登录状态互不干扰。




### 3.4 自动化交互系统 (Automation System)
为了实现“无人值守”或减少重复点击，Runtime 需要具备 PTY 输出流分析能力。

* **输出监听器 (Output Watcher)**:
  后台 Goroutine 实时分析 PTY 的 `Stdout`。

* **模式匹配与自动应答 (Pattern & Action)**:
  根据 `Profile.AutoApprove` 的级别，匹配 Claude 的常见提问模式并自动写入 `Stdin`。

  | 检测模式 (Regex) | 动作 | 适用级别 |
      | :--- | :--- | :--- |
  | `Allow .* to read files? \[y/N\]` | Write `y\n` | Safe, Vibe, Yolo |
  | `Allow .* to edit files? \[y/N\]` | Write `y\n` | Vibe, Yolo |
  | `Execute command .*? \[y/N\]` | Write `y\n` | Yolo (Only) |

* **安全熔断**:
  即使在 YOLO 模式下，如果检测到 `rm -rf /` 或敏感操作，应强制暂停并弹窗请求人工介入。


### 3.5 通知钩子模块 (Notification Hooks)
Lazyvibe 需要感知 Claude 的工作状态变化，并触发通知。

* **状态检测逻辑**:
  通过分析 PTY 输出判定状态流转：
    * `Input Required`: 当输出流停止且最后一行匹配 `[y/N]` 或 `>` 时。
    * `Task Completed`: 当检测到 `Cost: $0.xx` 或 `Task finished` 关键字时。
    * `Error`: 当检测到 `Error:` 或 `Context window exceeded` 时。

* **通知渠道**:
    1.  **System Native**: 使用 `gen2brain/beeep` 库发送 macOS/Windows/Linux 原生通知。
    2.  **Webhook**: 发送 JSON Payload 到用户配置的 URL (支持自定义 Header)。

* **Payload 示例**:
    ```json
    {
      "project": "backend-api",
      "event": "input_required",
      "message": "Claude needs your approval to delete main.go",
      "timestamp": 1712345678
    }
    ```
---

## 4. UI/UX 设计 (Interface Design)

界面分为三个主要面板 (Pane)，支持 Tab 键循环切换焦点。

```text
+---------------------+------------------------------------------+
|  PROJECTS (List)    |  TERMINAL (Viewport)                     |
|                     |                                          |
| > [My-Backend]      |  $ claude                                |
|   * Profile: Work   |  > Reading context...                    |
|   * Status: Idle    |  > Planning changes...                   |
|                     |                                          |
|   [My-Frontend]     |  [ Claude is thinking... ]               |
|   * Profile: Vibe   |                                          |
|   * Status: Running |                                          |
+---------------------+                                          |
|  PROFILES (List)    |                                          |
| > [Work-Strict]     |                                          |
|   [Personal]        |                                          |
+---------------------+------------------------------------------+
|  <Tab> Switch Pane | <R> Run | <Q> Quit | <C> Config Profile   |
+---------------------+------------------------------------------+

```

### 4.1 快捷键定义

* **全局**:
* `Tab`: 在 Projects / Profiles / Terminal 之间切换焦点。
* `Ctrl+c`: 退出程序（如果在 Terminal 焦点下，则发送给 Claude 进程）。


* **Projects 面板**:
* `a`: 添加本地项目 (Add)。
* `Enter`: 启动/连接当前项目的 Claude 实例。
* `d`: 移除项目 (Delete)。
* `p`: 绑定/切换 Profile。


* **Profiles 面板**:
* `n`: 新建 Profile。
* `e`: 编辑选中的 Profile (API Key, Env vars)。



---

## 5. 开发阶段规划 (Development Phases)

### Phase 1: 骨架与 PTY 原型

* 实现 `Project` 和 `Profile` 的基础增删改查。
* 集成 `creack/pty`，实现“选中项目 -> 启动 claude -> 在右侧显示输出”的最小闭环。

### Phase 2: 配置注入与隔离

* 实现 `EnvVars` 的合并逻辑。
* 验证 `CLAUDE_CONFIG_DIR` 是否生效（即两个不同 Profile 的项目是否需要分别登录）。

### Phase 3: 多实例管理

* 实现后台进程池。切换 UI 上的项目选择时，不要杀死之前的进程，而是将 PTY 输出流“后台化”，切回来时恢复显示。

### Phase 4: UI 美化

* 使用 `lipgloss` 添加边框、颜色和状态指示灯（🟢 Running, ⚪ Idle, 🔴 Error）。
