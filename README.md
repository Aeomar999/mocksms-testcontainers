# mocksms-testcontainers

Testcontainers modules for the [mocksms](https://github.com/Aeomar999/CommPit)
sandbox (Node, Go, Python), each with a `waitForOtp` helper. Your tests
start a real sandbox container, send SMS/OTP to it, and assert — nothing
is delivered.

## Layout

- `node/` — npm package `mocksms-testcontainers` ([README](node/README.md))
- `go/` — Go module `github.com/Aeomar999/mocksms-testcontainers/go` ([README](go/README.md))
- `python/` — pip package `mocksms-testcontainers` ([README](python/README.md))

All three default to `ghcr.io/aeomar999/commpit/mocksms:latest` with an
in-memory store and `MOCKSMS_OTP_CODE=123456`, all overridable.

## Testing

Unit tests (polling logic against stub HTTP servers) run anywhere:

```bash
# node
cd node && npm ci && npm test
# go
cd go && go test ./...
# python
cd python && pip install -e '.[test]' && pytest
```

Container round-trips need a Docker daemon and set `MOCKSMS_TEST_DOCKER=1`;
CI runs the full suites nightly and on pushes.
