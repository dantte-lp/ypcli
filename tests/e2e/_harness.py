"""Test harness: run the ypcli binary, manage a real yopass container, and a fake server.

The harness has no third-party dependencies beyond pytest; it drives the compiled
`ypcli` binary via subprocess and talks to a real yopass server (started with podman
or docker) for functional tests. A small in-process fake HTTP server covers cases the
free yopass image cannot produce deterministically (auth 401, server 500, missing
/version).
"""

from __future__ import annotations

import contextlib
import json
import os
import shutil
import socket
import subprocess
import threading
import time
import uuid
from collections.abc import Callable
from dataclasses import dataclass, field
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.request import urlopen

REPO_ROOT = Path(__file__).resolve().parents[2]


# --------------------------------------------------------------------------- ypcli


@dataclass
class Result:
    """Outcome of one ypcli invocation."""

    code: int
    stdout: str
    stderr: str

    def json(self) -> dict:
        return json.loads(self.stdout)


# Signature of the `run` / `yp_run` fixtures: (*args, **kwargs) -> Result.
Run = Callable[..., Result]


def run_ypcli(
    binary: str,
    *args: str,
    stdin: str | None = None,
    env_extra: dict[str, str] | None = None,
    timeout: int = 30,
) -> Result:
    """Invoke the ypcli binary and capture its result."""
    env = os.environ.copy()
    if env_extra:
        env.update(env_extra)
    # Force per-test config isolation. setdefault is not enough: the CI runner
    # may already export XDG_CONFIG_HOME, which would leak profiles across tests.
    env["XDG_CONFIG_HOME"] = os.environ["YPCLI_E2E_HOME"]
    proc = subprocess.run(
        [binary, *args],
        input=stdin,
        capture_output=True,
        text=True,
        env=env,
        timeout=timeout,
        check=False,
    )
    return Result(code=proc.returncode, stdout=proc.stdout, stderr=proc.stderr)


def build_ypcli(dest_dir: Path) -> str:
    """Build the ypcli binary once and return its path. Honors YPCLI_BIN."""
    if env_bin := os.environ.get("YPCLI_BIN"):
        return env_bin
    out = dest_dir / "ypcli"
    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"
    env.setdefault("GOTOOLCHAIN", "local")
    subprocess.run(
        ["go", "build", "-o", str(out), "./cmd/ypcli"],
        cwd=REPO_ROOT,
        env=env,
        check=True,
    )
    return str(out)


# ----------------------------------------------------------------------- real yopass


def _free_port() -> int:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def _container_engine() -> str | None:
    for engine in ("podman", "docker"):
        if shutil.which(engine):
            return engine
    return None


@dataclass
class Yopass:
    """A running yopass server, either external (YPCLI_E2E_API) or a managed container."""

    api: str
    argon2: bool
    _engine: str | None = None
    _net: str | None = None
    _names: list[str] = field(default_factory=list)

    @property
    def url(self) -> str:  # public URL == api for the CLI's purposes in tests
        return self.api

    def wait_ready(self, timeout: float = 40.0) -> None:
        deadline = time.time() + timeout
        while time.time() < deadline:
            with contextlib.suppress(Exception), urlopen(self.api + "/config", timeout=2) as r:
                if r.status == 200:
                    return
            time.sleep(1)
        raise RuntimeError(f"yopass not ready at {self.api}")

    def stop(self) -> None:
        if not self._engine:
            return
        for name in self._names:
            subprocess.run(
                [self._engine, "rm", "-f", name],
                capture_output=True,
                check=False,
            )
        if self._net:
            subprocess.run(
                [self._engine, "network", "rm", self._net],
                capture_output=True,
                check=False,
            )


