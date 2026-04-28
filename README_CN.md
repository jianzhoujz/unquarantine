<div align="center">

# Unquarantine

**极速 macOS 隔离属性清理工具**

[![Go 版本](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![许可证](https://img.shields.io/badge/许可证-MIT-blue.svg)](LICENSE)
[![平台](https://img.shields.io/badge/平台-macOS-lightgrey.svg)](https://www.apple.com/macos)

[English](./README.md) · [报告问题](https://github.com/jianzhoujz/homebrew-tap/issues) · [功能建议](https://github.com/jianzhoujz/homebrew-tap/issues)

</div>

---

## ✨ 问题背景

macOS 会为下载的应用添加 `com.apple.quarantine` 扩展属性。对于未签名软件，这会触发那个著名的错误：

```
"App.app" 已损坏，无法打开。您应该将它移到废纸篓。
```

应用其实没有损坏 —— 它只是被隔离了。

### 传统解决方案

```bash
xattr -dr com.apple.quarantine /path/to/App.app
```

**但每次都要手动运行太麻烦了。** Unquarantine 自动化这个过程，并支持智能增量处理。

---

## 🚀 特性

| 特性 | 说明 |
|------|------|
| ⚡ **原生性能** | Go syscall 直接操作 xattr —— 无子进程开销 |
| 🔄 **智能增量** | 只处理新增或更新的应用，跳过未变化的 |
| 🔐 **自动提权** | 需要时自动请求 sudo —— 无需手动输入 `sudo` |
| 🎯 **双目录扫描** | 同时处理 `/Applications` 和 `~/Applications` |
| 🎨 **彩色输出** | 清晰直观的状态反馈 |
| 💾 **零 footprint** | 单一二进制文件，无运行时依赖 |

---

## 📦 安装

### Homebrew（推荐）

```bash
brew tap jianzhoujz/tap
brew install unquarantine
```

### 手动下载

<details>
<summary>Apple Silicon (M1/M2/M3)</summary>

```bash
curl -L https://github.com/jianzhoujz/homebrew-tap/releases/latest/download/unquarantine-darwin-arm64.tar.gz | tar xz
sudo mv unquarantine /usr/local/bin/
```
</details>

<details>
<summary>Intel Mac</summary>

```bash
curl -L https://github.com/jianzhoujz/homebrew-tap/releases/latest/download/unquarantine-darwin-amd64.tar.gz | tar xz
sudo mv unquarantine /usr/local/bin/
```
</details>

---

## 🎮 使用方法

```bash
# 增量模式（默认）—— 只处理新增或更新的应用
unquarantine

# 完整模式 —— 清空历史记录并重新处理所有应用
unquarantine --full
unquarantine -f

# 帮助
unquarantine -h
```

### 输出示例

```
Running as root (original user: zhou)

Full mode: clearing history and reprocessing all apps

Scanning: /Applications
  OK     WeChat.app
  SKIP   Safari.app (no quarantine)
  SKIP   Chrome.app (unchanged)
  FAIL   System Preferences.app

Scanning: /Users/zhou/Applications
  OK     MyApp.app
  SKIP   AnotherApp.app (no quarantine)

Results: 6 total, 2 OK, 3 SKIP, 1 FAIL
```

### 状态说明

| 状态 | 含义 |
|:----:|------|
| ✅ **OK** | 成功移除隔离属性 |
| ⏭️ **SKIP (no quarantine)** | 应用是干净的 —— 没有隔离属性 |
| ⏭️ **SKIP (unchanged)** | 已处理过，隔离属性哈希未变化 |
| ❌ **FAIL** | 权限不足或其他错误 |

---

## ⚙️ 工作原理

### 增量处理算法

```
┌─────────────────────────────────────────────────────────────┐
│                    遍历每个 .app 应用包                       │
├─────────────────────────────────────────────────────────────┤
│  1. 通过 syscall 读取 com.apple.quarantine（微秒级）          │
│  2. 如果无属性 → SKIP（应用是干净的）                          │
│  3. 计算属性内容的 MD5 哈希（约 50-100 字节）                  │
│  4. 与历史记录对比：                                          │
│     • 不在历史中 → 新增 → 处理                                │
│     • 哈希变化 → 已更新 → 处理                                │
│     • 哈希匹配 → 未变化 → 跳过                                │
│  5. 成功后：将哈希存入历史记录                                 │
└─────────────────────────────────────────────────────────────┘
```

### 为什么哈希属性内容？

隔离属性包含下载时间戳、来源 URL 等元数据：

```
0081;67612345;Google Chrome;12345678-1234-1234-1234-123456789012
```

当应用 **更新** 或 **重新下载** 时，这个内容会变化。通过哈希对比：

| 场景 | 检测方式 | 操作 |
|------|----------|------|
| 新应用 | 无历史记录 | ✅ 处理 |
| 更新的应用 | 哈希变化 | ✅ 处理 |
| 未变化的应用 | 哈希匹配 | ⏭️ 跳过 |
| 无隔离属性 | 属性不存在 | ⏭️ 跳过 |

**性能对比：**

| 方法 | 耗时 | 可靠性 |
|------|------|--------|
| 文件内容哈希 | 秒到分钟级 | ✅ 可靠 |
| 文件修改时间 | 毫秒级 | ❌ 不可靠 |
| **隔离属性哈希** | **微秒级** | ✅ **可靠** |

历史记录存储在 `~/.local/share/unquarantine/history.json`。

---

## 🔧 从源码构建

```bash
git clone https://github.com/jianzhoujz/unquarantine.git
cd unquarantine
go build -o unquarantine .
sudo mv unquarantine /usr/local/bin/
```

要求：Go 1.21+

---

## 📋 技术细节

<details>
<summary>点击展开</summary>

- **原生 xattr 操作** —— 通过 `golang.org/x/sys/unix`，无 `exec.Command` 开销
- **零外部依赖** —— 仅使用 Go 标准库
- **单一二进制** —— 无安装痕迹
- **Root 安全** —— 正确处理 `SUDO_USER` 获取真实用户目录
- **原子更新** —— 处理成功后才写入 JSON 文件

</details>

---

## 📄 许可证

采用 MIT 许可证。详见 `LICENSE` 文件。

---

<div align="center">

为 macOS 高级用户用心打造 ❤️

[⬆ 返回顶部](#unquarantine)

</div>