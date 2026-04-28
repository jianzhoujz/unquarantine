<div align="center">

# Unquarantine

**Blazing-fast macOS quarantine attribute remover**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS-lightgrey.svg)](https://www.apple.com/macos)

[中文文档](./README_CN.md) · [Report Bug](https://github.com/jianzhoujz/homebrew-tap/issues) · [Request Feature](https://github.com/jianzhoujz/homebrew-tap/issues)

</div>

---

## ✨ The Problem

macOS marks downloaded applications with a `com.apple.quarantine` extended attribute. For unsigned software, this triggers the infamous error:

```
"App.app" is damaged and can't be opened. You should move it to the Bin.
```

The app isn't actually damaged — it's just quarantined.

### The Traditional Fix

```bash
xattr -dr com.apple.quarantine /path/to/App.app
```

**But running this manually for every app is tedious.** Unquarantine automates this with intelligent incremental processing.

---

## 🚀 Features

| Feature | Description |
|---------|-------------|
| ⚡ **Native Performance** | Go syscall for xattr operations — no subprocess overhead |
| 🔄 **Smart Incremental** | Only processes new or updated apps, skips unchanged ones |
| 🔐 **Auto-Elevation** | Automatically requests sudo when needed — no manual `sudo` required |
| 🎯 **Dual Scan** | Processes both `/Applications` and `~/Applications` |
| 🎨 **Colored Output** | Clear visual feedback with distinct status indicators |
| 💾 **Zero Footprint** | Single binary, no runtime dependencies |

---

## 📦 Installation

### Homebrew (Recommended)

```bash
brew tap jianzhoujz/tap
brew install unquarantine
```

### Manual Download

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

## 🎮 Usage

```bash
# Incremental mode (default) — process only new or updated apps
unquarantine

# Full mode — clear history and reprocess everything
unquarantine --full
unquarantine -f

# Help
unquarantine -h
```

### Output Example

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

### Status Indicators

| Status | Meaning |
|:------:|---------|
| ✅ **OK** | Quarantine attribute successfully removed |
| ⏭️ **SKIP (no quarantine)** | App is clean — no quarantine attribute present |
| ⏭️ **SKIP (unchanged)** | Already processed, quarantine hash unchanged |
| ❌ **FAIL** | Permission denied or other error |

---

## ⚙️ How It Works

### Incremental Processing Algorithm

```
┌─────────────────────────────────────────────────────────────┐
│                     For each .app bundle                     │
├─────────────────────────────────────────────────────────────┤
│  1. Read com.apple.quarantine via syscall (microseconds)    │
│  2. If no attribute → SKIP (app is clean)                   │
│  3. Compute MD5 hash of attribute content (~50-100 bytes)  │
│  4. Compare with history:                                   │
│     • Not in history → NEW → Process                        │
│     • Hash changed → UPDATED → Process                      │
│     • Hash matches → UNCHANGED → Skip                       │
│  5. On success: store hash in history                       │
└─────────────────────────────────────────────────────────────┘
```

### Why Hash the Attribute Content?

The quarantine attribute contains metadata like download timestamp and source:

```
0081;67612345;Google Chrome;12345678-1234-1234-1234-123456789012
```

When an app is **updated** or **re-downloaded**, this content changes. By hashing it:

| Scenario | Detection | Action |
|----------|-----------|--------|
| New app | No history entry | ✅ Process |
| Updated app | Hash changed | ✅ Process |
| Unchanged app | Hash matches | ⏭️ Skip |
| No quarantine | Attribute absent | ⏭️ Skip |

**Performance comparison:**

| Method | Time | Reliability |
|--------|------|-------------|
| File content hash | Seconds to minutes | ✅ Reliable |
| File modification time | Milliseconds | ❌ Unreliable |
| **Quarantine attribute hash** | **Microseconds** | ✅ **Reliable** |

History is stored at `~/.local/share/unquarantine/history.json`.

---

## 🔧 Build from Source

```bash
git clone https://github.com/jianzhoujz/unquarantine.git
cd unquarantine
go build -o unquarantine .
sudo mv unquarantine /usr/local/bin/
```

Requirements: Go 1.21+

---

## 📋 Technical Details

<details>
<summary>Click to expand</summary>

- **Native xattr operations** via `golang.org/x/sys/unix` — no `exec.Command` overhead
- **Zero dependencies** beyond Go standard library
- **Single binary** — no installation footprint
- **Root-safe** — respects `SUDO_USER` for home directory resolution
- **Atomic history updates** — JSON file written after successful processing

</details>

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---

<div align="center">

Made with ❤️ for macOS power users

[⬆ Back to Top](#unquarantine)

</div>