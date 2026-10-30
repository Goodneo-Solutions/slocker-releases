<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/Goodneo-Solutions/slocker-releases/main/assets/slocker-logo-dark.svg">
  <img src="https://raw.githubusercontent.com/Goodneo-Solutions/slocker-releases/main/assets/slocker-logo.svg" alt="sLocker" height="56">
</picture>

# sLocker

Deliver to Superroute smart lockers. sLocker is the SDK for carriers and delivery partners that put parcels into
Superroute smart lockers: create a delivery task, open the doors remotely for the driver, confirm the deposit, follow
the pickup, and receive and verify the webhooks — with a mock cabinet to rehearse all of it.
Superroute SDK License — see [LICENSE](LICENSE) and [open.superroute.ca/developer/license](https://open.superroute.ca/developer/license).

| Package | Platform | Install |
|---|---|---|
| `superroute-slocker` (PyPI) | Python 3.9+ | `pip install superroute-slocker` |
| `superroute/slocker` (Packagist) | PHP 8.1+ | `composer require superroute/slocker` |
| `ca.superroute:slocker` (Maven) | Java 11+ and Android | see [Java and Kotlin](#java-and-kotlin) |
| `ca.superroute:slocker-android` (Maven) | Kotlin coroutines and OkHttp, for Android | see [Java and Kotlin](#java-and-kotlin) |
| `github.com/Goodneo-Solutions/slocker-releases/go` | Go 1.21+ | `go get github.com/Goodneo-Solutions/slocker-releases/go` |
| `Superroute.SLocker` (NuGet) | .NET 8 | `dotnet add package Superroute.SLocker` |
| `@superrouteca/slocker` (npm) | Node 20+, browsers, Deno, Bun; TypeScript types | `npm install @superrouteca/slocker` |
| `SLocker` (Swift package) | iOS 15+, macOS 12+ | `https://github.com/Goodneo-Solutions/slocker-releases` |
| `slocker` (pub.dev) | Dart 3.5+ and Flutter | `dart pub add slocker` |

Every package of a release carries the same version and covers the same operations.

## What it covers

The partner locker delivery API of Superroute (`/api/v1/third-party-delivery/…`), 40 operations, each a method:

- **Delivery tasks** — create (idempotent), list, read, cancel, rotate a code.
- **At the cabinet** — open the doors remotely with the delivery code, confirm (all or a partial delivery), abort,
  reassign a package to another compartment, cancel the placement, reopen within the 60-second correction window,
  open for pickup.
- **Scopes and locations** — the grants and sub-scopes, create / update / disable a sub-scope, the locations of a
  scope, the locations near an address or a point.
- **Usage** — summary, daily, by location.
- **Carriers** — the carrier profile, intake locations and receiving warehouses, pre-advising packages into a
  warehouse, the carrier's packages, cancelling, rerouting, and resetting a pickup code.
- **Mock cabinet** — create, list and delete mock lockers, force doors open or closed, take a cabinet offline, change
  its features and compartment sizes, seed parcels, clear them, or declare them taken.

The operations are generated from Superroute's published OpenAPI document (`specs/partner-locker-api.json`). The same
operations exist in Superroute's GraphQL API (`specs/partner-locker-graphql.json` lists them); sLocker calls REST.

## Signing in and choosing the provider

The API takes the bearer token of a Superroute API account (`POST /api/v1/user/login`; a token lasts a week). Pass a
token you already have, or sign in with the client, which keeps the token for the following calls:

```js
import { SLockerClient } from '@superrouteca/slocker';

const sl = new SLockerClient({ providerId: 4611 });          // baseUrl defaults to https://api.superroute.ca
await sl.login({ email: 'api-partner@example.com', password: process.env.SR_PASSWORD });
// later: await sl.refreshToken();  await sl.rotateToken({ gracePeriodMinutes: 60 });  await sl.logout();
```

`providerId` is your third-party delivery provider (sent as `X-Third-Party-Provider-Id`). It may be left out when the
account acts for exactly one provider; any call can name another one (`providerId` in its parameters).

## A delivery, end to end

1. **Create the task** with the order's own idempotency key. The answer carries `delivery_no`, the driver's
   `access_code` and the recipient's `pickup_code`. A repeat with the same key and the same content answers the
   original task (`replayed: true`); the same key with different content is `409 IDEMPOTENCY_CONFLICT`.
2. **At the cabinet**, the driver types the code on the screen — or your app opens the doors remotely with
   `openPartnerLockerDelivery`, which answers a `claim_token` and the compartments that opened.
3. **Confirm** with the claim token (`confirm_all`, or `accept_partial` when some packages did not fit). For a remote
   open the cabinet settles the deposit by itself once every opened door reports closed; an explicit confirm afterwards
   answers the same result.
4. **Webhooks** tell you the rest: `partner_locker.delivery.doors_opened`, `.delivered` (with the pickup code and the
   compartments), `partner_locker.pickup.completed`, `.failed`, `.expired`, `.cancelled`, `.correction_reopened`.

```js
const created = await sl.api.createPartnerLockerDelivery({
  idempotencyKey: 'FOOD-20260714-1001-v1',                    // your own, stable for this order
  body: {
    external_order_id: 'FOOD-20260714-1001',
    platform_scope_group_id: 12,
    recipient: { phone: '+14165550123' },
    package_count: 2,
  },
});
const { delivery_no, access_code } = created.data;

const opened = await sl.api.openPartnerLockerDelivery({ body: { location_id: 101, access_code } });
const { claim_token, attempt } = opened.data;
if (attempt.can_confirm) {
  await sl.api.confirmPartnerLockerDelivery({ body: { claim_token, decision: 'confirm_all' } });
}
const task = await sl.api.partnerLockerDelivery({ deliveryNo: delivery_no });
```

The same in Python:

```python
from slocker import SLockerClient, SLockerError

sl = SLockerClient(provider_id=4611)
sl.login('api-partner@example.com', password)
created = sl.create_partner_locker_delivery(
    idempotency_key='FOOD-20260714-1001-v1',
    body={'external_order_id': 'FOOD-20260714-1001', 'platform_scope_group_id': 12,
          'recipient': {'phone': '+14165550123'}, 'package_count': 2})
try:
    opened = sl.open_partner_locker_delivery(body={'location_id': 101, 'access_code': created['data']['access_code']})
    sl.confirm_partner_locker_delivery(body={'claim_token': opened['data']['claim_token'], 'decision': 'confirm_all'})
except SLockerError as e:
    if e.code == 'LOCKER_DELIVERY_IN_USE':
        ...  # another attempt is in progress: wait, or use its claim token
    raise
```

## The sandbox: a mock cabinet

A mock cabinet runs the same pipeline as a real one without hardware; a task created with `mock: true` only uses mock
cabinets, and its webhooks carry `livemode: false`, an `event_id` starting `PLE-MOCK-` and the header `X-Webhook-Test:
1` (sent to your sandbox URL when one is set). Up to 5 mock cabinets per provider, 1–60 compartments each.

```python
locker = sl.create_partner_locker_mock_locker(body={'location_id': 101, 'name': 'Dev cabinet', 'compartments': 8})
device_id = locker['data']['locker']['device_id']

task = sl.create_partner_locker_delivery(idempotency_key='TEST-1', body={
    'external_order_id': 'TEST-1', 'platform_scope_group_id': 12, 'recipient': {'email': 'qa@example.com'}, 'mock': True})
opened = sl.open_partner_locker_delivery(body={'location_id': 101, 'access_code': task['data']['access_code']})
sl.confirm_partner_locker_delivery(body={'claim_token': opened['data']['claim_token'], 'decision': 'confirm_all'})
# → webhook partner_locker.delivery.delivered, livemode false

sl.mark_partner_locker_mock_parcels_taken(device_id, body={'compartments': [1]})
# → webhook partner_locker.pickup.completed (a full pickup settlement)

sl.clear_partner_locker_mock_parcels(device_id)              # empties the cabinet without a pickup settlement
sl.set_partner_locker_mock_door(device_id, body={'compartment': 2, 'action': 'open'})   # a door left open
sl.set_partner_locker_mock_compartments(device_id, body={'compartments': [1, 2], 'grid_length_cm': 20,
    'grid_width_cm': 20, 'grid_height_cm': 20, 'weight_limit_kg': 5})                # rehearse "does not fit"
sl.delete_partner_locker_mock_locker(device_id)
```

Use the location list in the same mode as the task (`mock: true` on `partnerLockerScopeLocations` /
`partnerLockerNearbyLocations`): a live location for a mock task answers `409 LOCKER_NO_CABINET_FOR_MODE`.

## Webhooks

Verify every webhook against the **raw body** exactly as received, before parsing it. sLocker checks the timestamp
(±5 minutes by default), the `X-Webhook-Content-Digest` and the `X-Webhook-Signature`
(`v1=` base64 HMAC-SHA256 of `{timestamp}\n{X-Webhook-Delivery-Id}\n{body}` with your endpoint's signing secret), and
answers the event. During a key rotation the previous secret stays valid for 24 hours: pass both. An endpoint switched
to Standard Webhooks (`webhook-id` / `webhook-timestamp` / `webhook-signature`) or to CloudEvents structured bodies is
verified too. The same verification covers the carrier channel's `carrier_package.distribution.*` events.

Deduplicate on `eventId` (stable across retries; `deliveryId` changes on every attempt), answer 2xx within 10 seconds,
and keep sandbox events (`event.test`) apart from production.

```js
import { verifyWebhook, WebhookVerificationError } from '@superrouteca/slocker';

app.post('/webhooks/slocker', express.raw({ type: '*/*' }), async (req, res) => {
  try {
    const event = await verifyWebhook(req.body, req.headers, [process.env.SLOCKER_SECRET, process.env.SLOCKER_OLD_SECRET]);
    if (!event.test) await queue.add(event.eventId, event);
    res.sendStatus(204);
  } catch (e) {
    if (e instanceof WebhookVerificationError) return res.status(400).send(e.reason);
    throw e;
  }
});
```

| Language | Verify | Sign a test request |
|---|---|---|
| Python | `slocker.verify(body, headers, secrets)` | `slocker.sign(secret, body)` |
| PHP | `Webhooks::verify(file_get_contents('php://input'), getallheaders(), $secrets)` | `Webhooks::sign($secret, $body)` |
| Java | `Webhooks.verify(body, headers, secrets)` | `Webhooks.sign(...)` |
| Go | `slocker.Verify(body, slocker.HeaderMap(r.Header), secrets, nil)` | `slocker.Sign(secret, body, slocker.SignOptions{})` |
| C# | `Webhooks.Verify(body, headers, secrets)` | `Webhooks.Sign(secret, body)` |
| JavaScript | `await verifyWebhook(body, headers, secrets)` | `await signWebhook(secret, body)` |
| Swift | `try Webhooks.verify(body:headers:secrets:)` | `Webhooks.sign(secret:body:)` |
| Dart | `verifyWebhook(bodyBytes, headers, secrets)` | `signWebhook(secret, bodyBytes)` |

A refused webhook raises an error whose reason is one of `MISSING_HEADER`, `INVALID_TIMESTAMP`,
`TIMESTAMP_OUT_OF_TOLERANCE`, `DIGEST_MISMATCH`, `SIGNATURE_MISMATCH` or `INVALID_PAYLOAD`.

## Errors and retries

Every failed call raises one error type with the HTTP `status`, the server's stable `code`, its `message`, the field
`errors` and `retryAfter` (seconds). Act on the status and the code, never on the message. Codes the SDK adds:
`UNAUTHENTICATED` (401 without a code: sign in again), `HTTP_<status>` (an answer without a code), `NETWORK_ERROR`
(no answer; status 0) and `INVALID_ARGUMENT` (a missing path parameter or an unknown operation; nothing was sent).

Only repeatable calls are retried: reads, deletes, and the two creates that carry an idempotency key
(`createPartnerLockerDelivery`, `submitCarrierPackages`). They are repeated after a timeout, a lost connection, 408,
425, 429 or 5xx — at most `maxRetries` times (2 by default), waiting what `Retry-After` asks (at most 60 seconds) or
an exponential backoff with jitter. Opening doors, confirming, aborting, reassigning and every other change are never
repeated by the SDK: after a timeout, read the task (`partnerLockerDelivery`) before deciding.

When the SDK makes the idempotency key itself it is kept across its own retries only; give your own key (one per
order, unchanged on every retry of yours) so that a retry after a crash finds the original task.

## Each language

### Python

```python
from slocker import SLockerClient
sl = SLockerClient(token, provider_id=4611)                      # sync; urllib from the standard library
sl.partner_locker_deliveries(status='delivered', per_page=50)
sl.call('partnerLockerDeliveries', {'status': 'delivered'})       # any operation by name
```

### PHP

```php
use Superroute\SLocker\Client;
$sl = new Client($token, providerId: 4611);                       // curl
$sl->partnerLockerDeliveries(status: 'delivered', perPage: 50);
```

### Java and Kotlin

Add the Maven repository `https://goodneo-solutions.github.io/slocker-releases/maven`, then
`implementation("ca.superroute:slocker:0.1.1")` — or, in an Android app, `ca.superroute:slocker-android:0.1.1`.

```java
Client sl = Client.builder().token(token).providerId(4611L).build();   // java.net.http, or HttpURLConnection on Android
Object page = sl.partnerLockerDeliveries(Params.create().put("status", "delivered").put("per_page", 50));
```

```kotlin
val sl = SLockerAsync(Client.builder().token(token).providerId(4611L).transport(OkHttpTransport()).build())
val page = sl.api.partnerLockerDeliveries(status = "delivered", perPage = 50)   // suspend, on Dispatchers.IO
```

Answers are plain JSON values (`Map`, `List`, `String`, `Long`, `Double`, `Boolean`, `null`).

### Go

```go
sl := slocker.New(slocker.Config{Token: token, ProviderID: 4611})
page, err := sl.PartnerLockerDeliveries(ctx, slocker.Params{"status": "delivered", "per_page": 50})
var e *slocker.Error
if errors.As(err, &e) && e.Code == "LOCKER_RATE_LIMITED" { /* … */ }
```

### C#

```csharp
var sl = new SLockerClient(new SLockerOptions { Token = token, ProviderId = 4611 });
var page = await sl.PartnerLockerDeliveriesAsync(status: "delivered", perPage: 50);
```

### Swift

```swift
let sl = SLockerClient(token: token, providerId: 4611)
let page = try await sl.partnerLockerDeliveries(status: "delivered", perPage: 50)
```

### Dart and Flutter

```dart
final sl = SLockerClient(token: token, providerId: 4611);
final page = await sl.partnerLockerDeliveries(status: 'delivered', perPage: 50);
```

Every client takes its own transport (for tests, proxies or an HTTP stack the app already has), a base URL (white-label
API hosts and test environments), the retry count and the wait function.

## Development

```
specs/                       the partner locker slice of Superroute's REST and GraphQL documents, and operations.json
tools/sync_specs.py          copies the slice from a Superroute checkout (and checks no route lacks its Swagger entry)
tools/generate.py            writes every language's operation table and typed methods from the spec
tools/make_client_cases.py   writes tests/cases/client.json: requests, answers, retries, sign-in
tools/make_webhook_vectors.py writes tests/cases/webhooks.json, signed exactly as Superroute signs
tests/cases/                 the cases every language runs
```

```sh
python3 -I tools/sync_specs.py ../Superroute       # then commit specs/ and the generated files
cd python && python3 -m unittest discover -s tests
cd js && npm test
cd php && php tests/run.php
cd go && go test ./...
cd java && ./gradlew :slocker:test :slocker-android:test
dotnet test dotnet/tests/Superroute.SLocker.Tests
cd swift && swift test
cd dart && dart pub get && dart test
```

Never edit generated files (`generated/`, `Generated/`, `_operations.py`, `operations_gen.go`, `SLockerOperations.kt`):
change Superroute's spec, sync, and generate again.

## Releases

sLocker started at 0.1.0; each release adds 0.0.1 to the last one, skipping every version that contains the digit 3
or 4 (0.1.0, 0.1.1, 0.1.2, 0.1.5, 0.1.6, …, 0.1.9, 0.2.0, …). Tagging a version (`0.1.1`, no `v`) runs
`.github/workflows/release.yml`, which tests every language and then publishes:

- npm (`@superrouteca/slocker`), PyPI (`superroute-slocker`), NuGet (`Superroute.SLocker`) and pub.dev (`slocker`)
  directly, by trusted publishing;
- to the public repository `Goodneo-Solutions/slocker-releases`: the Swift package (a prebuilt XCFramework attached to
  that repository's GitHub Release of the version), the Maven repository (`maven/`, served by GitHub Pages) with
  `ca.superroute:slocker` and `slocker-android`, and the Go module (`go/`, tag `go/v<version>`);
- to the public repository `Goodneo-Solutions/slocker-php`: the PHP package at its root, tag `v<version>`, which
  Packagist reads.

This source repository stays private; the packages are public under the Superroute SDK License (see [LICENSE](LICENSE)).
