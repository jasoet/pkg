# CLAUDE.md

Read INSTRUCTION.md for full project context.

## ABSOLUTE RULE — NO EXCEPTIONS

**NEVER add AI (Claude, Copilot, or any AI) as co-author, committer, or contributor in git commits.**
Only the user's registered email may appear in commits. This is company policy — commits with AI
authorship WILL BE REJECTED. Do not use `--author`, `Co-authored-by`, or any other mechanism to
attribute commits to AI. This applies to ALL commits, including those made by tools and subagents.

## Critical Rules (never forget)

- Always use `task <name>` to run commands — never run raw commands directly. Run `task --list` to discover tasks.
- Node.js: always `bun`/`bunx` (never node, npm, npx).
- Containers: dual Docker/Podman support. This is a shared library — consumers use either runtime.
- Use brainstorming skill when user starts a new topic or plans something.
- Check and update INSTRUCTION.md and README.md when making significant changes.
- Conventional Commits: `<type>(<scope>): <description>`.
- Branch per change, squash merge. Use `gh` for PR and CI checks.