def start_yopass(argon2: bool = True) -> Yopass:
    """Use an external server if provided, else start memcached + yopass via a container engine."""
    if api := os.environ.get("YPCLI_E2E_API"):
        yp = Yopass(api=api.rstrip("/"), argon2=os.environ.get("YPCLI_E2E_ARGON2") == "1")
        yp.wait_ready()
        return yp

    engine = _container_engine()
    if engine is None:
        raise RuntimeError("no container engine (podman/docker) and no YPCLI_E2E_API set")

    suffix = uuid.uuid4().hex[:8]
    net = f"ye2e-{suffix}"
    mc = f"ye2e-mc-{suffix}"
    yp_name = f"ye2e-yp-{suffix}"
    port = _free_port()

    subprocess.run([engine, "network", "create", net], capture_output=True, check=True)
    subprocess.run(
        [engine, "run", "-d", "--name", mc, "--network", net, "docker.io/library/memcached:alpine"],
        capture_output=True,
        check=True,
    )
    yp_args = [
        engine,
        "run",
        "-d",
        "--name",
        yp_name,
        "--network",
        net,
        "-p",
        f"127.0.0.1:{port}:1337",
        "docker.io/jhaals/yopass:latest",
        f"--memcached={mc}:11211",
    ]
    if argon2:
        yp_args.append("--argon2")
    subprocess.run(yp_args, capture_output=True, check=True)

    yp = Yopass(
        api=f"http://127.0.0.1:{port}",
        argon2=argon2,
        _engine=engine,
        _net=net,
        _names=[yp_name, mc],
    )
    yp.wait_ready()
    return yp


# ------------------------------------------------------------------------- fake server


class _FakeState:
    def __init__(self) -> None:
        self.secrets: dict[str, str] = {}
        self.files: dict[str, bytes] = {}
        self.argon2 = False
        self.version: str | None = "fake-1.0"
        self.require_auth = False
        self.valid_token = "s3cr3t-token"
        self.force_status: int | None = None
        self.last_auth: str | None = None
        self.counter = 0

    def next_id(self) -> str:
        self.counter += 1
        return f"id{self.counter:04d}"


class _Handler(BaseHTTPRequestHandler):
    state: _FakeState

    def log_message(self, format: str, *args: object) -> None:  # silence
        pass

    def _json(self, status: int, body: dict) -> None:
        payload = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def _authorized(self) -> bool:
        st = self.state
        st.last_auth = self.headers.get("Authorization")
        if not st.require_auth:
            return True
        return st.last_auth == f"Bearer {st.valid_token}"

    def do_GET(self) -> None:
        st = self.state
        if self.path == "/config":
            self._json(200, {"ARGON2": st.argon2})
        elif self.path == "/version":
            if st.version is None:
                self._json(404, {"message": "not found"})
            else:
                self._json(200, {"version": st.version})
        elif self.path.startswith("/secret/"):
            sid = self.path.removeprefix("/secret/")
            msg = st.secrets.pop(sid, None)
            if msg is None:
                self._json(404, {"message": "gone"})
            else:
                self._json(200, {"message": msg})
        elif self.path.startswith("/file/"):
            fid = self.path.removeprefix("/file/")
            data = st.files.pop(fid, None)
            if data is None:
                self.send_response(404)
                self.end_headers()
            else:
                self.send_response(200)
                self.send_header("Content-Type", "application/octet-stream")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)
        else:
            self._json(404, {"message": "unknown"})

    def do_POST(self) -> None:
        st = self.state
        if st.force_status is not None:
            self._json(st.force_status, {"message": "forced"})
            return
        if not self._authorized():
            self._json(401, {"message": "unauthorized"})
            return
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        if self.path == "/create/secret":
            msg = json.loads(body)["message"]
            sid = st.next_id()
            st.secrets[sid] = msg
            self._json(200, {"message": sid})
        elif self.path == "/create/file":
            fid = st.next_id()
            st.files[fid] = body
            self._json(200, {"message": fid})
        else:
            self._json(404, {"message": "unknown"})


@dataclass
class FakeServer:
    api: str
    state: _FakeState
    _httpd: ThreadingHTTPServer

    def stop(self) -> None:
        self._httpd.shutdown()


def start_fake() -> FakeServer:
    state = _FakeState()
    handler = type("BoundHandler", (_Handler,), {"state": state})
    httpd = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    threading.Thread(target=httpd.serve_forever, daemon=True).start()
    host, port = httpd.server_address[:2]
    return FakeServer(api=f"http://{host}:{port}", state=state, _httpd=httpd)


# --------------------------------------------------------------------------- helpers


def url_of(result: Result) -> str:
    """Extract the share URL from a --json send result."""
    assert result.code == 0, f"send failed: {result.stderr}"
    return result.json()["url"]
