"""End-to-end round-trips against a live yopass server."""

from __future__ import annotations

import pytest

from _harness import Run, Yopass, url_of

pytestmark = pytest.mark.container


def test_text_stdin_roundtrip(yp_run: Run) -> None:
    secret = "top secret over the wire"
    url = url_of(yp_run("send", "--json", stdin=secret))
    got = yp_run("receive", url)
    assert got.code == 0
    assert got.stdout == secret


def test_text_flag_roundtrip(yp_run: Run) -> None:
    url = url_of(yp_run("send", "--text", "flag secret", "--json"))
    assert yp_run("receive", url).stdout == "flag secret"


def test_file_roundtrip_to_dir(yp_run: Run, tmp_path) -> None:
    src = tmp_path / "creds.env"
    src.write_text("USER=admin\nPASS=hunter2\n")
    res = yp_run("send", "--file", str(src), "--json")
    body = res.json()
    assert body["file"] is True
    assert "/#/f/" in body["url"]

    outdir = tmp_path / "out"
    outdir.mkdir()
    recv = yp_run("receive", body["url"], "-o", str(outdir))
    assert recv.code == 0
    assert (outdir / "creds.env").read_text() == "USER=admin\nPASS=hunter2\n"


def test_file_roundtrip_explicit_path(yp_run: Run, tmp_path) -> None:
    src = tmp_path / "in.bin"
    src.write_bytes(b"\x00\x01\x02binary\xff")
    url = url_of(yp_run("send", "--file", str(src), "--json"))
    dest = tmp_path / "restored.bin"
    assert yp_run("receive", url, "-o", str(dest)).code == 0
    assert dest.read_bytes() == b"\x00\x01\x02binary\xff"


def test_large_file_roundtrip(yp_run: Run, tmp_path) -> None:
    src = tmp_path / "big.dat"
    payload = ("ypcli-" * 20 + "\n") * 800  # ~100 KB, under the 512 KB server limit
    src.write_text(payload)
    url = url_of(yp_run("send", "--file", str(src), "--json"))
    dest = tmp_path / "big.out"
    assert yp_run("receive", url, "-o", str(dest)).code == 0
    assert dest.read_text() == payload


def test_one_time_consumed_on_second_read(yp_run: Run) -> None:
    url = url_of(yp_run("send", "--one-time", "--json", stdin="burn after reading"))
    assert yp_run("receive", url).stdout == "burn after reading"
    second = yp_run("receive", url)
    assert second.code == 6  # not found / already consumed


def test_multi_view_secret_readable_twice(yp_run: Run) -> None:
    url = url_of(yp_run("send", "--one-time=false", "--json", stdin="reusable"))
    assert yp_run("receive", url).stdout == "reusable"
    assert yp_run("receive", url).stdout == "reusable"


def test_manual_key_roundtrip(yp_run: Run) -> None:
    res = yp_run("send", "--key", "manual-key-1234567890", "--json", stdin="with manual key")
    body = res.json()
    assert body["manual_key"] is True
    assert body["url"].count("/") == 5  # .../#/s/<id> — no key segment
    got = yp_run("receive", body["url"], "--key", "manual-key-1234567890")
    assert got.stdout == "with manual key"


def test_receive_by_id_and_key(yp_run: Run) -> None:
    body = yp_run("send", "--json", stdin="by id and key").json()
    got = yp_run("receive", "--id", body["id"], "--key", body["key"])
    assert got.stdout == "by id and key"


@pytest.mark.parametrize("exp", ["1h", "1d", "1w"])
def test_supported_expirations(yp_run: Run, exp: str) -> None:
    res = yp_run("send", "--expiration", exp, "--json", stdin=f"exp {exp}")
    assert res.code == 0
    assert res.json()["expiration"] == exp


def test_argon2_autodetected_roundtrip(yp_run: Run, yopass: Yopass) -> None:
    # The session server runs with --argon2; ypcli must auto-detect and still round-trip.
    assert yopass.argon2 is True
    url = url_of(yp_run("send", "--json", stdin="argon2 path"))
    assert yp_run("receive", url).stdout == "argon2 path"
