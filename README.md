# Leamout Backend

Leamout is a mobile-money-first billing and monetization backend for African creators and developers.

The backend now covers more than the original renewal prototype. It includes products, prices, customers, subscriptions, checkout sessions, payment routing, billing orchestration, dunning recovery, communication credits, access checks, benefits, prepaid usage credits, usage events, customer meters, wallets, transactions, cron scheduling, and River workers.

## Current status

| Area | Status |
| --- | --- |
| Product, price, customer, and subscription modules | Built |
| Checkout session lifecycle | Built |
| Payment routing foundation with PawaPay and Tola adapters | Built foundation |
| Billing orchestration module | Built foundation |
| Dunning state module | Built |
| Dunning workflow engine with cron scan and River reminder worker | Built |
| Communication credits and SMS routing/debiting | Built foundation |
| Access Check API and legacy entitlement check alias | Built |
| Benefits, benefit grants, customer meters, and prepaid usage credits | Built foundation |
| Usage event ingestion and prepaid credit deduction | Built foundation |
| Wallet and transaction settlement records | Built foundation |
| Provider webhook hardening and reconciliation | Current hardening work |
| Hosted checkout UI, dashboard, and production short-domain routing | Later product work |

## Architecture snapshot

Leamout separates infrastructure, workflows, state modules, and paid-business orchestration.

```text
cmd/server
  -> HTTP API
  -> modules/*
  -> payment, checkout, and billing wiring

cmd/scheduler
  -> platform/cron
  -> workflows/dunning.Scanner
  -> platform/queue.Insert

cmd/worker
  -> platform/queue
  -> workflows/dunning.SendReminderWorker

platform/cron
  -> robfig/cron wrapper only

platform/queue
  -> River client, workers, start/stop only

workflows/dunning
  -> system dunning engine
  -> scans subscriptions, enqueues reminders, sends SMS, records retry visibility

modules/dunning
  -> developer-facing dunning state/API module
  -> attempts, tokens, transitions, metrics, failure visibility

modules/billing
  -> paid flow orchestration
  -> captured payment settlement, checkout completion, subscription renewal, benefits, usage credits
```

The rule is:

```text
modules own state.
workflows own system automation.
platform owns infrastructure.
billing owns paid-business orchestration.
```

## Main renewal and dunning flow

```text
Creator creates product + recurring price
  ↓
Creator creates customer + active subscription
  ↓
Creator tops up communication credits
  ↓
cmd/scheduler runs cron
  ↓
workflows/dunning.Scanner finds subscriptions due soon
  ↓
River job is inserted
  ↓
cmd/worker runs workflows/dunning.SendReminderWorker
  ↓
Dunning attempt and token are created/reused
  ↓
SMS is sent and communication credits are debited
  ↓
Customer opens /v1/dunning/:token
  ↓
Dunning module records token use and creates renewal checkout
  ↓
Customer pays through /v1/checkout/:clientSecret/pay
  ↓
Payment capture goes to billing.SettleCapturedPayment
  ↓
Billing creates transaction/wallet records
  ↓
Billing completes checkout
  ↓
Billing renews subscription, marks dunning attempt paid, revokes token, grants benefits, and applies usage credits
```

## Core product concepts

| Concept | Meaning |
| --- | --- |
| Products | Creator/developer-owned things being sold. |
| Prices | One-time, recurring, or usage price definitions. MVP renewal uses recurring prices. |
| Customers | User-scoped customer records with phone numbers and external IDs. |
| Subscriptions | Active customer subscriptions tied to prices. |
| Checkout sessions | Public payment or renewal sessions with client secrets. |
| Payments | Provider-facing collection attempts and payment records. |
| Billing | Orchestrates successful paid flows across checkout, payment, subscription, dunning, benefits, usage credits, wallet, and transactions. |
| Dunning attempts | System-created renewal recovery attempts. |
| Dunning tokens | Short-lived hashed recovery tokens used in SMS links. |
| Communication credits | Prepaid creator balance used for outbound SMS. |
| Benefits | Product access or usage-credit grants attached to products. |
| Access checks | Read-only checks for whether a customer can use a feature or benefit. |
| Usage events | Developer-ingested usage records that can deduct prepaid usage credits. |
| Customer meters | Current prepaid usage-credit balances per customer/meter. |
| Wallets and transactions | Merchant-side money movement records after captured payments. |
| Personal access tokens | User-scoped API keys for local/internal workflows. |

## Access and prepaid usage credits

The public language is **Access Check** and **Usage Credits**.

```text
POST /v1/access/check
POST /v1/entitlements/check   legacy alias
POST /v1/events/ingest        record usage and deduct matched credits
```

Mental model:

```text
/v1/access/check = Can this customer use it?
/v1/events/ingest = This customer used it; record usage and deduct credits if matched.
```

See:

```text
docs/access-and-usage-credits.md
```

## Tech stack

