package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// History records the quarantine attribute hash of processed apps
type History struct {
	Apps map[string]string `json:"apps"` // path -> quarantine attr hash
}

func main() {
	fullMode := flag.Bool("full", false, "Process all apps, clear history and reprocess")
	flag.BoolVar(fullMode, "f", false, "Process all apps, clear history and reprocess (shorthand)")
	flag.Parse()

	// Auto-elevate if not root
	if os.Geteuid() != 0 {
		reexecWithSudo()
		return
	}

	run(*fullMode)
}

func reexecWithSudo() {
	fmt.Println("Need admin privileges to process system apps")
	fmt.Println("Requesting elevation...")

	cmd := exec.Command("sudo", append([]string{os.Args[0]}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\n%sFailed to elevate privileges: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

func run(fullMode bool) {
	origUser := os.Getenv("SUDO_USER")
	if origUser == "" {
		origUser = os.Getenv("USER")
	}
	fmt.Printf("\n%sRunning as root (original user: %s)%s\n\n", colorCyan, origUser, colorReset)

	historyFile := getHistoryFile(origUser)
	history := loadHistory(historyFile)

	if fullMode {
		fmt.Printf("%sFull mode: clearing history and reprocessing all apps%s\n\n", colorYellow, colorReset)
		history = &History{Apps: make(map[string]string)}
	}

	scanDirs := getScanDirs(origUser)

	total := 0
	ok := 0
	skipNoQuarantine := 0
	skipUnchanged := 0
	fail := 0

	for _, dir := range scanDirs {
		apps, err := filepath.Glob(filepath.Join(dir, "*.app"))
		if err != nil {
			continue
		}

		if len(apps) == 0 {
			continue
		}

		fmt.Printf("Scanning: %s\n", dir)

		for _, app := range apps {
			total++
			name := filepath.Base(app)
			status, hash := processApp(app, history)

			switch status {
			case "OK":
				ok++
				fmt.Printf("  %sOK%s     %s\n", colorGreen, colorReset, name)
				history.Apps[app] = hash
			case "SKIP_NO_QUARANTINE":
				skipNoQuarantine++
				fmt.Printf("  %sSKIP%s   %s (no quarantine)\n", colorGray, colorReset, name)
			case "SKIP_UNCHANGED":
				skipUnchanged++
				fmt.Printf("  %sSKIP%s   %s (unchanged)\n", colorGray, colorReset, name)
			case "FAIL":
				fail++
				fmt.Printf("  %sFAIL%s   %s\n", colorRed, colorReset, name)
			}
		}
		fmt.Println()
	}

	saveHistory(historyFile, history)

	fmt.Printf("%sResults: %d total, %d OK, %d SKIP (no quarantine), %d SKIP (unchanged), %d FAIL%s\n",
		colorCyan, total, ok, skipNoQuarantine, skipUnchanged, fail, colorReset)
}

func getScanDirs(username string) []string {
	var dirs []string

	// System Applications
	dirs = append(dirs, "/Applications")

	// User Applications
	if username != "" {
		if u, err := user.Lookup(username); err == nil {
			dirs = append(dirs, filepath.Join(u.HomeDir, "Applications"))
		}
	}

	// Fallback to current user's home
	if len(dirs) == 1 {
		if home, err := os.UserHomeDir(); err == nil {
			userApps := filepath.Join(home, "Applications")
			if userApps != "/Applications" {
				dirs = append(dirs, userApps)
			}
		}
	}

	return dirs
}

func getHistoryFile(username string) string {
	var dataDir string

	if username != "" {
		if u, err := user.Lookup(username); err == nil {
			dataDir = filepath.Join(u.HomeDir, ".local", "share", "unquarantine")
		}
	}

	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share", "unquarantine")
	}

	os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, "history.json")
}

func loadHistory(path string) *History {
	h := &History{Apps: make(map[string]string)}

	data, err := os.ReadFile(path)
	if err != nil {
		return h
	}

	json.Unmarshal(data, h)
	if h.Apps == nil {
		h.Apps = make(map[string]string)
	}

	return h
}

func saveHistory(path string, h *History) {
	data, _ := json.MarshalIndent(h, "", "  ")
	os.WriteFile(path, data, 0644)
}

// getQuarantineAttr reads the quarantine attribute using unix.Getxattr (fast, no subprocess)
func getQuarantineAttr(path string) ([]byte, error) {
	// First get the size
	size, err := unix.Getxattr(path, "com.apple.quarantine", nil)
	if err != nil {
		return nil, err
	}

	if size == 0 {
		return []byte{}, nil
	}

	buf := make([]byte, size)
	_, err = unix.Getxattr(path, "com.apple.quarantine", buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// hashQuarantineAttr computes MD5 hash of quarantine attribute content
func hashQuarantineAttr(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// processApp processes a single app and returns status and hash
func processApp(appPath string, history *History) (string, string) {
	// Read quarantine attribute
	attr, err := getQuarantineAttr(appPath)
	if err != nil {
		// No quarantine attribute - app is clean
		return "SKIP_NO_QUARANTINE", ""
	}

	currentHash := hashQuarantineAttr(attr)

	// Check history
	if storedHash, exists := history.Apps[appPath]; exists {
		if storedHash == currentHash {
			// Same hash - already processed, no change
			return "SKIP_UNCHANGED", ""
		}
		// Hash changed - app was updated, need to reprocess
	}

	// Remove quarantine attribute
	if err := unix.Removexattr(appPath, "com.apple.quarantine"); err != nil {
		return "FAIL", ""
	}

	return "OK", currentHash
}