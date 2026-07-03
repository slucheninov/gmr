---
description: "Global workflow (revised 2026-07-03 for Claude Code + Fable 5 default). Main session runs Fable 5; it MUST delegate Write/Edit/Read/Shell past hard numerical thresholds to Sonnet subagents via the Agent tool. Use superpowers skills (brainstorming, writing-plans, executing-plans, dispatching-parallel-agents, using-git-worktrees, verification-before-completion). DO NOT use gsd-* (archived) or claude-mem skills (make-plan/do/learn-codebase)."
alwaysApply: true
---

# Global Workflow Conventions

User-level defaults for every Claude Code session. **User instructions in chat win.**

## Default model: Fable 5. The main session MUST delegate execution.

The main session runs **Fable 5** (`claude-fable-5`, Mythos-class — above Opus) because architectural reasoning, plan synthesis, review judgement, and multi-source integration benefit from depth. Direct Fable writes/edits are significantly more expensive than Sonnet, so cost discipline is structural: the main session operates under the hard numerical delegation rules below.

> ⚠️ There is currently NO enforcement hook: the old Cursor `preToolUse` hook (`~/.cursor/hooks/enforce-large-write.sh`) does not exist in this environment. The rules below are self-enforced convention. The only active hook is RTK (`~/.claude/hooks/rtk-rewrite.sh`), which rewrites shell commands for token savings — leave it enabled.

| Operation | Model |
|---|---|
| Main chat (default) | **Fable 5** (`claude-fable-5`) |
| Plan Mode | Fable 5 (already there) |
| `Agent()` subagent dispatch | **`sonnet`** (resolves to Sonnet 4.6) |
| Reviewer subagents (`coderabbit:code-reviewer`, `code-simplifier`) | `model: "sonnet"` |
| `general-purpose` for plan execution | Sonnet |
| Isolated worktree runs (`isolation: "worktree"`) | Sonnet |
| Heavy mechanical edits (typo, rename, boilerplate) | Optionally `haiku` |

## HARD RULES — check BEFORE every tool call

### Rule 1 — Writing files (Write tool)

| File size                | Action                                                                |
|--------------------------|-----------------------------------------------------------------------|
| >30 lines                | STOP. Dispatch `Agent(general-purpose, model="sonnet")`.              |
| 10–30 lines, mechanical  | Prefer Agent dispatch. Direct Write only with user reviewing inline.  |
| <10 lines, hand-written  | Direct Write OK.                                                      |

Multiple small files in a row (≥2) → batch into ONE Agent dispatch.

### Rule 2 — Editing files (Edit tool)

| Edit volume                    | Action                                                                |
|--------------------------------|-----------------------------------------------------------------------|
| Single edit, <10 lines diff    | Direct Edit OK.                                                       |
| ≥2 edits (same or multi-file)  | STOP. Dispatch `Agent(general-purpose, sonnet)` with full list.       |

### Rule 3 — Reading files (Read tool)

| Read volume                    | Action                                                       |
|--------------------------------|--------------------------------------------------------------|
| 1–2 files                      | Direct Read OK.                                              |
| ≥3 files for analysis          | STOP. Dispatch `Agent(Explore, sonnet)`.                     |

Exception: file the user is currently editing or which is the literal subject of the question.

### Rule 4 — Shell context-gathering

| Command volume                 | Action                                                                  |
|--------------------------------|--------------------------------------------------------------------------|
| 1–2 quick commands             | Direct Bash OK (`git status`, `ls`).                                    |
| ≥3 commands for context        | STOP. Dispatch `Agent(general-purpose, sonnet)` with batched commands.  |

There is no dedicated `shell` agent type in Claude Code — batched shell work goes to `general-purpose`.

### Rule 5 — Self-audit at start of every response

Before issuing ANY tool call, count what the current plan implies:

- Planned Write+Edit operations total ≥2 → one Agent dispatch instead.
- Planned Read operations total ≥3 → Agent(Explore, sonnet).
- Planned shell commands for context ≥3 → Agent(general-purpose, sonnet).
- Any single file >30 lines being written → Agent(general-purpose, sonnet).

If ANY threshold trips, the FIRST tool call in the response must be the Agent dispatch.

## Dispatch templates (use verbatim)

```
Agent(
  subagent_type: "general-purpose",
  model: "sonnet",
  description: "<3-5 word title>",
  prompt: "<full spec: paths, content verbatim or specs, acceptance criteria, lints to pass>"
)

Agent(
  subagent_type: "Explore",
  model: "sonnet",
  description: "<3-5 word title>",
  prompt: "<question + directory + expected return shape>"
)
```

