#!/usr/bin/env python3
"""Hook to read INSTRUCTION.md and provide project context to Claude.

Requires: uv (https://docs.astral.sh/uv/)
Run via: uv run python "$CLAUDE_PROJECT_DIR"/.claude/hooks/read-instruction.py
"""

import os
from pathlib import Path

MAX_CONTENT_LENGTH = 4000


def main():
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", ".")
    instruction_path = Path(project_dir) / "INSTRUCTION.md"

    if not instruction_path.exists():
        return

    content = instruction_path.read_text()
    if not content.strip():
        return

    content = content[:MAX_CONTENT_LENGTH]

    print("<system-reminder>")
    print("<project-instruction-context>")
    print("Project Instructions from INSTRUCTION.md:")
    print()
    print(content)
    print("</project-instruction-context>")
    print("</system-reminder>")


if __name__ == "__main__":
    main()
