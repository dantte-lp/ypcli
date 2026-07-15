"""Bearer-token authentication behavior, verified against a controllable fake server."""

from __future__ import annotations

from _harness import FakeServer, Run

TOKEN = "s3cr3t-token"


def test_token_flag_sends_bearer(run: Run, fake: FakeServer) -> None:
    res = run("send", "--api", fake.api, "--url", fake.api, "--token", TOKEN, "--json", stdin="x")
    assert res.code == 0
    assert fake.state.last_auth == f"Bearer {TOKEN}"


def test_token_env_sends_bearer(run: Run, fake: FakeServer) -> None:
    res = run(
        "send",
        "--api",
        fake.api,
        "--url",
        fake.api,
        "--json",
        stdin="x",
        env_extra={"YPCLI_TOKEN": TOKEN},
    )
    assert res.code == 0
    assert fake.state.last_auth == f"Bearer {TOKEN}"


def test_token_command_profile_sends_bearer(run: Run, fake: FakeServer) -> None:
    run(
        "config",
        "add",
        "tok",
        "--api",
        fake.api,
        "--url",
        fake.api,
        "--token-command",
        f"printf {TOKEN}",
    )
    res = run("send", "--profile", "tok", "--json", stdin="x")
    assert res.code == 0
    assert fake.state.last_auth == f"Bearer {TOKEN}"


def test_no_token_sends_no_auth_header(run: Run, fake: FakeServer) -> None:
    assert run("send", "--api", fake.api, "--url", fake.api, stdin="x").code == 0
    assert fake.state.last_auth is None


def test_require_auth_with_valid_token_succeeds(run: Run, fake: FakeServer) -> None:
    fake.state.require_auth = True
    res = run("send", "--api", fake.api, "--url", fake.api, "--token", TOKEN, "--json", stdin="x")
    assert res.code == 0


def test_require_auth_without_token_is_rejected(run: Run, fake: FakeServer) -> None:
    fake.state.require_auth = True
    assert run("send", "--api", fake.api, "--url", fake.api, stdin="x").code == 5
