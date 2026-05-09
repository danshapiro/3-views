---
name: three-views
description: Run multiple independent agents against the same query using different hidden models, returning all labeled results. Use when you need diverse perspectives, cross-validation, comparison, or consensus on a task. Invoked for requests involving multiple viewpoints, second opinions, parallel analysis, or getting several takes on the same problem.
---

# Three Views

Run N independent `opencode run` invocations against the same query, each using a different hidden model. Each subagent is read-only. Results are labeled **alpha**, **bravo**, **charlie** (default 3), up to **delta**, **echo**, **foxtrot** (max 6).

## Build

```bash
cd scripts/three-views && go build -o three-views .
```

Set `THREE_VIEWS_ROOT` to the skill directory if the binary is relocated. Otherwise the runner resolves `config/models.json` relative to the executable.

## Run

Preferred for long prompts:

```bash
three-views --query-file "<path>" --cwd "<working-directory>"
```

Inline query:

```bash
three-views --query "<query text>" --cwd "<working-directory>"
```

Custom agent count (1–6):

```bash
three-views --query-file "<path>" --cwd "<cwd>" --agents 5
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--query-file` | — | Path to file containing the query |
| `--query` | — | Inline query text (mutually exclusive with `--query-file`) |
| `--cwd` | — | Working directory for subagents (required) |
| `--agents` | 3 | Number of agents to launch (1–6) |
| `--out-dir` | OS temp dir | Output directory |
| `--timeout` | 60 | Wall-clock timeout in minutes |

## Important

This command may take up to 60 minutes. Wait for it to complete. After 60 minutes, the runner will terminate remaining subagent processes and return the completed results available so far. Synthesize only from the complete result sections returned by the runner.

Each subagent operates read-only on the repository.

## Output

```text
===== THREE-VIEWS RUN: <run-id> =====
Run directory: <path>
Agents requested: <N>
Timeout: <N> minutes

===== AGENT ALPHA RESULT =====
Saved response: <path-to-alpha.md>

<full alpha response>

===== AGENT BRAVO RESULT =====
...

===== THREE-VIEWS END =====
```

Failed or timed-out agents appear in label order with `STATUS: timed out` or `STATUS: failed` and a log path.

## Configuration

See [config/models.json](config/models.json) for label-to-model mappings. Default labels: alpha, bravo, charlie, delta, echo, foxtrot.

## Disk Layout

```
<run-dir>/
  query.txt
  alpha.md
  bravo.md
  charlie.md
  alpha.stderr.log
  bravo.stderr.log
  charlie.stderr.log
  metadata.json
```