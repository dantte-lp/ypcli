"""completion command generates a script for each supported shell."""

from __future__ import annotations

import pytest

from _harness import Run


@pytest.mark.parametrize("shell", ["bash", "zsh", "fish", "powershell"])
def test_completion_generates_script(run: Run, shell: str) -> None:
    res = run("completion", shell)
    assert res.code == 0
    assert len(res.stdout) > 100
    assert "ypcli" in res.stdout


def test_help_lists_commands(run: Run) -> None:
    res = run("--help")
    assert res.code == 0
    for cmd in ("send", "receive", "config", "version", "completion"):
        assert cmd in res.stdout
