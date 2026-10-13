# mocksms-testcontainers (Python)

```bash
pip install mocksms-testcontainers
```

```python
from mocksms_testcontainers import MocksmsContainer

with MocksmsContainer() as mocksms:  # in-memory store, OTP 123456
    mocksms.send_sms("+15555550100", "+15005550006", "hi")
    code = mocksms.wait_for_otp("+15005550006")
```

`MocksmsContainer(image=..., otp_code=...)` override the image and fixed
code. `base_url` is available for raw API calls.
