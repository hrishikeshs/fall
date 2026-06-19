package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

		name := filepath.Base(abs)
		fmt.Fprintf(os.Stderr, "indexing %s... ", name)

		cmd := exec.Command(indexer, "-index", indexDir, abs)
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "FAILED (%v)\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "ok\n")
		}
	}
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
