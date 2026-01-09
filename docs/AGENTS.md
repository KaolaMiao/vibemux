# 多 Agent 配置指南 (Multi-Agent Configuration Guide)

VibeMux 本质上是一个多路终端复用器，它通过 **Profile (配置方案)** 来驱动不同的 AI Agent。只要你的 Agent 能在命令行中交互运行，就可以在 VibeMux 中使用。

## 核心概念

在 `profiles.json` (或通过 TUI 的 Profiles 界面) 中，每个 Profile 定义了一个 "驱动命令"。

- **Driver**: 通常设为 `native` (直接运行命令)。
- **Command**: Agent 的启动命令 (如 `claude`, `ollama run qwen`, `gh copilot alias`)。

## 配置示例

### 1. Claude Code (默认)

VibeMux 默认对 Claude 做了特殊优化 (如会话隔离)，通常无需额外配置。

```json
{
  "name": "Claude",
  "driver": "native",
  "command": "claude", 
  "auto_approve": "safe" 
}
```

### 2. Gemini (Google)

如果你安装了 Google 的 GenAI 工具或自定义的 CLI：

```json
{
  "name": "Gemini",
  "driver": "native",
  "command": "gemini-cli chat", 
  "env_vars": {
    "API_KEY": "your_api_key_here"
  }
}
```
*注：你需要先确保 `gemini-cli` 在你的 PATH 环境变量中。*

### 3. Qwen / Llama (via Ollama)

使用 [Ollama](https://ollama.com/) 运行本地模型是最佳组合：

```json
{
  "name": "Qwen 2.5",
  "driver": "native",
  "command": "ollama run qwen2.5:coder",
  "auto_approve": "none"
}
```
*提示：Ollama 的交互模式非常适合在 VibeMux 中运行。*

### 4. Codex (GitHub Copilot CLI)

使用 GitHub 官方 CLI：

```json
{
  "name": "Copilot",
  "driver": "native",
  "command": "gh copilot suggest -t shell", 
  "auto_approve": "none"
}
```

## 如何添加 Profile?

目前推荐直接编辑配置文件 (功能更完整)：

1. 打开 `C:\Users\Dota\.config\vibemux\data.json`
2. 在 `"profiles"` 数组中添加上述 JSON 对象。
3. 重启 VibeMux。
4. 在主界面按 `p` 进入 Profile 选择，或在创建 Project 时指定 Profile ID。

## 常见问题

**Q: 多个 Agent 会冲突吗？**
A: VibeMux 2.0+ (当前版本) 已经为 Claude 实现了自动隔离 (`CLAUDE_CONFIG_DIR`)。对于其他 Agent (如 Ollama)，它们通常自带会话管理或无状态，因此可以安全并行运行。

**Q: 如何让 Agent 自动执行命令？**
A: `auto_approve` 字段控制自动化级别。
- `none`: 也就是"人类模式"，不自动执行任何操作。
- `safe`: 仅自动批准无害的读取操作 (取决于 Agent 输出格式，目前主要针对 Claude 优化)。
- `yolo`: 自动批准所有操作 (慎用！)。
