# three-views

Agent skill that runs N independent `opencode run` subagents against the same query with different hidden models, returning labeled results.

## Build

```bash
cd scripts/three-views && go build -o three-views .
```

Binary is gitignored; rebuild after Go source changes.

## Run

```bash
./scripts/three-views/three-views --query "<text>" --cwd "<dir>"
./scripts/three-views/three-views --query-file "<path>" --cwd "<dir>" --agents 5
```

`--cwd` is required. Default `--agents 3`, max 6. Default `--timeout 60` (minutes).

## Architecture

- `SKILL.md` — skill frontmatter + caller-facing docs (opencode loads this)
- `scripts/three-views/main.go` — Go CLI runner; launches concurrent `opencode run` processes
- `config/models.json` — label-to-model mapping; runner reads this at startup
- Runner sets `OPENCODE_PERMISSION` env var to enforce read-only subagent policy
- Runner resolves `config/models.json` relative to executable (`../config/models.json`), or via `THREE_VIEWS_ROOT` env var if binary is relocated

## Key conventions

- Agent labels are fixed order: alpha, bravo, charlie, delta, echo, foxtrot
- Output is always emitted in label order regardless of completion order
- All subagents are read-only (no edit, no write, restricted bash)
- Each run creates a temp directory with `{label}.md`, `{label}.stderr.log`, `metadata.json`, `query.txt`