Removed / unavailable in Claude Code (do not reference):
- Cursor `Task()` tool and `generalPurpose`/`explore`/`shell` subagent types — use the `Agent` tool with `general-purpose`, `Explore`, `Plan`, or `fork`.
- `best-of-n-runner` — use `Agent(..., isolation: "worktree")` or the `using-git-worktrees` skill.
- `composer-2-fast` — use `model: "haiku"`.
- Cursor reviewer subagents (`go-expert`, `postgres-expert`, `react-ts-expert`, `slack-expert`, `test-writer`, `plan-verifier`, `requirements-planner`) — not installed; use `coderabbit:code-reviewer`, `code-simplifier`, or a Sonnet `general-purpose` agent with a review prompt.
- `implementor` / `react-reviewer` — removed 2026-05-16.

## What the main session DOES directly

- Read 1–2 relevant files for reasoning.
- 1–2 inline shell commands for orientation.
- Edit ≤1 per response, ≤10 lines.
- Plan documents, ADRs, design conversations, review verdicts (in chat, not as files).
- Synthesis of subagent results.
- Direct user dialogue.

## What the main session NEVER does directly

- Write a file >30 lines.
- Apply ≥2 edits in one response.
- Read ≥3 files for analysis.
- Run ≥3 context-gathering shell commands.
- Multi-file refactors.
- Migration writing.

## Anti-rationalisation list

| Tempting thought                                | Reality                                                  |
|-------------------------------------------------|----------------------------------------------------------|
| "It's faster if I do it directly"               | Sonnet subagent returns in 10–30s. Count seconds.        |
| "I already have the context"                    | Pass it in the prompt.                                   |
| "Just this one file"                            | Measure lines. >30? Dispatch.                            |
| "Simple mechanical edit"                        | Simple → Sonnet trivial. Dispatch.                       |
| "The user is waiting"                           | Burning budget makes them wait longer. Dispatch.         |
| "Subagent might miss context"                   | Pass exact paths and snippets.                           |
| "I'll just start and switch later"              | No. First tool call = the dispatch.                      |
| "A hook will catch me if I'm wrong"             | There is no enforcement hook. The rules are self-enforced. |

## Three workflow patterns

**Pattern A — Parallel isolated agents** (multiple independent features):
- `Agent(..., isolation: "worktree")` (auto git-worktree), or
- `using-git-worktrees` superpowers skill (manual)
- See `dispatching-parallel-agents` skill.

**Pattern B — Plan in Fable, execute in Sonnet** (one big feature):
1. Plan Mode (Fable) or `writing-plans` skill → PLAN.md
2. Dispatch each atomic step to `Agent(general-purpose, sonnet)`
3. Executor invokes the relevant skill before coding
4. `verification-before-completion` skill before any "done" claim

**Pattern C — Parallel reviewer subagents** (tests + review + plan-verify on one PR):
- ONE message with multiple parallel `Agent(...)` calls, all Sonnet
- Typical fan-out: `coderabbit:code-reviewer`, `code-simplifier`, plus a `general-purpose` agent for test coverage
- See `dispatching-parallel-agents` skill

## Skill priority

- `brainstorming` — before ANY creative work
- `writing-plans` — after brainstorming, before code
- `executing-plans` — execute plan with review checkpoints
- `subagent-driven-development` — parallel tasks in current session
- `dispatching-parallel-agents` — 2+ independent tasks
- `using-git-worktrees` — isolation
- `verification-before-completion` — before "done" claims
- `requesting-code-review` / `receiving-code-review`
- `finishing-a-development-branch` — merge/PR/cleanup decision
- `systematic-debugging` — any bug or unexpected behavior
- `test-driven-development` — when implementing features

Workspace-specific skills under `<repo>/.claude/skills/` win over global skills (none exist in this repo currently).

## DO NOT use

- `claude-mem` skills (`make-plan`, `do`, `learn-codebase`, `pathfinder`, `smart-explore`) — use superpowers equivalents. Cross-session context is covered by Claude Code's built-in persistent memory.
- `gsd-*` anything — archived 2026-05-16 to `~/temp/dotfiles-cleanup-2026-05-16/`.
- Disabling the RTK hook (`~/.claude/hooks/rtk-rewrite.sh`) without replacement.

## Cost discipline

1. Fable default but Fable delegates — hard rules above
2. Batch parallel `Agent()` calls in ONE message — prompt cache stays warm
3. RTK proxy (via hook) trims shell output tokens — check `rtk gain` occasionally
4. Lean on built-in persistent memory — don't re-explain known context
5. Plan Mode for big decisions — small edits go straight to Sonnet dispatch
6. Healthy week: Sonnet >70% of tokens, Fable <30%

## Workspace anti-patterns

- Sequential `Agent()` calls when they could be parallel — batch in one message.
- Re-running already-memoized tools in the same session.
- Hesitating to dispatch reviewer subagents — Sonnet review (~$0.02) is trivial vs catching bugs late.
