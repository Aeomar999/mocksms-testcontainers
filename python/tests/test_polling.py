"""Unit tests for the polling helpers (no Docker daemon needed)."""

import json
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest

from mocksms_testcontainers import MocksmsContainer


class StubServer:
    def __init__(self, code):
        self.code = code

        class Handler(BaseHTTPRequestHandler):
            parent = self

            def do_GET(self):  # noqa: N802
                if self.parent.code is None:
                    self.send_response(404)
                    body = b'{"error":{"code":"not_found"}}'
                else:
                    self.send_response(200)
                    body = json.dumps({"code": self.parent.code}).encode()
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *args):  # noqa: ANN002, ANN202
                pass

        self.server = HTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    def __enter__(self):
        self.thread.start()
        host, port = self.server.server_address
        return _StubContainer(f"http://{host}:{port}")

    def __exit__(self, *exc):
        self.server.shutdown()
        self.thread.join()


class _StubContainer(MocksmsContainer):
    def __init__(self, base_url):
        self._base_url = base_url

    @property
    def base_url(self):
        return self._base_url


def test_wait_for_otp_returns_code():
    with StubServer("482916") as container:
        assert container.wait_for_otp("+15005550006", timeout=5.0, poll_interval=0.1) == "482916"


def test_wait_for_otp_timeout():
    with StubServer(None) as container:
        with pytest.raises(TimeoutError, match="timed out"):
            container.wait_for_otp("+19999999999", timeout=1.0, poll_interval=0.1)


def test_container_round_trip():
    import os

    if not os.environ.get("MOCKSMS_TEST_DOCKER"):
        pytest.skip("needs Docker daemon")
    from mocksms_testcontainers import MocksmsContainer

    with MocksmsContainer() as container:
        container.send_sms("+15555550100", "+15005550006", "Your verification code is 731946")
        assert container.wait_for_otp("+15005550006", timeout=60.0) == "731946"
