"""receive command: targets, output routing, and JSON mode."""

from __future__ import annotations

import pytest

from _harness import Run, url_of

pytestmark = pytest.mark.container


def test_receive_text_to_stdout_is_exact(yp_run: Run) -> None:
    url = url_of(yp_run("send", "--json", stdin="exact-bytes-no-newline"))
    got = yp_run("receive", url)
    assert got.stdout == "exact-bytes-no-newline"  # no trailing newline added


def test_receive_json_content_field(yp_run: Run) -> None:
    url = url_of(yp_run("send", "--json", stdin="json content"))
    body = yp_run("receive", url, "--json").json()
    assert body["content"] == "json content"


def test_receive_file_default_name_in_cwd(yp_run: Run, tmp_path) -> None:
    src = tmp_path / "report.txt"
    src.write_text("file body")
    url = url_of(yp_run("send", "--file", str(src), "--json"))
    workdir = tmp_path / "work"
    workdir.mkdir()
    got = yp_run("receive", url, "-o", str(workdir) + "/")
    assert got.code == 0
    assert (workdir / "report.txt").read_text() == "file body"


def test_receive_creates_missing_output_dir(yp_run: Run, tmp_path) -> None:
    src = tmp_path / "payload.bin"
    src.write_text("deep")
    url = url_of(yp_run("send", "--file", str(src), "--json"))
    target = tmp_path / "a" / "b" / "c"
    got = yp_run("receive", url, "-o", str(target) + "/")
    assert got.code == 0
    assert (target / "payload.bin").read_text() == "deep"


def test_receive_manual_key_link_without_key_is_usage_error(yp_run: Run) -> None:
    body = yp_run("send", "--key", "manual-1234567890ab", "--json", stdin="x").json()
    # Turn the s-link into a c-link (manual key) that carries no key.
    c_link = body["url"].replace("/#/s/", "/#/c/")
    res = yp_run("receive", c_link)
    assert res.code == 2
    assert "key" in res.stderr.lower()


def test_receive_id_without_key_is_usage_error(yp_run: Run) -> None:
    body = yp_run("send", "--json", stdin="x").json()
    res = yp_run("receive", "--id", body["id"])
    assert res.code == 2
