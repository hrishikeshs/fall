package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sourcegraph/zoekt"
)

// repoShortName extracts a short name from a zoekt repository name.
// "github.com/hrishikeshs/magnus" → "magnus"
func repoShortName(repo string) string {
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		return repo[i+1:]
	}
	return repo
}

// formatPlain writes agent-friendly output: repo/file:line: content
func formatPlain(w io.Writer, files []zoekt.FileMatch, listOnly bool) {
	for _, f := range files {
		repo := repoShortName(f.Repository)
		if listOnly {
			fmt.Fprintf(w, "%s/%s\n", repo, f.FileName)
			continue
		}
		for _, m := range f.LineMatches {
			if m.FileName {
				continue
			}
			line := bytes.TrimRight(m.Line, "\n")
			fmt.Fprintf(w, "%s/%s:%d:%s\n", repo, f.FileName, m.LineNumber, line)
		}
	}
}

// Fall colors — autumn palette (256-color ANSI)
const (
	colorReset  = "\033[0m"
	colorPath   = "\033[38;5;208m" // burnt orange — fallen leaves
	colorLine   = "\033[38;5;172m" // russet — bark and branches
	colorMatch  = "\033[1;38;5;220m" // bold gold — sunlit foliage
)

// formatColor writes human-friendly output with ANSI highlighting.
func formatColor(w io.Writer, files []zoekt.FileMatch, listOnly bool) {
	for _, f := range files {
		repo := repoShortName(f.Repository)
		if listOnly {
			fmt.Fprintf(w, "%s%s/%s%s\n", colorPath, repo, f.FileName, colorReset)
			continue
		}
		for _, m := range f.LineMatches {
			if m.FileName {
				continue
			}
			line := bytes.TrimRight(m.Line, "\n")
			highlighted := highlightMatches(line, m.LineFragments)
			fmt.Fprintf(w, "%s%s/%s%s:%s%d%s:%s\n",
				colorPath, repo, f.FileName, colorReset,
				colorLine, m.LineNumber, colorReset,
				highlighted)
		}
	}
}

// highlightMatches inserts ANSI color codes around matched fragments.
func highlightMatches(line []byte, fragments []zoekt.LineFragmentMatch) string {
	if len(fragments) == 0 {
		return string(line)
	}

	var b strings.Builder
	prev := 0
	for _, frag := range fragments {
		start := frag.LineOffset
		end := frag.LineOffset + frag.MatchLength
		if start > len(line) {
			break
		}
		if end > len(line) {
			end = len(line)
		}
		if start > prev {
			b.Write(line[prev:start])
		}
		b.WriteString(colorMatch)
		b.Write(line[start:end])
		b.WriteString(colorReset)
		prev = end
	}
	if prev < len(line) {
		b.Write(line[prev:])
	}
	return b.String()
}

// jsonMatch is the JSON output structure for a single line match.
type jsonMatch struct {
	Repo    string      `json:"repo"`
	File    string      `json:"file"`
	Line    int         `json:"line"`
	Content string      `json:"content"`
	Matches []jsonSpan  `json:"matches,omitempty"`
}

type jsonSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// formatJSON writes JSONL output, one object per line match.
func formatJSON(w io.Writer, files []zoekt.FileMatch, listOnly bool) {
	enc := json.NewEncoder(w)
	for _, f := range files {
		repo := repoShortName(f.Repository)
		if listOnly {
			enc.Encode(jsonMatch{Repo: repo, File: f.FileName})
			continue
		}
		for _, m := range f.LineMatches {
			if m.FileName {
				continue
			}
			spans := make([]jsonSpan, 0, len(m.LineFragments))
			for _, frag := range m.LineFragments {
				spans = append(spans, jsonSpan{
					Start: frag.LineOffset,
					End:   frag.LineOffset + frag.MatchLength,
				})
			}
			enc.Encode(jsonMatch{
				Repo:    repo,
				File:    f.FileName,
				Line:    m.LineNumber,
				Content: string(bytes.TrimRight(m.Line, "\n")),
				Matches: spans,
			})
		}
	}
}
