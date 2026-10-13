# mocksms-testcontainers/go

```bash
go get github.com/Aeomar999/mocksms-testcontainers/go
```

```go
c, err := mocksms.Start(ctx, mocksms.Options{})
if err != nil { t.Fatal(err) }
defer c.Stop(ctx)

if err := c.SendSMS(ctx, "+15555550100", "+15005550006", "hi"); err != nil { t.Fatal(err) }
code, err := c.WaitForOTP(ctx, "+15005550006", 30*time.Second)
```

`Options{Image, OTPCode}` override the image and fixed code. `MOCKSMS_IMAGE`
env overrides the image (CI matrix, local builds).
