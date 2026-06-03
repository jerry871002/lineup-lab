# XSS Hardening

This document records the baseline XSS mitigations for the gateway-routed
browser app under `#104`.

## Why It Matters

The auth model uses an `HttpOnly` session cookie plus a readable CSRF cookie.
That is a good browser-session shape, but it assumes the app also reduces XSS
risk. If attacker-controlled JavaScript runs in the Lineup Lab origin, it may be
able to read the CSRF cookie and perform authenticated actions through the
trusted frontend path.

XSS protection does not replace session validation, CSRF checks, or
authorization. It limits the chance that attacker-controlled script can run in
the first place.

## Gateway Headers

The gateway sets baseline browser security headers:

- `Content-Security-Policy`
- `X-Content-Type-Options`
- `X-Frame-Options`
- `Referrer-Policy`
- `Cross-Origin-Opener-Policy`
- `Cross-Origin-Resource-Policy`
- `Permissions-Policy`

The current CSP is intentionally conservative:

```text
default-src 'self';
script-src 'self';
style-src 'self';
img-src 'self' data:;
font-src 'self';
connect-src 'self';
object-src 'none';
base-uri 'self';
form-action 'self';
frame-ancestors 'none';
manifest-src 'self'
```

This policy allows the built frontend to load assets from the gateway origin and
to call same-origin `/api/*` routes. It blocks inline scripts, third-party
scripts, plugin/object embeds, and framing.

## Frontend Rules

The frontend should keep relying on React's default text escaping.

Avoid these patterns unless a future PR adds a reviewed sanitizer and CSP update:

- `dangerouslySetInnerHTML`
- direct `innerHTML` assignment
- `eval()` or `new Function()`
- rendering untrusted URL/query/body values as HTML
- adding third-party scripts without revisiting the CSP

Rich text or user-authored HTML should be treated as a separate security design
task, not a small rendering detail.

## Current Frontend Check

The current frontend source does not use obvious raw-HTML or dynamic-code
patterns such as `dangerouslySetInnerHTML`, `innerHTML`, `eval()`, or
`new Function()`.

That scan is not a permanent guarantee. It is a baseline check that should be
repeated when adding new rendering features or dependencies.
