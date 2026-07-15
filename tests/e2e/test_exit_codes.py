"""Every documented exit code, exercised end-to-end."""

from __future__ import annotations

import socket

import pytest

from _harness import FakeServer, Run, url_of


def _closed_port() -> int:
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


@pytest.mark.container
def test_exit_0_success(yp_run: Run) -> None:
    assert yp_run("send", "--json", stdin="ok").code == 0


def test_exit_2_usage(run: Run) -> None:
    assert run("receive", "--id", "someid").code == 2  # --key required with --id


def test_exit_3_config(run: Run) -> None:
    assert run("send", "--profile", "does-not-exist", stdin="x").code == 3


def test_exit_4_network(run: Run) -> None:
    api = f"http://127.0.0.1:{_closed_port()}"
    res = run("send", "--api", api, "--url", api, "--timeout", "5s", stdin="x")
    assert res.code == 4


def test_exit_5_auth(run: Run, fake: FakeServer) -> None:
    fake.state.require_auth = True
    res = run("send", "--api", fake.api, "--url", fake.api, stdin="x")
    assert res.code == 5


@pytest.mark.container
def test_exit_6_not_found_or_consumed(yp_run: Run) -> None:
    url = url_of(yp_run("send", "--one-time", "--json", stdin="once"))
    assert yp_run("receive", url).code == 0
    assert yp_run("receive", url).code == 6


@pytest.mark.container
def test_exit_7_decrypt_failure(yp_run: Run) -> None:
    body = yp_run("send", "--key", "right-key-1234567890", "--json", stdin="x").json()
    res = yp_run("receive", body["url"], "--key", "wrong-key-1234567890")
    assert res.code == 7
