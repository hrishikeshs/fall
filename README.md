# fall

*Find ALL.* Indexed code search for Claude Code agents and humans.

A thin CLI on top of [zoekt](https://github.com/sourcegraph/zoekt) (Sourcegraph's trigram-indexed code search engine).

## Install

```sh
make deps    # install zoekt tools
make install # build fall, copy fall + scripts to ~/bin
```

## Usage

```sh
# Search
fall "functionName"              # search all repos
fall "repo:magnus defun"         # filter by repo
fall "file:*.go func main"       # filter by file pattern
fall "lang:swift class"          # filter by language
fall -l "TODO"                   # list matching files only
fall --colors "error"            # human-friendly colored output
fall --json "error"              # JSONL output

# Index
fall-index                       # index all git repos under ~/workspace
fall-index ~/other/dir           # index repos under a different root

# Web UI
fall-serve start                 # start zoekt web UI on :6070
fall-serve stop                  # stop it
fall-serve status                # check status
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--colors` | off | ANSI color highlighting |
| `--json` | off | JSONL output |
| `-l` | off | List matching files only |
| `-n` | 50 | Max file results |
| `--context` | 0 | Context lines around matches |
| `--index-dir` | `~/.zoekt` | Index directory |

## Architecture

Three unix-style tools:

- **`fall`** — Go binary. Searches the zoekt index with agent-friendly output.
- **`fall-index`** — Shell script. Indexes all git repos under `~/workspace`.
- **`fall-serve`** — Shell script. Manages the zoekt webserver daemon.
