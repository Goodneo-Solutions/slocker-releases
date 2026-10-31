# 0.2.0

## SDK

### Customer side (new)
- **Smart lockers for customer apps**: the locker operations an app makes for a service provider's customers come to sLocker: parcels waiting in lockers and their pickup codes, opening the door to collect, handing a pickup over to someone else and taking it back; locker locations, their indoor guides and the customer's preferred locker; locker storage rentals (claiming, keeping, the return time, sending on, pickup sites, resending the code); and the locker shop (catalogue, orders, opening the door, refund requests, access codes).
- **Same names as in sApp**: the 29 locker methods sApp 0.2.1 had keep exactly their names and arguments, so moving from sApp is a change of package, not of call.
- **Locker resale**: list the resale terms and the person's deals, create a deal, release or revoke the buyer's code, abort, cancel or report a deal.
- **Standalone or beside sApp**: create a customer client with the app code, the service provider's client code, the app key and a token store, or pass an sApp client the app already has to share one sign-in. Sign-in, token refresh, idempotency keys, retries and errors are sApp's own.
- **Every language**: JavaScript and TypeScript, Dart, Swift, Kotlin, Java, C#, Python, PHP and Go.

### Carrier side
- **Unchanged**: the carrier and partner locker delivery API, its packages and their calls stay exactly as in 0.1.1 and do not depend on sApp.

## UI kits

### Flutter, web, React, React Native, SwiftUI, Jetpack Compose, WPF and WinForms (new)
- **Pickup list**: the person's parcels in lockers with their pickup codes, remaining parcels and expiry, and opening the door after confirming.
- **Locker map**: lockers near the person with distance, status and free compartments, search and filters for pickup, returns and free space, and the same lockers on the app's own sMap map, fitted to them; tapping a locker or its point selects it.
- **Themed by sApp**: the components use the sApp UI kit's branding, strings in 14 languages and sign-in handling, as they did in sApp.

# 0.1.1

## Licence

### Every package
- **Superroute SDK License**: from this version on the packages are licensed under the Superroute SDK License — see `LICENSE` and [open.superroute.ca/developer/license](https://open.superroute.ca/developer/license). Versions published before keep the licence they were published with.

# 0.1.0

## SDK

### Every language
- **The partner locker delivery API in nine languages**: Python, PHP, Java, Go, C# and JavaScript / TypeScript on the server; Kotlin (Android), Swift (iOS and macOS) and Dart (Flutter) in apps. Every operation of the API has its own method: delivery tasks, remote door opening and confirmation, reassigning, cancelling and correcting a placement, opening for pickup, scopes and locations, usage reports, carrier pre-advice and carrier packages, and the mock cabinet.
- **Sign-in**: sign in with the API account, refresh, rotate with a grace period, or sign out; the provider to act for is set once or per call.
- **Errors as the server names them**: every failure carries the HTTP status and the server's stable code (`LOCKER_DELIVERY_IN_USE`, `IDEMPOTENCY_CONFLICT`, …), its message, field errors and the wait the server asked for.
- **Safe retries**: only repeatable calls are retried — reads, deletes and creates that carry an idempotency key — after a timeout, a lost connection, 408, 425, 429 or 5xx, waiting as long as the server asks. Opening doors, confirming and every other change are never repeated by the SDK.
- **Idempotency**: creating a delivery task and pre-advising packages always send an idempotency key, the caller's or one made for the call and kept across its retries.
- **Webhook verification**: `partner_locker.*` and `carrier_package.distribution.*` webhooks are checked against the raw body, the content digest, the signature and the timestamp, with several secrets accepted during a key rotation, and with Standard Webhooks and CloudEvents bodies understood; sandbox events are told apart. Test requests can be signed the same way.
- **One set of test cases** run by every language: requests, answers, retries, sign-in and webhook vectors.
