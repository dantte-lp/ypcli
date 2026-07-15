"""send command: input sources, flags, and output shapes."""

from __future__ import annotations

import os
import stat
from pathlib import Path

import pytest

from _harness import FakeServer, Run

pytestmark = pytest.mark.container


def test_send_stdin_prints_url(yp_run: Run) -> None:
    res = yp_run("send", stdin="hello")
    assert res.code == 0
    assert res.stdout.strip().startswith("http")
    assert "/#/s/" in res.stdout


def test_send_file_url_prefix(yp_run: Run, tmp_path) -> None:
    f = tmp_path / "secret.conf"
    f.write_text("data")
    res = yp_run("send", "--file", str(f))
    assert "/#/f/" in res.stdout


def test_send_json_shape(yp_run: Run) -> None:
    body = yp_run("send", "--json", "--expiration", "1d", "--one-time", stdin="x").json()
    assert set(body) >= {"id", "url", "key", "manual_key", "file", "one_time", "expiration"}
    assert body["one_time"] is True
    assert body["file"] is False
    assert body["expiration"] == "1d"
    assert body["key"] and body["key"] in body["url"]


def test_send_manual_key_reports_key_on_stderr(yp_run: Run) -> None:
    res = yp_run("send", "--key", "abc-key-1234567890ab", stdin="x")
    assert res.code == 0
    assert "abc-key-1234567890ab" in res.stderr  # key surfaced separately
    assert "abc-key-1234567890ab" not in res.stdout  # not in the URL


def test_send_qr_renders_to_stderr(yp_run: Run) -> None:
    res = yp_run("send", "--qr", stdin="qr me")
    assert res.code == 0
    assert any(ch in res.stderr for ch in "█▀▄")


def test_send_copy_is_non_fatal_when_headless(yp_run: Run) -> None:
    # In CI there is no clipboard; --copy must warn but never fail the command.
    res = yp_run("send", "--copy", stdin="clip")
    assert res.code == 0
    assert res.stdout.strip().startswith("http")


def test_send_require_auth_propagated_to_server(yp_run: Run) -> None:
    # ypcli sends the require_auth flag; a server without auth configured rejects
    # it with a clear 400, which ypcli must surface faithfully (non-zero exit).
    res = yp_run("send", "--require-auth", "--json", stdin="ra")
    assert res.code != 0
    assert "authentication" in res.stderr.lower()


def test_send_invalid_expiration_is_usage_error(yp_run: Run) -> None:
    res = yp_run("send", "--expiration", "2w", stdin="x")
    assert res.code == 2
    assert "expiration" in res.stderr.lower()


@pytest.mark.skipif(os.name == "nt", reason="uses a POSIX fake-editor script")
def test_send_editor_mode(yp_run: Run, tmp_path: Path) -> None:
    # A non-interactive "editor" that writes a fixed payload into the temp file.
    editor = tmp_path / "fakeeditor.sh"
    editor.write_text('#!/bin/sh\nprintf "composed in editor" > "$1"\n')
    editor.chmod(editor.stat().st_mode | stat.S_IEXEC)

    res = yp_run("send", "--editor", "--json", env_extra={"YPCLI_EDITOR": str(editor)})
    assert res.code == 0
    url = res.json()["url"]
    assert yp_run("receive", url).stdout == "composed in editor"


def test_send_from_vault(run: Run, yopass, fake: FakeServer) -> None:
    # Real yopass for the share, fake server acting as the Vault/OpenBao KV engine.
    fake.state.vault = {"password": "hunter2"}
    res = run(
        "send",
        "--api",
        yopass.api,
        "--url",
        yopass.url,
        "--json",
        "--vault-addr",
        fake.api,
        "--vault-token",
        fake.state.valid_token,
        "--vault-path",
        "db",
        "--vault-field",
        "password",
    )
    assert res.code == 0
    url = res.json()["url"]
    assert run("receive", url, "--api", yopass.api).stdout == "hunter2"


def test_send_from_vault_wrong_token_is_auth_error(run: Run, yopass, fake: FakeServer) -> None:
    res = run(
        "send",
        "--api",
        yopass.api,
        "--url",
        yopass.url,
        "--vault-addr",
        fake.api,
        "--vault-token",
        "wrong",
        "--vault-path",
        "db",
        "--vault-field",
        "password",
    )
    assert res.code == 5  # Vault 403 -> ErrUnauthorized
