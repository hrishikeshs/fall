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

// splitContext splits a byte slice into individual lines, trimming trailing newline.
func splitContext(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := string(bytes.TrimRight(b, "\n"))
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// formatPlain writes agent-friendly output with optional context.
func formatPlain(w io.Writer, files []zoekt.FileMatch, listOnly bool) {
	for fi, f := range files {
		repo := repoShortName(f.Repository)
		if listOnly {
			fmt.Fprintf(w, "%s/%s\n", repo, f.FileName)
			continue
		}
		prefix := repo + "/" + f.FileName
		lastEnd := -1

		for mi, m := range f.LineMatches {
			if m.FileName {
				continue
			}

			before := splitContext(m.Before)
			after := splitContext(m.After)

			// Separator between non-contiguous groups
			if len(before) > 0 || len(after) > 0 {
				firstCtx := m.LineNumber - len(before)
				if mi > 0 && lastEnd >= 0 && firstCtx > lastEnd {
					fmt.Fprintf(w, "--\n")
				}
			}

			// Before context
			for i, line := range before {
				num := m.LineNumber - len(before) + i
				fmt.Fprintf(w, "%s-%d-%s\n", prefix, num, line)
			}

			// Match line
			line := bytes.TrimRight(m.Line, "\n")
			fmt.Fprintf(w, "%s:%d:%s\n", prefix, m.LineNumber, line)

			// After context
			for i, line := range after {
				num := m.LineNumber + 1 + i
				fmt.Fprintf(w, "%s-%d-%s\n", prefix, num, line)
			}

			lastEnd = m.LineNumber + len(after) + 1
		}

		// Separator between files when showing context
		if !listOnly && fi < len(files)-1 && lastEnd > 0 {
			fmt.Fprintf(w, "--\n")
		}
	}
}

// Fall colors — autumn palette (256-color ANSI)
const (
	colorReset   = "\033[0m"
	colorPath    = "\033[38;5;208m" // burnt orange — fallen leaves
	colorLine    = "\033[38;5;172m" // russet — bark and branches
	colorMatch   = "\033[1;38;5;220m" // bold gold — sunlit foliage
	colorContext = "\033[38;5;243m" // grey — bare branches
	colorSep     = "\033[38;5;240m" // dark grey
)

// formatColor writes human-friendly output with ANSI highlighting.
func formatColor(w io.Writer, files []zoekt.FileMatch, listOnly bool) {
	for fi, f := range files {
		repo := repoShortName(f.Repository)
		if listOnly {
			fmt.Fprintf(w, "%s%s/%s%s\n", colorPath, repo, f.FileName, colorReset)
			continue
		}
		prefix := repo + "/" + f.FileName
		lastEnd := -1

		for mi, m := range f.LineMatches {
			if m.FileName {
				continue
			}

			before := splitContext(m.Before)
			after := splitContext(m.After)

			// Separator
			if len(before) > 0 || len(after) > 0 {
				firstCtx := m.LineNumber - len(before)
				if mi > 0 && lastEnd >= 0 && firstCtx > lastEnd {
					fmt.Fprintf(w, "%s--%s\n", colorSep, colorReset)
				}
			}

			// Before context
			for i, line := range before {
				num := m.LineNumber - len(before) + i
				fmt.Fprintf(w, "%s%s%s-%s%d%s-%s%s%s\n",
					colorPath, prefix, colorReset,
					colorLine, num, colorReset,
					colorContext, line, colorReset)
			}

			// Match line
			line := bytes.TrimRight(m.Line, "\n")
			highlighted := highlightMatches(line, m.LineFragments)
			fmt.Fprintf(w, "%s%s%s:%s%d%s:%s\n",
				colorPath, prefix, colorReset,
				colorLine, m.LineNumber, colorReset,
				highlighted)

			// After context
			for i, line := range after {
				num := m.LineNumber + 1 + i
				fmt.Fprintf(w, "%s%s%s-%s%d%s-%s%s%s\n",
					colorPath, prefix, colorReset,
					colorLine, num, colorReset,
					colorContext, line, colorReset)
			}

			lastEnd = m.LineNumber + len(after) + 1
		}

		// File separator
		if !listOnly && fi < len(files)-1 && lastEnd > 0 {
			fmt.Fprintf(w, "%s--%s\n", colorSep, colorReset)
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
	Repo    string     `json:"repo"`
	File    string     `json:"file"`
	Line    int        `json:"line"`
	Content string     `json:"content"`
	Before  []string   `json:"before,omitempty"`
	After   []string   `json:"after,omitempty"`
	Matches []jsonSpan `json:"matches,omitempty"`
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
				Before:  splitContext(m.Before),
				After:   splitContext(m.After),
				Matches: spans,
			})
		}
	}
}
