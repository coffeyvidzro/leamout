# Leamout Backend

Leamout is a prototype billing and monetization backend for Africa-focused products built around MoMo-style renewal flows.

The immediate goal is not to build a full billing platform. The clean MVP is to prove one renewal lifecycle end to end:

> Create one test subscription expiring in 3 days, run the scanner, see mock SMS output, open the renewal link, click mock pay, and confirm the subscription expiry date extends.

## Prototype thesis

Many small businesses need subscription billing without card rails. The prototype models a MoMo-friendly renewal flow:

1. A subscription is close to expiry.
2. The system creates a short-lived renewal token.
3. The customer receives a payment link over SMS.
4. The customer opens the link and sees a simple checkout page.
5. A mock payment succeeds.
6. The subscription is extended.

For now, every external integration should be mocked or local-first. The important thing is proving the lifecycle, boundaries, jobs, and data model.

## Tech stack

| Concern | Technology | Prototype use |
| --- | --- | --- |
| Language | Go | Backend services, workers, scheduler, HTTP handlers |
| API router | Gin | JSON APIs and renewal checkout routes |
| Database | PostgreSQL | Users, customers, subscriptions, renewal tokens, sessions, payment records |
| Cache | Redis | Session/token cache and short-lived runtime state where useful |
| Queue | River | Durable background jobs for renewal notifications and payment-side effects |
| Scheduler | robfig/cron | Hourly scanner that finds subscriptions expiring soon |
| SMS | Mock SMS | Print renewal message/link to logs/stdout |
| Payments | Mock MoMo payment | Local success path to extend subscriptions |
| Container | Docker | Reproducible server build/runtime image |

## Target MVP flow

```text
robfig/cron
  ↓
scanner runs every hour
  ↓
find subscriptions expiring in 3 days
  ↓
insert River job
  ↓
River worker creates renewal token
  ↓
Mock SMS prints message/link
  ↓
customer opens /r/:token
  ↓
Gin validates token
  ↓
show simple checkout page
  ↓
mock payment success
  ↓
extend subscription
```

## Clean MVP acceptance test

The prototype is successful when we can do this locally:

1. Start PostgreSQL, Redis, the API server, scheduler, and worker.
2. Seed one customer and one active subscription expiring exactly 3 days from now.
3. Run the scanner manually or wait for the hourly cron tick.
4. Confirm a River job is inserted for the subscription renewal.
5. Confirm the worker creates a renewal token.
6. Confirm the mock SMS prints a message containing a link like:

   ```text
   http://localhost:8080/r/<token>
   ```

7. Open the link in a browser.
8. See a minimal checkout page with subscription/customer details.
9. Click **Mock Pay**.
10. Confirm the subscription expiry date is extended by the configured billing period.

## Suggested prototype modules

### Existing/foundation modules

- `internal/modules/auth`: OAuth login and logout.
- `internal/modules/session`: Cookie-backed sessions.
- `internal/modules/user`: Current user profile endpoints.
- `internal/modules/customer`: User-scoped customer records.
- `internal/platform/queue`: River client and worker registry.
- `internal/platform/cron`: robfig/cron scheduler wrapper.

### Next MVP modules

#### `internal/modules/subscription`

Owns subscription records and renewal state.

Likely responsibilities:

- Create test subscriptions.
- List subscriptions expiring within a time window.
- Extend a subscription after successful mock payment.
- Prevent double-extension from the same renewal token/payment attempt.

Minimum fields:

- `id`
- `user_id`
- `customer_id`
- `plan_name`
- `amount`
- `currency`
- `interval_days`
- `status`
- `current_period_end`
- `created_at`
- `updated_at`

#### `internal/modules/renewal`

Owns renewal tokens and checkout links.

Likely responsibilities:

- Create a token for a subscription renewal.
- Validate `/r/:token` requests.
- Expire or mark tokens as used.
- Link token usage to payment/subscription extension.

Minimum fields:

- `id`
- `subscription_id`
- `token_hash`
- `expires_at`
- `used_at`
- `created_at`

#### `internal/modules/payment`

Owns the mock payment flow.

Likely responsibilities:

- Render/serve mock checkout actions.
- Record a mock payment success.
- Trigger subscription extension.

Minimum fields:

- `id`
- `subscription_id`
- `renewal_token_id`
- `amount`
- `currency`
- `provider`
- `status`
- `created_at`

#### `internal/modules/notification`

Owns mock SMS sending.

Likely responsibilities:

- Format renewal messages.
- Print SMS output to logs/stdout.
- Eventually swap mock sender for an SMS provider.

Example mock SMS:

```text
[MOCK SMS] To +233501234567: Your Leamout subscription expires soon. Renew here: http://localhost:8080/r/abc123
```

## Scheduler and worker design

### Scanner

The scanner should run hourly through `robfig/cron`, but it should also be callable manually for the MVP.

Scanner behavior:

1. Query active subscriptions where `current_period_end` is between now and now + 3 days.
2. Exclude subscriptions that already have an active unused renewal token for the same period.
3. Insert a River job for each subscription that needs a reminder.

### River job

The River worker should process one subscription renewal reminder at a time.

Worker behavior:

1. Load subscription and customer.
2. Create or reuse a valid renewal token.
3. Build `/r/:token` link.
4. Send mock SMS by printing the message.
5. Record enough state to avoid duplicate messages during repeated scanner runs.

## HTTP endpoints for the MVP

### Internal/API endpoints

These can be JSON endpoints protected by auth middleware:

- `POST /customers`
- `GET /customers`
- `POST /subscriptions/test-expiring-soon`
- `GET /subscriptions`
- `POST /scanner/run-once`

The manual scanner endpoint is acceptable for prototype speed. It can be removed or locked down later.

### Public renewal endpoints

These are customer-facing and should not require login:

- `GET /r/:token` — validate token and show checkout page.
- `POST /r/:token/pay` — mock payment success and extend subscription.

## Data consistency rules

Even in a prototype, the renewal path should avoid obvious billing bugs:

- A renewal token can only be used once.
- A successful mock payment should extend the subscription exactly once.
- Re-running the scanner should not create unlimited duplicate active tokens.
- Subscription extension should happen in a database transaction with payment/token updates.
- Token values should be stored hashed, not raw.

## What we are intentionally not building yet

- Real MoMo provider integration.
- Real SMS provider integration.
- Multi-organization support.
- Plan catalog complexity.
- Web dashboard polish.
- Full ledger/accounting system.
- Production-grade retry/backoff policy.

## Recommended implementation order

1. Add subscription and renewal migrations.
2. Add `subscription` repository/service for creating a test expiring subscription and extending it.
3. Add `renewal` token repository/service.
4. Add mock SMS sender.
5. Add River renewal reminder job and worker.
6. Add scanner function and cron registration.
7. Add `/r/:token` checkout page and mock pay endpoint.
8. Add a manual seed/run path for the clean MVP demo.

## Local development target

The intended local demo should eventually look like this:

```bash
# Start dependencies
# docker compose up -d

# Run migrations
# go run ./cmd/migrate up

# Start API
# go run ./cmd/server

# Start worker
# go run ./cmd/worker

# Start scheduler or trigger scanner manually
# go run ./cmd/scheduler
```

Then create one test subscription, run the scanner, copy the mock SMS link, open it, click mock pay, and verify the expiry date moved forward.
