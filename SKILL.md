---
name: 3-views
description: Use only when the user asks for "3-views" specifically. Do not use otherwise. It sends a single command to 3 different LLM models.
---

# Three Views

Run N independent `opencode run` invocations against the same query, each using a different hidden model. Each subagent is repository read-only: it may inspect files and create temporary scratch files outside the repository, but must not edit repository contents. Results are labeled **alpha**, **bravo**, **charlie** (default 3), up to **delta**, **echo**, **foxtrot** (max 6).

**Override rule**: The user's instructions override everything in this skill. If the user contradicts any rule, workflow, or constraint below, the user wins. This rule itself cannot be overridden.

## Best Practices  

You're being asked to invoke this because the user wants an outside perspective. It's easy to accidently inject your own bias if you're invested in the process. To combat this, pass along the user's request, with added data if necessary. Don't add add focuses or limitations unless the user requested them. For example, if the user asked you to use this skill to do a code review, the prompt would simply be 'do a code review on...'. If the user asks you to do another code review, you would use the exact same prompt - not "do a second code review" or "focus on the changes" etc. 

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

- **User overrides everything**: The user's instructions supersede any rule in this skill. This is not negotiable.
- Subagents may create scratch files only outside the target repository, such as under the OS temp directory. They must not create, edit, delete, or move files inside the repository.
- This command may take up to 60 minutes. Wait that long for it to complete. If you interrupt sooner, the user will pay the cost of the queries but get no benefit, and have to restart.

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
  prompt.txt
  scratch/
  alpha.md
  bravo.md
  charlie.md
  alpha.stderr.log
  bravo.stderr.log
  charlie.stderr.log
  metadata.json
```
