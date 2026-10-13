"""Testcontainers module for the mocksms sandbox, with wait_for_otp helpers."""

from __future__ import annotations

import json
import os
import time
import urllib.parse
import urllib.request
from typing import Any, Optional

from testcontainers.core.container import DockerContainer
from testcontainers.core.waiting_utils import wait_for_logs


DEFAULT_IMAGE = "ghcr.io/aeomar999/commpit/mocksms:latest"


def _get(url: str, params: Optional[dict[str, str]] = None) -> tuple[int, Any]:
    if params:
        url += "?" + urllib.parse.urlencode(params)
    req = urllib.request.Request(url, headers={"X-Mocksms": "true"})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else None
    except urllib.error.HTTPError as exc:
        return exc.code, None


class MocksmsContainer:
    """A running mocksms sandbox (in-memory store)."""

    def __init__(self, image: str = "", otp_code: str = "123456") -> None:
        # MOCKSMS_IMAGE overrides the image (CI matrix, local builds).
        self._container = (
            DockerContainer(image or os.environ.get("MOCKSMS_IMAGE", DEFAULT_IMAGE))
            .with_env("MOCKSMS_MEMORY", "true")
            .with_env("MOCKSMS_OTP_CODE", otp_code)
            .with_env("MOCKSMS_HTTP_HOST", "0.0.0.0")
            .with_env("MOCKSMS_SMTP_HOST", "0.0.0.0")
            .with_exposed_ports(4010, 1025)
        )

    @property
    def base_url(self) -> str:
        return f"http://{self._container.get_container_host_ip()}:{self._container.get_exposed_port(4010)}"

    def start(self, timeout: float = 30.0) -> "MocksmsContainer":
        self._container.start()
        wait_for_logs(self._container, ".*", timeout=timeout)
        deadline = time.time() + timeout
        while True:
            try:
                status, _ = _get(self.base_url + "/api/v1/healthz")
                if status == 200:
                    return self
            except OSError:
                pass
            if time.time() >= deadline:
                raise TimeoutError("mocksms did not become healthy in time")
            time.sleep(0.2)

    def stop(self) -> None:
        self._container.stop()

    def __enter__(self) -> "MocksmsContainer":
        return self.start()

    def __exit__(self, *exc: Any) -> None:
        self.stop()

    def send_sms(self, sender: str, to: str, body: str) -> Any:
        payload = json.dumps({"from": sender, "to": to, "body": body}).encode()
        req = urllib.request.Request(
            self.base_url + "/api/v1/sms",
            data=payload,
            headers={"X-Mocksms": "true", "Content-Type": "application/json"},
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            if resp.status not in (201, 202):
                raise RuntimeError(f"send SMS failed: {resp.status}")
            return json.loads(resp.read().decode())

    def wait_for_otp(self, to: str, timeout: float = 30.0, poll_interval: float = 0.5) -> str:
        """Poll /otp/latest until a code appears or timeout elapses."""
        deadline = time.time() + timeout
        while True:
            status, body = _get(self.base_url + "/api/v1/otp/latest", {"to": to})
            if status == 200 and isinstance(body, dict) and body.get("code"):
                return str(body["code"])
            if time.time() >= deadline:
                raise TimeoutError(f"wait_for_otp({to}): timed out after {timeout}s")
            time.sleep(poll_interval)
