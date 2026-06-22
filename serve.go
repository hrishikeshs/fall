package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const reindexInterval = 20 * time.Minute

func cmdServe(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fall serve <start|stop|status|restart>\n")
		os.Exit(2)
	}

	switch args[0] {
	case "start":
		serveStart()
	case "stop":
		serveStop()
	case "status":
		serveStatus()
	case "restart":
		serveStop()
		serveStart()
	default:
		fmt.Fprintf(os.Stderr, "Usage: fall serve <start|stop|status|restart>\n")
		os.Exit(2)
	}
}

// cmdDaemon is the hidden long-running process started by `fall serve start`.
// It runs zoekt-webserver as a child and re-indexes tracked repos every 20 minutes.
func cmdDaemon() {
	indexDir := defaultIndexDir()
	os.MkdirAll(indexDir, 0755)

	listen := os.Getenv("FALL_SERVE_LISTEN")
	if listen == "" {
		listen = ":6070"
	}

	// Start zoekt-webserver as a child
	server := findServer()
	if server == "" {
		fmt.Fprintf(os.Stderr, "fall: zoekt-webserver not found\n")
		os.Exit(1)
	}

	webCmd := exec.Command(server, "-listen", listen, "-index", indexDir, "-rpc")
	webCmd.Stdout = nil
	webCmd.Stderr = nil
	if err := webCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "fall: failed to start webserver: %v\n", err)
		os.Exit(1)
	}

	// Handle shutdown — kill webserver child on exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(reindexInterval)
	defer ticker.Stop()

	// Run initial reindex
	reindexAll(indexDir)

	for {
		select {
		case <-ticker.C:
			reindexAll(indexDir)
		case <-sigCh:
			webCmd.Process.Signal(syscall.SIGTERM)
			webCmd.Process.Wait()
			return
		}
	}
}

func reindexAll(indexDir string) {
	repos := loadTrackedRepos()
	if len(repos) == 0 {
		return
	}

	indexer := findIndexer()
	if indexer == "" {
		return
	}

	for _, repo := range repos {
		if _, err := os.Stat(filepath.Join(repo, ".git")); os.IsNotExist(err) {
			continue
		}
		cmd := exec.Command(indexer, "-index", indexDir, repo)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
}

func serveStart() {
	if isRunning() {
		pid, _ := readPID()
		fmt.Printf("already running (pid %d)\n", pid)
		return
	}

	indexDir := defaultIndexDir()
	os.MkdirAll(indexDir, 0755)

	listen := os.Getenv("FALL_SERVE_LISTEN")
	if listen == "" {
		listen = ":6070"
	}

	// Launch ourselves as the daemon
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fall: cannot find own executable: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command(exe, "_daemon")
	cmd.Env = append(os.Environ(), "FALL_SERVE_LISTEN="+listen)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "fall: failed to start daemon: %v\n", err)
		os.Exit(1)
	}

	writePID(cmd.Process.Pid)
	repos := loadTrackedRepos()
	fmt.Printf("started on %s (pid %d)\n", listen, cmd.Process.Pid)
	fmt.Printf("tracking %d repos, re-indexing every %v\n", len(repos), reindexInterval)
}

func serveStop() {
	pid, err := readPID()
	if err != nil || !isRunning() {
		removePID()
		fmt.Println("not running")
		return
	}

	proc, err := os.FindProcess(pid)
	if err == nil {
		proc.Signal(syscall.SIGTERM)
	}
	removePID()
	fmt.Printf("stopped (pid %d)\n", pid)
}

func serveStatus() {
	if isRunning() {
		pid, _ := readPID()
		indexDir := defaultIndexDir()
		shards, _ := filepath.Glob(filepath.Join(indexDir, "*.zoekt"))
		repos := loadTrackedRepos()
		fmt.Printf("running (pid %d)\n", pid)
		fmt.Printf("index: %s\n", indexDir)
		fmt.Printf("shards: %d\n", len(shards))
		fmt.Printf("tracking: %d repos (re-index every %v)\n", len(repos), reindexInterval)
	} else {
		fmt.Println("not running")
	}
}

func pidFile() string {
	return filepath.Join(defaultIndexDir(), "webserver.pid")
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFile())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writePID(pid int) {
	os.WriteFile(pidFile(), []byte(strconv.Itoa(pid)), 0644)
}

func removePID() {
	os.Remove(pidFile())
}

func isRunning() bool {
	pid, err := readPID()
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func findServer() string {
	if p, err := exec.LookPath("zoekt-webserver"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	gopath := filepath.Join(home, "go", "bin", "zoekt-webserver")
	if _, err := os.Stat(gopath); err == nil {
		return gopath
	}
	return ""
}
