import socket
import subprocess
import time
from itertools import count
from pathlib import Path
from threading import Lock

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
READY_TIMEOUT = 5.0  # seconds to wait for the server to accept a connection


def _free_port() -> int:
    """Ask the OS for an unused TCP port instead of hardcoding one, so the
    test server never fights the default 8080 or a previous run."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def _wait_until_ready(host: str, port: int, timeout: float) -> None:
    """Poll for a successful TCP connect instead of a fixed sleep — the
    server is ready exactly when it starts accepting connections, so that's
    what we check for, no more and no less."""
    deadline = time.monotonic() + timeout
    last_err = None
    while time.monotonic() < deadline:
        try:
            with socket.create_connection((host, port), timeout=0.2):
                return
        except OSError as e:
            last_err = e
            time.sleep(0.05)
    raise TimeoutError(f"server never accepted a connection on {host}:{port}: {last_err}")


@pytest.fixture(scope="session")
def server():
    """Start the real Go server once for the whole test session."""
    host = "127.0.0.1"
    port = _free_port()

    proc = subprocess.Popen(
        [
            "go", "run",
            "server/main.go", "server/engine.go", "server/protocol.go",
            "-addr", host, "-port", str(port),
        ],
        cwd=REPO_ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )

    try:
        _wait_until_ready(host, port, READY_TIMEOUT)
    except TimeoutError:
        proc.terminate()
        out, _ = proc.communicate(timeout=5)
        pytest.fail(f"server failed to become ready:\n{out}")

    yield host, port

    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait()


@pytest.fixture
def client_socket(server):
    """A fresh TCP connection per test. Keeping connections per-test (rather
    than sharing one across the whole session) keeps tests isolated — one
    test's malformed frame or half-read response can't corrupt another
    test's view of the stream."""
    host, port = server
    sock = socket.create_connection((host, port), timeout=2)
    yield sock
    sock.close()


# The vault pre-seeds keys 0..9999 at startup (see engine.go NewVault), so a
# genuine "missing key" test needs a key outside that range. Start well
# clear of it and hand out a fresh one per test so tests never collide.
_key_counter = count(10_000)
_key_lock = Lock()


@pytest.fixture
def unique_key():
    """Call it each time you need a fresh key — every call returns a new
    value from the same counter, so no two keys anywhere in the suite ever
    collide, no matter how many a single test needs."""
    def _next() -> int:
        with _key_lock:
            return next(_key_counter)
    return _next