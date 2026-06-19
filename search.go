package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/sourcegraph/zoekt"
	"github.com/sourcegraph/zoekt/query"
	"github.com/sourcegraph/zoekt/search"
)

type SearchOpts struct {
	IndexDir    string
	MaxResults  int
	ContextLines int
}

func runSearch(pattern string, opts SearchOpts) ([]zoekt.FileMatch, error) {
	// Suppress zoekt's internal logging (shard loading info)
	log.SetOutput(io.Discard)

	searcher, err := search.NewDirectorySearcher(opts.IndexDir)
	if err != nil {
		return nil, fmt.Errorf("opening index at %s: %w", opts.IndexDir, err)
	}
	defer searcher.Close()

	q, err := query.Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("parsing query: %w", err)
	}
	q = query.Map(q, query.ExpandFileContent)
	q = query.Simplify(q)

	sOpts := &zoekt.SearchOptions{
		MaxDocDisplayCount: opts.MaxResults,
		NumContextLines:    opts.ContextLines,
	}

	result, err := searcher.Search(context.Background(), q, sOpts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return result.Files, nil
}
