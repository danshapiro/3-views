# 3-views

Agent skill that runs N independent `opencode run` subagents against the same query with different hidden models, returning labeled results.

## Run

```bash
cd scripts/3-views && go run . --query "<text>" --cwd "<dir>"
cd scripts/3-views && go run . --query-file "<path>" --cwd "<dir>" --agents 5
```

`--cwd` is required. Default `--agents 3`, max 6. Default `--timeout 60` (minutes).

## Architecture

- `SKILL.md` — skill frontmatter + caller-facing docs (opencode loads this)
- `scripts/3-views/main.go` — Go CLI runner; launches concurrent `opencode run` processes
- `config/models.json` — label-to-model mapping; runner reads this at startup
- Runner sets `OPENCODE_PERMISSION` env var to enforce read-only subagent policy
- Runner resolves `config/models.json` relative to executable (`../config/models.json`), or via `3_VIEWS_ROOT` env var if binary is relocated

## Key conventions

- Agent labels are fixed order: alpha, bravo, charlie, delta, echo, foxtrot
- Output is always emitted in label order regardless of completion order
- All subagents are read-only (no edit, no write, restricted bash)
- Each run creates a temp directory with `{label}.md`, `{label}.stderr.log`, `metadata.json`, `query.txt`
- **No extraneous docs:** Do not create `README.md`, `CHANGELOG.md`, or other standard repo files. The product is `SKILL.md` + its scripts/configs.

## Installing locally

Claude Code loads skills from `~/.claude/skills/<name>/`. OpenCode reads the same directory. Codex loads from `~/.agents/skills/<name>/`.

**Linux / macOS / WSL:**

```bash
mkdir -p ~/.claude/skills ~/.agents/skills
rsync -a --exclude='.opencode/' --exclude='.git/' --exclude='.claude/' --exclude='.agents/' "$(pwd)/" ~/.claude/skills/3-views/
rsync -a --exclude='.opencode/' --exclude='.git/' --exclude='.claude/' --exclude='.agents/' "$(pwd)/" ~/.agents/skills/3-views/
```

If running under WSL, also install to the Windows side so tools launched from Windows can find it:

```bash
if grep -qi microsoft /proc/version 2>/dev/null; then
  win_home="$(wslpath "$(powershell.exe '$env:USERPROFILE' 2>/dev/null | tr -d '\r')")"
  mkdir -p "$win_home/.claude/skills" "$win_home/.agents/skills"
  rsync -a --exclude='.opencode/' --exclude='.git/' --exclude='.claude/' --exclude='.agents/' "$(pwd)/" "$win_home/.claude/skills/3-views/"
  rsync -a --exclude='.opencode/' --exclude='.git/' --exclude='.claude/' --exclude='.agents/' "$(pwd)/" "$win_home/.agents/skills/3-views/"
fi
```

**Windows (PowerShell):**

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.claude\skills", "$env:USERPROFILE\.agents\skills"
Copy-Item -Recurse -Force "$(Get-Location)\*" "$env:USERPROFILE\.claude\skills\3-views" -Exclude ".opencode", ".claude", ".agents"
Copy-Item -Recurse -Force "$(Get-Location)\*" "$env:USERPROFILE\.agents\skills\3-views" -Exclude ".opencode", ".claude", ".agents"
```

Copy rather than symlink so you can edit locally without every change immediately taking effect. Run the copy again to update the installed version after local changes.