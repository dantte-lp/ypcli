"""Pytest fixtures for the ypcli end-to-end suite."""

from __future__ import annotations

import os
from collections.abc import Iterator
from pathlib import Path

import pytest

from _harness import (
    FakeServer,
    Result,
    Run,
    Yopass,
    build_ypcli,
    run_ypcli,
    start_fake,
    start_yopass,
)


@pytest.fixture(scope="session")
def ypcli_bin(tmp_path_factory: pytest.TempPathFactory) -> str:
    """Path to the compiled ypcli binary (built once per session)."""
    return build_ypcli(tmp_path_factory.mktemp("bin"))


@pytest.fixture
def home(tmp_path: Path) -> Iterator[Path]:
    """Isolated XDG config home so profiles never leak between tests."""
    cfg = tmp_path / "cfg"
    cfg.mkdir()
    prev = os.environ.get("YPCLI_E2E_HOME")
    os.environ["YPCLI_E2E_HOME"] = str(cfg)
    try:
        yield cfg
    finally:
        if prev is None:
            os.environ.pop("YPCLI_E2E_HOME", None)
        else:
            os.environ["YPCLI_E2E_HOME"] = prev


@pytest.fixture(scope="session")
def yopass() -> Iterator[Yopass]:
    """A live yopass server for the whole session."""
    server = start_yopass(argon2=True)
    try:
        yield server
    finally:
        server.stop()


@pytest.fixture
def run(ypcli_bin: str, home: Path) -> Run:
    """Invoke ypcli with the given args (no server flags injected)."""

    def _run(*args: str, **kwargs) -> Result:
        return run_ypcli(ypcli_bin, *args, **kwargs)

    return _run


@pytest.fixture
def yp_run(ypcli_bin: str, home: Path, yopass: Yopass) -> Run:
    """Invoke ypcli with --api/--url already pointed at the live yopass server."""

    def _run(*args: str, **kwargs) -> Result:
        return run_ypcli(ypcli_bin, *args, "--api", yopass.api, "--url", yopass.url, **kwargs)

    return _run


@pytest.fixture
def fake() -> Iterator[FakeServer]:
    """A controllable in-process fake yopass server."""
    server = start_fake()
    try:
        yield server
    finally:
        server.stop()
