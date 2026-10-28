# User-owned resource authorization

This contract defines the identity boundary for #34 and saved lineups under #35.
The internal auth verifier and gateway enforcement protect the internal lineups
service. Browser requests use `/api/lineups` for saved lineups and run requests,
and `/api/simulations` for personal simulation history.

## Ownership

- `auth` owns users, sessions, session expiry/revocation, and CSRF verification.
- `lineups`, a new internal service, owns saved lineups and their authorization.
- `stats` owns batting records; `simulation` owns simulation behavior.
- Each service writes only its own tables. Sharing Postgres does not transfer
  ownership of auth records or the session signing secret.

## Gateway to auth contract

For every protected resource request, the gateway must call
`GET /internal/auth/verify` on the internal auth service before forwarding the
request to the resource service. This endpoint has no browser-facing `/api`
route and must not be exposed by a public proxy rule. Auth must remain on the
internal network.

The gateway supplies these headers to the verification subrequest:

- `Cookie`: the browser's original cookies, including `lineup_lab_session` and,
  for writes, `lineup_lab_csrf` (names are configurable in auth).
- `X-Original-Method`: the actual original HTTP method, set by the gateway and
  replacing any client-supplied value. The auth subrequest itself uses GET.
- `X-CSRF-Token`: the browser's CSRF header for state-changing requests.

Auth checks the active, unexpired, unrevoked session and its user on every call.
Only uppercase GET, HEAD, OPTIONS, POST, PUT, PATCH, and DELETE are accepted as
original methods. Missing or unsupported methods fail closed with 403. POST,
PUT, PATCH, and DELETE additionally require a matching CSRF cookie and header,
and a valid existing HMAC signature tied to the same active session. Safe
methods require no CSRF token.

Success returns 204, an empty body, and `X-Authenticated-User-ID` containing the
canonical decimal user ID from the session's user record. Client identity
headers are ignored by auth. Missing, malformed, unknown, expired, or revoked
sessions, and missing users, return 401. Invalid CSRF returns 403. Failure
responses never contain the authenticated identity header. Auth/database errors
must deny forwarding rather than bypass verification.

The gateway must capture the verifier's response identity and overwrite
`X-Authenticated-User-ID` on the downstream request. It must discard other
client-supplied identity headers, strip the Cookie header before forwarding to
lineups, and prevent direct public access to the internal service. It must not
reuse a previous request's verification result or cache a successful check
across session revocation. The downstream service requires the validated
identity header, rejects missing or malformed IDs, and never accepts a user ID
from a resource payload or query parameter as authorization.

## Resource service contract

Every saved-lineup operation uses the authenticated user ID as its owner:

- Create assigns the authenticated owner on the server.
- List queries filter by that owner before applying team/season filters.
- Read, update, and delete query by both resource ID and owner ID.
- Updates cannot reassign ownership.
- Missing resources and resources belonging to someone else both return 404.
- Batch operations must apply the same ownership predicate to every item.

The lineups service does not query session tables or receive the session HMAC
secret. It must be reachable only through the trusted gateway; identity headers
alone are not authentication on a publicly exposed service. Table migrations
belong to the lineups service and follow ADR 0003.

## Integration acceptance checks

Before protected resources are considered complete, verify through the actual
gateway that anonymous, expired, and revoked sessions cannot reach lineups;
forged identity headers cannot select another owner; write requests with
missing, mismatched, tampered, or another session's CSRF token are rejected;
and each owner can only list/read/update/delete their own resources. Also
verify that nonexistent and cross-owner IDs both return 404 and that neither
the internal verifier nor lineups is publicly routed.

The auth tests cover the verifier's session/CSRF decisions using an in-memory
database double and the real session and signing helpers. The Compose workflow
also runs `python3 gateway/tests/lineups_smoke.py` through the real nginx, auth,
lineups, stats, simulation, and Postgres services. It verifies signed-cookie
handling, missing/tampered/cross-session CSRF rejection, original-method spoof
rejection, identity replacement, owner CRUD isolation, revision conflicts, and
revoked sessions. Compose configuration checks ensure auth and lineups publish
no host ports and that lineups receives no session signing secret.

The protected location captures the original method before `auth_request` makes
its GET subrequest. Setting `X-Original-Method: GET` on a browser POST cannot
skip CSRF checks. Only auth receives the browser cookies and CSRF header;
lineups receives the canonical identity and no Cookie, Authorization, CSRF, or
client identity alias headers. Failed auth never falls through to frontend HTML.
