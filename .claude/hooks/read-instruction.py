#!/usr/bin/env python3
"""Hook to read INSTRUCTION.md and provide project context to Claude.

Copy this file to your project's .claude/hooks/read-instruction.py
and configure .claude/settings.json to run it on UserPromptSubmit.

Injection strategy:
  - First prompt of session: always inject
  - Every 15 prompts: re-inject (handles context compaction)
  - Otherwise: skip (context still fresh)

Requires: uv (https://docs.astral.sh/uv/)
Run via: uv run python "$CLAUDE_PROJECT_DIR"/.claude/hooks/read-instruction.py
"""

import hashlib
import os
import tempfile
from pathlib import Path

REINJECT_INTERVAL = 15


def get_marker_path(session_id: str, project_dir: str) -> Path:
    """Build a marker file path unique to this session + project."""
    project_hash = hashlib.md5(project_dir.encode()).hexdigest()[:8]
    marker_dir = Path(tempfile.gettempdir()) / "claude-instruction-hooks"
    marker_dir.mkdir(parents=True, exist_ok=True)
    return marker_dir / f"{session_id}-{project_hash}.count"


def should_inject(marker_path: Path) -> bool:
    """Check prompt count and decide whether to inject."""
    count = 0
    if marker_path.exists():
        try:
            count = int(marker_path.read_text().strip())
        except (ValueError, OSError):
            count = 0

    count += 1
    marker_path.write_text(str(count))

    return count == 1 or count % REINJECT_INTERVAL == 0


def main():
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", ".")
    session_id = os.environ.get("CLAUDE_SESSION_ID", "unknown")

    instruction_path = Path(project_dir) / "INSTRUCTION.md"
    if not instruction_path.exists():
        return

    marker_path = get_marker_path(session_id, project_dir)
    if not should_inject(marker_path):
        return

    content = instruction_path.read_text()
    if not content.strip():
        return

    print("<system-reminder>")
    print("<project-instruction-context>")
    print("Project Instructions from INSTRUCTION.md:")
    print()
    print(content)
    print("</project-instruction-context>")
    print("</system-reminder>")


if __name__ == "__main__":
    main()
