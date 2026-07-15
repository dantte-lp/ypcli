"""version command against the real server and a legacy (no /version) fake."""

from __future__ import annotations

import pytest

from _harness import FakeServer, Run


@pytest.mark.container
def test_version_text(yp_run: Run) -> None:
    res = yp_run("version")
    assert res.code == 0
    assert "ypcli" in res.stdout
    assert "server" in res.stdout


@pytest.mark.container
def test_version_json_reports_real_server(yp_run: Run) -> None:
    body = yp_run("version", "--json").json()
    assert {"version", "commit", "date", "server"} <= set(body)
    assert body["server"]
    assert body["server"] != "unsupported (pre-13.x)"


def test_version_against_legacy_server(run: Run, fake: FakeServer) -> None:
    fake.state.version = None  # server without a /version endpoint
    body = run("version", "--api", fake.api, "--json").json()
    assert "unsupported" in body["server"]
