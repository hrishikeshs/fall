package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fall add <repo-path> [repo-path ...]\n")
		os.Exit(2)
	}

	indexer := findIndexer()
	if indexer == "" {
		fmt.Fprintf(os.Stderr, "fall: zoekt-git-index not found\n")
		fmt.Fprintf(os.Stderr, "Install it: go install github.com/sourcegraph/zoekt/cmd/zoekt-git-index@latest\n")
		os.Exit(1)
	}

	indexDir := defaultIndexDir()
	os.MkdirAll(indexDir, 0755)

	for _, path := range args {
		abs, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fall: %s: %v\n", path, err)
			continue
		}

		if _, err := os.Stat(filepath.Join(abs, ".git")); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "fall: %s: not a git repository\n", path)
			continue
		}

		trackRepo(abs)
		indexRepo(indexer, indexDir, abs)
	}
}

func indexRepo(indexer, indexDir, repoPath string) {
	name := filepath.Base(repoPath)
	fmt.Fprintf(os.Stderr, "indexing %s... ", name)

	cmd := exec.Command(indexer, "-index", indexDir, repoPath)
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAILED (%v)\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "ok\n")
	}
}

func reposFile() string {
	return filepath.Join(defaultIndexDir(), "repos")
}

func trackRepo(abs string) {
	for _, r := range loadTrackedRepos() {
		if r == abs {
			return
		}
	}
	f, err := os.OpenFile(reposFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, abs)
}

func loadTrackedRepos() []string {
	data, err := os.ReadFile(reposFile())
	if err != nil {
		return nil
	}
	var repos []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			repos = append(repos, line)
		}
	}
	return repos
}

func findIndexer() string {
	if p, err := exec.LookPath("zoekt-git-index"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	gopath := filepath.Join(home, "go", "bin", "zoekt-git-index")
	if _, err := os.Stat(gopath); err == nil {
		return gopath
	}
	return ""
}
