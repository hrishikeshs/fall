package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "add":
		cmdAdd(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "_daemon":
		cmdDaemon()
	case "list":
		cmdList()
	case "version", "--version", "-v":
		fmt.Printf("fall %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		cmdSearch(os.Args[1:])
	}
}

func cmdSearch(args []string) {
	fs := flag.NewFlagSet("fall", flag.ExitOnError)
	colors := fs.Bool("colors", false, "ANSI color highlighting (fall colors)")
	jsonOut := fs.Bool("json", false, "JSONL output")
	listOnly := fs.Bool("l", false, "list matching files only")
	maxResults := fs.Int("n", 50, "max file results")
	contextLines := fs.Int("context", 0, "context lines around matches")
	indexDir := fs.String("index-dir", defaultIndexDir(), "zoekt index directory")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fall [flags] <query>\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  fall \"functionName\"              search all repos\n")
		fmt.Fprintf(os.Stderr, "  fall \"repo:magnus defun\"         filter by repo\n")
		fmt.Fprintf(os.Stderr, "  fall \"file:*.go func main\"       filter by file\n")
		fmt.Fprintf(os.Stderr, "  fall \"lang:swift class\"          filter by language\n")
		fmt.Fprintf(os.Stderr, "  fall -l \"TODO\"                   list files only\n")
		fmt.Fprintf(os.Stderr, "  fall --colors \"error\"            human-friendly output\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(2)
	}

	query := fs.Arg(0)
	for i := 1; i < fs.NArg(); i++ {
		query += " " + fs.Arg(i)
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

func printUsage() {
	fmt.Fprintf(os.Stderr, `fall — find all

Indexed code search powered by zoekt trigram indices.

Usage:
  fall <query>            search indexed repos
  fall add <path> ...     add git repos to the index
  fall list               list indexed repos
  fall serve <action>     manage the web UI (start|stop|status|restart)
  fall version            print version

Search flags:
  --colors                fall-colored ANSI highlighting
  --json                  JSONL output
  -l                      list matching files only
  -n <count>              max results (default: 50)
  --context <n>           context lines (default: 0)

Query syntax:
  fall "func main"        literal search
  fall "repo:name query"  filter by repo
  fall "file:*.go query"  filter by file pattern
  fall "lang:go query"    filter by language
  fall "/regex/"          regular expression

Install: go install github.com/hrishikeshs/fall@latest
`)
}

func defaultIndexDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".zoekt")
	}
	return filepath.Join(home, ".zoekt")
}
