# mocksms-testcontainers (Node)

```bash
npm i -D mocksms-testcontainers testcontainers
```

```ts
import { startMocksms } from "mocksms-testcontainers";

const mocksms = await startMocksms({}); // in-memory store, OTP 123456
try {
  await mocksms.sendSms({ from: "+15555550100", to: "+15005550006", body: "hi" });
  const code = await mocksms.waitForOtp("+15005550006"); // polls /otp/latest
} finally {
  await mocksms.stop();
}
```

`startMocksms({ image, otpCode, timeoutMs })`. Standalone `waitForOtp(baseUrl,
to, opts?)` is exported for existing servers. `MOCKSMS_IMAGE` env overrides
the image (CI matrix, local builds).
