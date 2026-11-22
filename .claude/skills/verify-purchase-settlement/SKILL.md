---
name: verify-purchase-settlement
description: Check that one payment extends a subscription at most once. Use when a change touches PurchaseRepository.Settle, PaymentService.ProcessPurchaseById, a payment webhook (YooKassa, Platega, WATA, Tribute), the Telegram Stars success handler, or the CryptoPay invoice cron.
---

# Verify purchase settlement

**Property.** One purchase extends the subscription in Remnawave at most once, however many deliveries of its payment arrive and in whatever order.
**Harm when broken.** The payer gets the paid months two or more times and sees the activation message more than once.

Deliveries that race: a provider webhook and its redelivery, two overlapping CryptoPay cron ticks, a Telegram update replayed after a restart. Before `Settle` existed, each delivery read the status with `FindById` and the row stayed unlocked until `MarkAsPaid`, so four concurrent Platega callbacks gave four extensions.

## Checks

Run both. Each runs in CI (`.github/workflows/ci.yml`).

1. Model: `formal/purchase/check.sh` (needs Java 11+ and curl; it downloads a pinned `tla2tools.jar` and checks its SHA-256).
   - `Settle.cfg` models the code and must pass.
   - `CheckThenAct.cfg` models the old flow and must fail on `ExtendedAtMostOnce`. If it passes, the invariant lost its teeth.
2. Code: `TEST_DATABASE_URL=postgres://user:pass@host:5432/postgres?sslmode=disable go test ./internal/payment -run TestDuplicatePlategaCallbacksExtendSubscriptionOnce -count=3`. It sends four concurrent Platega callbacks and one redelivery through the real webhook handler, `PaymentService` and Postgres, against fake Remnawave and Telegram servers, and counts extensions and activation messages.

## Coverage

| Model element | Code |
| --- | --- |
| `Read` with `Locked = TRUE` | `SELECT ... FOR UPDATE` and the status check in `PurchaseRepository.Settle` (`internal/database/purchase.go`) |
| `Extend`, `ExtendFails` | `apply` in `PaymentService.ProcessPurchaseById` calling `remnawave.Client.CreateOrUpdateUser` (`internal/payment/payment.go`) |
| `MarkPaid` | the `UPDATE purchase SET status` and `COMMIT` in `Settle` |
| `Read` with `Locked = FALSE` | `FindById` plus the status check in `internal/platega/webhook.go`, `internal/yookasa/webhook.go`, `internal/wata/webhook.go` |

Sync commit: `ddc0197` plus the change that added `Settle`. Move it forward only after you confirm the table still describes the code at the new commit.

## Reading a result

- `Settle` fails: the trace lists states. Read `pc` per delivery and `extensions`. Two deliveries in `"extend"` at once means the lock no longer spans the check and the update. Find the code change that released it early.
- The Go test fails with `Remnawave got N subscription extensions`: some path settles a purchase without `Settle`, or `Settle` stopped locking. `grep -rn "PurchaseStatusPaid" internal` lists every writer of the paid status. Only `Settle` may write it.
- Both pass after an intended redesign: update the model and the coverage table in the same PR.

## Not covered

- A crash or failed `COMMIT` after Remnawave accepted the extension. The purchase stays pending and the next delivery extends again. This is at-least-once, and only an idempotency key on the Remnawave side removes it.
- Tribute `new_subscription` webhooks. Each creates a new purchase row, so redelivery of one Tribute webhook creates two purchases. Deduplicating needs a Tribute event id stored with the purchase.
- Referral bonuses, which run after settlement through `ReferralRepository.MarkBonusGranted` without a lock.
