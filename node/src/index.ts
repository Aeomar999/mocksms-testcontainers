import { GenericContainer, Wait, type StartedTestContainer } from "testcontainers";

export interface StartMocksmsOptions {
  /** Container image (default ghcr.io/aeomar999/commpit/mocksms:latest). */
  image?: string;
  /** Fixed OTP code for deterministic assertions (default "123456"). */
  otpCode?: string;
  /** Ready budget in ms (default 30_000). */
  timeoutMs?: number;
}

export interface WaitForOtpOptions {
  /** Wait budget in ms (default 30_000). */
  timeoutMs?: number;
  /** Poll interval in ms (default 500). */
  pollMs?: number;
  /** Project ULID; empty uses the default project. */
  project?: string;
}

export interface SendSmsInput {
  from: string;
  to: string;
  body: string;
}

async function fetchJson(url: string, init?: RequestInit): Promise<{ status: number; body: unknown }> {
  const res = await fetch(url, {
    ...init,
    headers: { "X-Mocksms": "true", "Content-Type": "application/json", ...init?.headers },
  });
  const text = await res.text();
  let body: unknown = text;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    // leave raw text
  }
  return { status: res.status, body };
}

/** Poll /otp/latest until a code appears or the budget runs out. */
export async function waitForOtp(
  baseUrl: string,
  to: string,
  opts: WaitForOtpOptions = {}
): Promise<string> {
  const timeoutMs = opts.timeoutMs ?? 30_000;
  const pollMs = opts.pollMs ?? 500;
  const deadline = Date.now() + timeoutMs;
  const params = new URLSearchParams({ to });
  if (opts.project) {
    params.set("project", opts.project);
  }
  for (;;) {
    const { status, body } = await fetchJson(
      `${baseUrl}/api/v1/otp/latest?${params.toString()}`
    );
    if (status === 200 && body && typeof body === "object" && "code" in body) {
      return (body as { code: string }).code;
    }
    if (Date.now() >= deadline) {
      throw new Error(`waitForOtp(${to}): timed out after ${timeoutMs}ms`);
    }
    await new Promise((resolve) => setTimeout(resolve, pollMs));
  }
}

export interface MocksmsContainer {
  baseUrl: string;
  smtpHost: string;
  smtpPort: number;
  stop(): Promise<void>;
  sendSms(input: SendSmsInput): Promise<unknown>;
  waitForOtp(to: string, opts?: WaitForOtpOptions): Promise<string>;
}

/** Start mocksms in a container (in-memory store) and wait until healthy. */
export async function startMocksms(opts: StartMocksmsOptions = {}): Promise<MocksmsContainer> {
  const container: StartedTestContainer = await new GenericContainer(
    opts.image ?? "ghcr.io/aeomar999/commpit/mocksms:latest"
  )
    .withEnvironment({
      MOCKSMS_MEMORY: "true",
      MOCKSMS_OTP_CODE: opts.otpCode ?? "123456",
    })
    .withExposedPorts(4010, 1025)
    .withWaitStrategy(Wait.forHttp("/api/v1/healthz", 4010))
    .withStartupTimeout(opts.timeoutMs ?? 30_000)
    .start();

  const baseUrl = `http://${container.getHost()}:${container.getMappedPort(4010)}`;
  return {
    baseUrl,
    smtpHost: container.getHost(),
    smtpPort: container.getMappedPort(1025),
    stop: () => container.stop().then(() => undefined),
    sendSms: async (input: SendSmsInput) => {
      const { status, body } = await fetchJson(`${baseUrl}/api/v1/sms`, {
        method: "POST",
        body: JSON.stringify(input),
      });
      if (status !== 201 && status !== 202) {
        throw new Error(`sendSms failed: ${status} ${JSON.stringify(body)}`);
      }
      return body;
    },
    waitForOtp: (to: string, waitOpts?: WaitForOtpOptions) =>
      waitForOtp(baseUrl, to, waitOpts),
  };
}
