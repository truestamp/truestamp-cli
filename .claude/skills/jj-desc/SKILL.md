---
name: jj-desc
description: Write jj/git commit descriptions with intelligent change analysis. Use when the user wants to write a commit message, describe changes, or uses jj-desc, /jj-desc, or asks to describe a changeset. Also triggers on "describe this change", "commit message", or "what did I change".
allowed-tools: Bash, Read, Grep, Glob, AskUserQuestion
---

# jj-desc: Intelligent Commit Description Writer

Write well-formed commit descriptions for jj changesets by analyzing the actual
diff content. Produces commit messages following strict git/jj conventions.

This repo (truestamp-cli) is managed with jj — see the "Version control
workflow" section of CLAUDE.md. Use `jj desc` rather than `git commit -m` so
the description lands on the right change.

## Usage

- `/jj-desc` - Describe current change (@ or @- if empty)
- `/jj-desc @-` - Describe specific change
- `/jj-desc vuln-check in precommit` - With focus hint
- "describe this change" - Natural language

## Execution Flow

Only ONE user confirmation point — at the final `jj desc` command. Everything
before that is silent analysis.

### Phase 1: Detect Change ID (Silent)

IMPORTANT: Always determine the change ID fresh by running jj commands.
Never reuse a change ID from earlier in the conversation — the working
copy moves after each `jj desc`, so a previously-correct ID may now
point to a different change.

1. If argument looks like a change ID (`@`, `@-`, `@--`, short hash): use it
2. Otherwise: run `jj diff -r @ --stat --color=never` — if it has changes,
   use `@`. If empty, fall back to `@-`
3. After selecting the change ID, run `jj log` to confirm it points to the
   change you intend to describe (verify the diff content matches)
4. Store any non-change-ID argument as a focus hint for message generation

### Phase 2: Triage and Read Diff (Silent)

Start with `--stat` to understand the shape of the change, then read the
appropriate level of detail. Always use `--color=never` for machine-parseable
output.

```bash
# Step 1: Scope overview (always run first)
jj diff -r <ID> --stat --color=never

# Step 2: Triage based on stat output (see rules below)
```

#### Triage rules

Classify the changeset based on the `--stat` output. The goal is to read
substantive code without drowning in mechanical noise.

**Mechanical-only** — Every changed file is mechanical (e.g. `go.sum`,
`testdata/fuzz/Fuzz*/...` corpus files, generated CHANGELOG entries):
- Do NOT read the full diff — it's mostly opaque to human review
- Write a concise single-subject-line message; body usually unnecessary
- Examples:
  - `Update go.sum after dependency bump`
  - `Add fuzz corpus reproducers from FuzzParseBundle session`

**Code + mechanical mixed** — Some files mechanical, some not:
- Read only the substantive (non-mechanical) hunks; skip `go.sum`,
  `testdata/fuzz/`, and similar from analysis
- Focus the commit message on the code changes
- If a dependency bump motivated the change, mention it as a one-liner
  rather than enumerating every `go.sum` line

**Code-only** — Standard Go source / test / doc changes:
- Read the full diff:
  ```bash
  jj diff -r <ID> --git --color=never
  ```
- For large diffs (>800 lines), read in stages: stat first, then the most
  important files based on the stat output, rather than dumping everything

### Phase 3: Analyze Changes (Silent)

Analyze the diff to understand:

1. **Primary purpose** — What is the single most important thing this change does?
2. **Secondary changes** — Supporting changes, cleanup, test updates
3. **Scope** — Which packages/subcommands are affected?
4. **Impact** — What does this change for users or the system?
5. **Focus hint** — If the user gave one, weight the message toward it

For complex diffs (many files, multiple concerns), group related changes and
identify the unifying theme. The subject line should capture the primary purpose;
the body explains supporting details.

This is a CLI tool: when changes affect a subcommand's behaviour, flags, or
output format, that's almost always the "primary purpose" worth leading with.

### Phase 4: Draft Message (Silent)

Follow format rules from [FORMAT_RULES.md](FORMAT_RULES.md).

Summary of requirements:

**Subject line (first line):**
- 50 characters maximum — count carefully
- Imperative mood: "Add", "Fix", "Refactor" (not "Added", "Adding")
- Specific: "Add vuln-check to precommit" not "Update Taskfile"
- No trailing period
- Capitalize first word

**Second line:** MUST be blank (separates subject from body).

**Body (remaining lines):**
- Wrap at 72 characters per line
- Plain text only — no Markdown (no `##`, `**`, backticks for blocks)
- Bullet points use "- " prefix
- Explain WHY the change was made, not just WHAT changed
- Inline `code` for identifiers is acceptable sparingly

**NEVER include:**
- Co-Authored-By lines
- Signed-off-by lines
- Any attribution crediting AI, Claude, or any tool
- Issue/PR references (jj handles these separately)

The commit message is the developer's voice describing their change. It should
read as if the developer wrote it themselves, because it goes in their permanent
history.

### Phase 5: Present and Execute (ONLY User Confirmation)

Present the proposed message clearly:

```
Analyzing change: <CHANGE-ID>
Files changed: <count>
Lines modified: ~<count>

Proposed commit message:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
<formatted commit message>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Command to execute:
jj desc -r <CHANGE-ID> -m "<message>"
```

Use **AskUserQuestion** with options:

- "Yes, execute" → Run the `jj desc` command
- "Edit message" → Get revisions from user, re-draft
- "Cancel" → Stop without applying

### Phase 6: Execute or Revise

- **Yes**: Run `jj desc -r <ID> -m '...'`, confirm with
  "Description applied to change <ID>"
- **Edit**: Get user feedback, revise message, return to Phase 5
- **Cancel**: Stop, no changes made

## Supporting Files

- [FORMAT_RULES.md](FORMAT_RULES.md) - Detailed format rules and character limits
- [EXAMPLES.md](EXAMPLES.md) - Good and bad commit message examples

## Key Principles

1. **Diff is truth** — Always read the actual diff, never rely on conversation context
2. **Triage first** — Use `--stat` to understand the shape before reading details
3. **Skip noise** — `go.sum`, fuzz corpus, and generated files get one-liner or one-bullet treatment
4. **One confirmation** — Only prompt the user at the final `jj desc` command
5. **50/72 limits** — Subject 50 chars max, body wraps at 72
6. **Imperative mood** — "Add", "Fix", never past tense or gerund
7. **No attribution** — Never add Co-Authored-By or any credit line, ever
8. **Plain text body** — No Markdown formatting in commit messages
