package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/sourcegraph/zoekt"
	"github.com/sourcegraph/zoekt/query"
	"github.com/sourcegraph/zoekt/search"
)

func cmdList() {
	log.SetOutput(io.Discard)
	indexDir := defaultIndexDir()

	searcher, err := search.NewDirectorySearcher(indexDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fall: no index at %s\n", indexDir)
		fmt.Fprintf(os.Stderr, "Run: fall add <repo-path>\n")
		os.Exit(1)
	}
	defer searcher.Close()

	repos, err := searcher.List(context.Background(), &query.Const{Value: true}, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fall: %v\n", err)
		os.Exit(1)
	}

	for _, r := range repos.Repos {
		name := r.Repository.Name
		short := name
		if i := strings.LastIndex(name, "/"); i >= 0 {
			short = name[i+1:]
		}
		fmt.Printf("%-20s %s (%d files, %d branches)\n",
			short, name, r.Stats.Documents, countBranches(r.Repository))
	}
}

func countBranches(repo zoekt.Repository) int {
	return len(repo.Branches)
}
