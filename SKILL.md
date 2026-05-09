---
name: 3-views
description: Use only when the user asks for "3-views" specifically. Do not use otherwise. It sends a single command to 3 different LLM models.
---

# Three Views

Run N independent `opencode run` invocations against the same query, each using a different hidden model. Each subagent is read-only. Results are labeled **alpha**, **bravo**, **charlie** (default 3), up to **delta**, **echo**, **foxtrot** (max 6).

## Run

Use `go run` directly from the skill's root directory:

Preferred for long prompts:

```bash
cd scripts/3-views && go run . --query-file "<path>" --cwd "<working-directory>"
```

Inline query:

```bash
cd scripts/3-views && go run . --query "<query text>" --cwd "<working-directory>"
```

Custom agent count (1–6):

```bash
cd scripts/3-views && go run . --query-file "<path>" --cwd "<cwd>" --agents 5
```

Set `3_VIEWS_ROOT` to the skill directory if the binary is relocated. Otherwise the runner resolves `config/models.json` relative to the executable.

## Run

Preferred for long prompts:

```bash
3-views --query-file "<path>" --cwd "<working-directory>"
```

Inline query:

```bash
3-views --query "<query text>" --cwd "<working-directory>"
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--query-file` | — | Path to file containing the query |
| `--query` | — | Inline query text (mutually exclusive with `--query-file`) |
| `--cwd` | — | Working directory for subagents (required) |
| `--agents` | 3 | Number of agents to launch (1–6) |
| `--out-dir` | OS temp dir | Base directory for the unique run folder |
| `--timeout` | 60 | Wall-clock timeout in minutes |

## Important

This command may take up to 60 minutes. Wait that long for it to complete. If you interrupt sooner, the user will pay the cost of the queries but get no benefit, and have to restart.

## Output

```text
===== 3-VIEWS RUN: <run-id> =====
Run directory: <path>
Agents requested: <N>
Timeout: <N> minutes

===== AGENT ALPHA RESULT =====
Saved response: <path-to-alpha.md>

<full alpha response>

===== AGENT BRAVO RESULT =====
...

===== 3-VIEWS END =====
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