package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	colors := flag.Bool("colors", false, "ANSI color highlighting for human use")
	jsonOut := flag.Bool("json", false, "JSONL output (one JSON object per line match)")
	listOnly := flag.Bool("l", false, "list matching files only")
	maxResults := flag.Int("n", 50, "max file results")
	contextLines := flag.Int("context", 0, "context lines around matches")
	indexDir := flag.String("index-dir", defaultIndexDir(), "zoekt index directory")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "fall — find all\n\n")
		fmt.Fprintf(os.Stderr, "Usage: fall [flags] <query>\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  fall \"functionName\"              search all repos\n")
		fmt.Fprintf(os.Stderr, "  fall \"repo:magnus defun\"         filter by repo\n")
		fmt.Fprintf(os.Stderr, "  fall \"file:*.go func main\"       filter by file\n")
		fmt.Fprintf(os.Stderr, "  fall \"lang:swift class\"          filter by language\n")
		fmt.Fprintf(os.Stderr, "  fall -l \"TODO\"                   list files only\n")
		fmt.Fprintf(os.Stderr, "  fall --colors \"error\"            human-friendly output\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	query := flag.Arg(0)
	if flag.NArg() > 1 {
		// Join remaining args as part of query
		for i := 1; i < flag.NArg(); i++ {
			query += " " + flag.Arg(i)
		}
	}

	files, err := runSearch(query, SearchOpts{
		IndexDir:     *indexDir,
		MaxResults:   *maxResults,
		ContextLines: *contextLines,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "fall: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		os.Exit(0)
	}

	w := os.Stdout
	switch {
	case *jsonOut:
		formatJSON(w, files, *listOnly)
	case *colors:
		formatColor(w, files, *listOnly)
	default:
		formatPlain(w, files, *listOnly)
	}
}

func defaultIndexDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".zoekt")
	}
	return filepath.Join(home, ".zoekt")
}
