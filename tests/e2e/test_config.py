"""config command: profile CRUD and settings precedence."""

from __future__ import annotations

import pytest

from _harness import Run, Yopass


def test_config_lifecycle(run: Run) -> None:
    assert run("config", "add", "work", "--api", "https://api.x", "--url", "https://x").code == 0
    listing = run("config", "list")
    assert "work" in listing.stdout
    assert "* work" in listing.stdout  # first profile becomes active
    assert "https://api.x" in listing.stdout

    run("config", "add", "other", "--api", "https://api.y", "--url", "https://y")
    assert run("config", "use", "other").code == 0
    assert "* other" in run("config", "list").stdout

    assert run("config", "remove", "work").code == 0
    assert "work" not in run("config", "list").stdout


def test_config_use_unknown_is_usage_error(run: Run) -> None:
    assert run("config", "use", "ghost").code == 2


@pytest.mark.container
def test_global_defaults_drive_send(run: Run, yopass: Yopass) -> None:
    # Global defaults (no profile) point ypcli at the server; send needs no --api.
    assert run("config", "defaults", "--api", yopass.api, "--url", yopass.url).code == 0
    res = run("send", "--json", stdin="via global defaults")
    assert res.code == 0
    url = res.json()["url"]
    assert url.startswith(yopass.url)
    assert run("receive", url).stdout == "via global defaults"


@pytest.mark.container
def test_profile_overrides_global_defaults(run: Run, yopass: Yopass) -> None:
    # A profile that only sets url inherits the api from global defaults.
    assert run("config", "defaults", "--api", yopass.api).code == 0
    assert run("config", "add", "u", "--url", yopass.url).code == 0
    res = run("send", "--profile", "u", "--json", stdin="merged")
    assert res.code == 0
    assert run("receive", "--profile", "u", res.json()["url"]).stdout == "merged"


@pytest.mark.container
def test_profile_drives_send_and_receive(run: Run, yopass: Yopass) -> None:
    run("config", "add", "yp", "--api", yopass.api, "--url", yopass.url)
    res = run("send", "--profile", "yp", "--json", stdin="via profile")
    assert res.code == 0
    url = res.json()["url"]
    assert run("receive", "--profile", "yp", url).stdout == "via profile"


@pytest.mark.container
def test_flag_overrides_profile(run: Run, yopass: Yopass) -> None:
    run("config", "add", "bad", "--api", "http://127.0.0.1:1", "--url", "http://127.0.0.1:1")
    res = run(
        "send",
        "--profile",
        "bad",
        "--api",
        yopass.api,
        "--url",
        yopass.url,
        "--json",
        stdin="flag wins",
    )
    assert res.code == 0


@pytest.mark.container
def test_env_overrides_profile(run: Run, yopass: Yopass) -> None:
    run("config", "add", "bad", "--api", "http://127.0.0.1:1", "--url", "http://127.0.0.1:1")
    res = run(
        "send",
        "--profile",
        "bad",
        "--json",
        stdin="env wins",
        env_extra={"YPCLI_API": yopass.api, "YPCLI_URL": yopass.url},
    )
    assert res.code == 0