| Concern | Technology |
| --- | --- |
| Language | Go |
| Router | Gin |
| Database | PostgreSQL |
| Queue | River |
| Scheduler | robfig/cron |
| Support dependency | Redis |
| Payments | Internal payment router with PawaPay and Tola adapters |
| SMS | Internal SMS orchestration with provider routing and local mock support |
| Auth | OAuth sessions and user-scoped personal access tokens |
| Local dependencies | Docker Compose |

## Important directories

```text
server/cmd/server         HTTP API process
server/cmd/scheduler      cron scheduler process
server/cmd/worker         River worker process
server/cmd/migrate        database and River migration runner

server/internal/modules   developer-facing domain modules
server/internal/workflows system workflow engines
server/internal/platform  infrastructure wrappers
server/internal/payment   core payment router/providers
server/internal/sms       SMS routing/providers
```

Current dunning split:

```text
server/internal/modules/dunning
  model.go
  repository.go
  service.go
  handler.go
  routes.go

server/internal/workflows/dunning
  scanner.go
  reminder_worker.go
```

## Local setup

Start dependencies:

```powershell
docker compose up -d
```

Set local environment variables in `.env`:

```env
APP_ENV=development
PORT=8080
DATABASE_URL=postgres://leamout:leamout@localhost:5432/leamout?sslmode=disable
REDIS_URL=redis://localhost:6379/0

API_BASE_URL=http://localhost:8080
FRONTEND_BASE_URL=http://localhost:3000
SHORT_BASE_URL=http://localhost:3000

TRUSTED_PROXIES=127.0.0.1,::1
GEOIP_DATABASE_PATH=./assets/GeoLite2-City.mmdb
```

Run migrations:

```powershell
cd server
go run ./cmd/migrate up
```

Start runtime processes in separate terminals:

```powershell
cd server
go run ./cmd/server

cd server
go run ./cmd/worker

cd server
go run ./cmd/scheduler
```

## Useful test commands

```powershell
cd server

go test ./internal/modules/billing
go test ./internal/modules/dunning
go test ./internal/workflows/dunning
go test ./...
```

Some integration tests require `DATABASE_URL`.

## Important routes

### Auth and API keys

```text
GET    /v1/auth/google
GET    /v1/auth/google/callback
GET    /v1/auth/github
GET    /v1/auth/github/callback
POST   /v1/auth/logout

GET    /v1/personal-access-tokens
POST   /v1/personal-access-tokens
DELETE /v1/personal-access-tokens/:id
```

### Products, customers, subscriptions

```text
POST   /v1/products
GET    /v1/products
GET    /v1/products/:id
PATCH  /v1/products/:id
DELETE /v1/products/:id

POST   /v1/customers
GET    /v1/customers
GET    /v1/customers/:id
PATCH  /v1/customers/:id
DELETE /v1/customers/:id

POST   /v1/subscriptions
GET    /v1/subscriptions
GET    /v1/subscriptions/:id
PATCH  /v1/subscriptions/:id
DELETE /v1/subscriptions/:id
```

### Checkout, payments, and dunning

```text
GET    /v1/checkouts
POST   /v1/checkouts
GET    /v1/checkouts/:id
PATCH  /v1/checkouts/:id

GET    /v1/checkout/:clientSecret
POST   /v1/checkout/:clientSecret/pay

GET    /v1/payments
GET    /v1/payments/:id

GET    /v1/dunning/:token
GET    /v1/dunning-events
GET    /v1/dunning-events/metrics
GET    /v1/dunning-events/reminder-jobs/failures
GET    /v1/dunning-events/:id/transitions
GET    /v1/dunning-events/:id
```

### Credits, access, and usage

```text
GET    /v1/credits
GET    /v1/credits/ledger
POST   /v1/credits/topup

POST   /v1/access/check
POST   /v1/entitlements/check

POST   /v1/events/ingest
GET    /v1/events
GET    /v1/events/:id
```

## URL design

Use separate base URLs for separate responsibilities.

```env
API_BASE_URL=https://api.leamout.com
FRONTEND_BASE_URL=https://leamout.com
SHORT_BASE_URL=https://lmt.com
```

```text
API_BASE_URL       OAuth callbacks and backend-owned URLs
FRONTEND_BASE_URL  hosted checkout and dashboard redirects
SHORT_BASE_URL     SMS recovery links such as /r/<token>
```

## Current engineering focus

The old README milestone about a Next.js checkout path is no longer the right next milestone.

Current hardening focus comes from `ROADMAP.md` Phase 0:

```text
payment status transitions
checkout status transitions
subscription status transitions
checkout/payment idempotency
provider webhook storage and deduplication
provider webhook signature verification
stored webhook replay tooling
payment transition history
duplicate, delayed, failed, and out-of-order payment webhook tests
migration smoke tests
health checks
CI for formatting, tests, vetting, and migration validation
```

Product/UI work such as hosted checkout, dashboards, customer portal, and production short-domain routing remains roadmap work.

## Not production-complete yet

```text
provider webhook hardening
provider reconciliation and settlement imports
hosted checkout UI
full creator/developer dashboard
short-domain production deployment
organization/project ownership model
organization-scoped access tokens
advanced usage billing/rating engine
postpaid invoicing
WhatsApp Business API
```
