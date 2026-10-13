import { createServer, type Server } from "node:http";
import { afterEach, describe, expect, it } from "vitest";
import { waitForOtp } from "../src/index.js";

let server: Server | undefined;

afterEach(async () => {
  await new Promise<void>((resolve) => {
    if (server) {
      server.close(() => resolve());
    } else {
      resolve();
    }
    server = undefined;
  });
});

function stubOtpServer(code: string | null): Promise<string> {
  return new Promise((resolve) => {
    server = createServer((req, res) => {
      if (code === null) {
        res.writeHead(404, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ error: { code: "not_found", message: "no OTP" } }));
      } else {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ code, source: "extracted" }));
      }
    });
    server.listen(0, "127.0.0.1", () => {
      const addr = server?.address();
      const port = typeof addr === "object" && addr ? addr.port : 0;
      resolve(`http://127.0.0.1:${port}`);
    });
  });
}

describe("waitForOtp", () => {
  it("returns the code when present", async () => {
    const baseUrl = await stubOtpServer("482916");
    await expect(waitForOtp(baseUrl, "+15005550006")).resolves.toBe("482916");
  });

  it("times out when no OTP arrives", async () => {
    const baseUrl = await stubOtpServer(null);
    await expect(
      waitForOtp(baseUrl, "+19999999999", { timeoutMs: 1200, pollMs: 200 })
    ).rejects.toThrow(/timed out/);
  });
});

// Container lifecycle needs a Docker daemon (CI); skipped elsewhere.
describe.skipIf(!process.env.MOCKSMS_TEST_DOCKER)("container", () => {
  it("starts and round-trips an OTP", async () => {
    const { startMocksms } = await import("../src/index.js");
    const mocksms = await startMocksms({});
    try {
      await mocksms.sendSms({
        from: "+15555550100",
        to: "+15005550006",
        body: "Your verification code is 731946",
      });
      await expect(mocksms.waitForOtp("+15005550006")).resolves.toBe("731946");
    } finally {
      await mocksms.stop();
    }
  });
});
