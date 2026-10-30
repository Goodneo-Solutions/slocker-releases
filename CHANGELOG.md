# Unreleased

## Licence

### Every package
- **Superroute SDK License**: from the next version the packages are licensed under the Superroute SDK License — see `LICENSE` and [open.superroute.ca/developer/license](https://open.superroute.ca/developer/license). Versions published before keep the licence they were published with.

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
