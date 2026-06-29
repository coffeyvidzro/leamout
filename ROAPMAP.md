# Leamout Roadmap

A tickable roadmap for evolving Leamout from a creator-subscription MVP into precision billing and monetization infrastructure for modern African developers.

## How to use this roadmap

- Keep checklist items unchecked until the work is implemented, tested, documented, and released.
- Prefer completing roadmap items behind feature flags when they affect money movement, billing, or provider routing.
- For financial features, consider an item complete only after reconciliation, observability, rollback, and audit behavior are defined.
- Update this file in the same pull request that ships a roadmap milestone.

## Product north star

Leamout should become a mobile-money-native billing platform that helps African software businesses price, meter, charge, reconcile, and grow revenue through reliable developer APIs.

Core promises:

- Mobile-money-first collections across African markets.
- Precision metering and usage-based billing.
- Accounting-grade money movement and reconciliation.
- Developer-grade APIs, SDKs, webhooks, and sandbox tooling.
- Merchant dashboards for subscriptions, revenue, dunning, usage, wallets, and settlements.

## Phase 0 — MVP hardening

Goal: make the existing creator renewal flow dependable enough for private beta usage.

### Renewal and dunning reliability

- [ ] Add an end-to-end test for product creation, recurring price creation, customer creation, subscription creation, dunning scan, SMS reminder, checkout session creation, payment confirmation, subscription renewal, and token revocation.
- [ ] Add tests for expired, reused, revoked, and malformed dunning tokens.
- [ ] Add dunning conversion metrics: sent, clicked, checkout started, paid, failed, expired.
- [ ] Add dunning attempt transition history with actor, reason, previous status, next status, and metadata.
- [ ] Add dead-letter handling and retry visibility for failed renewal reminder jobs.

### Checkout and payment safety

- [ ] Add exact idempotency handling for checkout payment attempts.
- [ ] Store raw provider webhook payloads before processing.
- [ ] Deduplicate provider webhooks by provider event ID or provider transaction reference.
- [ ] Verify provider webhook signatures where supported.
- [ ] Add replay tooling for stored provider webhook events.
- [ ] Add payment state transition history.
- [ ] Add tests for duplicate, delayed, failed, and out-of-order payment webhooks.

### Local and production readiness

- [ ] Add a migration smoke test for all embedded migrations.
- [ ] Add documented `.env.example` files for server and web apps.
- [ ] Make optional third-party integrations no-op friendly in development.
- [ ] Add structured health checks for Postgres, Redis, queue workers, scheduler, and configured providers.
- [ ] Add CI checks for formatting, tests, vetting, and migration validation.

## Phase 1 — Developer platform foundation

Goal: make Leamout usable by external developers through clean APIs, docs, keys, and sandbox workflows.

### Accounts and environments

- [ ] Introduce organizations as the top-level ownership boundary.
- [ ] Add projects or environments under organizations.
- [ ] Add live mode and test mode data separation.
- [ ] Replace user-owned billing resources with organization/project-owned resources.
- [ ] Add team members, roles, and permissions.
- [ ] Add audit logs for sensitive actions and money-affecting changes.

### Organization access tokens and authentication

- [ ] Introduce organization access tokens with clear prefixes, such as `lmt_org_test_` and `lmt_org_live_`.
- [ ] Scope organization access tokens to organization, project, environment, and allowed capabilities.
- [ ] Support token roles for read-only analytics, billing operations, usage ingestion, checkout creation, webhook management, and administrative actions.
- [ ] Add token rotation, revocation, expiration, and emergency disable workflows.
- [ ] Track last-used timestamp, IP, geolocation context, user agent, endpoint, and request ID for every organization access token.
- [ ] Add per-token and per-organization rate limits.
- [ ] Keep personal access tokens for local/internal workflows or migrate them to organization-scoped tokens with explicit ownership.

### Geolocation and market intelligence

- [ ] Document how MaxMind and IPinfo are used to enrich request context with country, region, city, timezone, and confidence signals.
- [ ] Define fallback behavior when the local MaxMind database is missing, stale, or cannot resolve an IP address.
- [ ] Define fallback behavior when IPinfo is unavailable, rate limited, or returns incomplete data.
- [ ] Use geolocation context to improve checkout defaults, country selection, currency selection, network hints, fraud checks, and routing diagnostics.
- [ ] Add privacy and retention rules for IP-derived geolocation data.
- [ ] Add tests for MaxMind/IPinfo provider selection, failure behavior, and request-context propagation.

### Developer experience

- [ ] Publish an OpenAPI specification for public APIs.
- [ ] Add generated SDKs for Go and Node.js.
- [ ] Add webhook signing secrets and verification examples.
- [ ] Add a sandbox provider simulator for payments and SMS.
- [ ] Add API examples for common flows: subscription checkout, usage event ingestion, prepaid credits, renewal recovery, and webhook handling.
- [ ] Add a CLI for local API testing and sandbox event simulation.

## Phase 2 — Precision metering and usage billing

Goal: support modern SaaS, AI, API, creator, community, and digital-service monetization models.

### Usage ingestion

- [ ] Return per-event success, duplicate, and failure results for batch usage ingestion.
- [ ] Add explicit idempotency keys for usage ingestion requests.
- [ ] Add request fingerprint conflict detection for reused idempotency keys.
- [ ] Define allowed lateness windows for usage events.
- [ ] Add late-event adjustment policies.
- [ ] Add immutable correction or adjustment events instead of usage event mutation.
- [ ] Add backfill and replay tooling.

### Metering

- [ ] Add meter preview APIs that show whether an event would match a meter and how quantity would be calculated.
- [ ] Add schema validation for event names and metadata used by meters.
- [ ] Add aggregation windows for hourly, daily, monthly, and billing-period rollups.
- [ ] Add threshold alerts for usage, balance, quota, and overage risk.
- [ ] Add customer usage summaries suitable for customer portals and dashboards.
- [ ] Add tests for count, sum, max, min, average, and unique aggregations.

### Rating and pricing

- [ ] Add a rating engine that converts metered usage into billable line items.
- [ ] Support flat-rate, per-unit, package, tiered, volume, graduated, and overage pricing.
- [ ] Support prepaid credits, postpaid invoices, and hybrid prepaid/postpaid plans.
- [ ] Support custom billing intervals and billing-cycle anchors.
- [ ] Support trials, grace periods, and plan changes.
- [ ] Support proration for upgrades, downgrades, and mid-cycle changes.

## Phase 3 — Accounting-grade ledger and invoicing

Goal: make Leamout trustworthy for real money movement, balances, merchant payouts, taxes, refunds, and audits.

### Money model

- [ ] Introduce a canonical money type using integer minor units and required currency.
- [ ] Enforce currency precision rules, including zero-decimal currencies such as XOF and XAF.
- [ ] Remove or isolate floating-point arithmetic from money movement.
- [ ] Introduce a decimal or fixed-point quantity type for billable usage quantities.

### Double-entry ledger

- [ ] Add ledger accounts for customers, merchants, platform revenue, fees, provider clearing, taxes, refunds, and settlements.
- [ ] Add balanced ledger transactions with debit and credit postings.
- [ ] Reject any ledger transaction where debits and credits do not balance.
- [ ] Add immutable ledger entries with correction entries instead of destructive edits.
- [ ] Add balance snapshots for efficient reads.
- [ ] Add ledger reconciliation tests and invariants.

### Invoicing

- [ ] Add invoices and invoice line items.
- [ ] Add invoice statuses: draft, open, paid, void, uncollectible, refunded, partially paid.
- [ ] Add receipts for successful collections.
- [ ] Add credit notes and customer balance adjustments.
- [ ] Add invoice finalization jobs.
- [ ] Add hosted invoice and receipt pages.
- [ ] Add invoice webhooks.

### Tax and compliance hooks

- [ ] Add tax rate configuration by country and product category.
- [ ] Add tax-inclusive and tax-exclusive pricing support.
- [ ] Add invoice tax line calculation.
- [ ] Add exportable reports for finance and compliance teams.

## Phase 4 — African payment orchestration

Goal: maximize payment success and margin across countries, mobile networks, and providers.

### Runtime provider routing

- [ ] Move provider route and fee configuration from hardcoded defaults into database-managed configuration.
- [ ] Add admin APIs for enabling, disabling, and reprioritizing routes.
- [ ] Add route rules by country, network, currency, amount band, merchant, and environment.
- [ ] Add provider health checks.
- [ ] Add route scoring based on success rate, fee, availability, settlement speed, and amount limits.
- [ ] Add automatic failover for degraded providers.
- [ ] Add route decision logs for debugging and audits.

### Provider operations

- [ ] Add pending-payment polling for providers that support transaction lookup.
- [ ] Add provider timeout and stale-pending policies.
- [ ] Add provider-specific error normalization.
- [ ] Add retry policies by failure type.
- [ ] Add provider capability metadata: countries, networks, currencies, min/max amounts, collection support, payout support, refund support, and webhook behavior.

### Settlement and reconciliation

- [ ] Add settlement batch imports.
- [ ] Match settlement lines to internal transactions.
- [ ] Detect missing webhooks, unmatched settlements, duplicate captures, reversals, and amount mismatches.
- [ ] Add reconciliation exception workflows.
- [ ] Add merchant settlement statements.
- [ ] Add platform revenue and provider fee reports.

## Phase 5 — Monetization products

Goal: help developers and creators launch richer paid products without rebuilding billing logic.

### Hosted experiences

- [ ] Build the hosted checkout UI.
- [ ] Build hosted pricing pages.
- [ ] Build a customer portal for payment status, subscriptions, invoices, receipts, and usage.
- [ ] Build secure short-domain routing for checkout and dunning links.
- [ ] Add embeddable payment links and buttons.

### Subscription and entitlement features

- [ ] Add subscription items for multiple prices per subscription.
- [ ] Add add-ons and seat-based billing.
- [ ] Add entitlement checks API.
- [ ] Add license keys and activation tracking.
- [ ] Add plan limits and feature gates.
- [ ] Add quota reset jobs by billing period.

### Growth and revenue features

- [ ] Add coupons and discounts.
- [ ] Add referral or affiliate tracking.
- [ ] Add churn analytics.
- [ ] Add failed-payment recovery analytics.
- [ ] Add revenue cohorts by country, network, plan, and acquisition channel.
- [ ] Add creator/developer revenue dashboards.

## Cross-cutting engineering work

### Observability

- [ ] Add structured logs with request ID, organization ID, project ID, payment ID, checkout ID, invoice ID, provider reference, and geolocation provider where available.
- [ ] Add metrics for payment success rate by provider, network, country, amount band, and currency.
- [ ] Add metrics for webhook latency and failure rate.
- [ ] Add metrics for queue lag, job retries, and dead-letter count.
- [ ] Add metrics for usage ingestion throughput and duplicate rate.
- [ ] Add traces across checkout, payment, webhook, ledger, geolocation enrichment, and reconciliation flows.
- [ ] Add dashboards and alerts for money-affecting incidents.

### Security

- [ ] Add explicit rules for when IP-derived geolocation can influence risk scoring, provider routing, and checkout UX without blocking legitimate customers.
- [ ] Add webhook signature verification documentation and examples.
- [ ] Add secret rotation procedures.
- [ ] Add sensitive-data redaction in logs.
- [ ] Add least-privilege database roles for app, migrations, workers, and read-only analytics.
- [ ] Add fraud and abuse controls around checkout, SMS, and payment attempts.
- [ ] Add security review checklists for provider integrations.

### Testing

- [ ] Add contract tests for payment providers.
- [ ] Add golden tests for fee calculation and route selection.
- [ ] Add property tests for ledger balancing.
- [ ] Add concurrency tests for usage credit consumption and duplicate webhooks.
- [ ] Add integration tests with Postgres and Redis.
- [ ] Add load tests for usage ingestion.

### Documentation

- [ ] Add architecture decision records for billing, ledger, provider routing, idempotency, and reconciliation.
- [ ] Add API guides for subscriptions, usage billing, prepaid credits, checkout, webhooks, and reconciliation.
- [ ] Add runbooks for payment incidents, provider outages, stuck pending payments, failed billing jobs, and ledger correction.
- [ ] Add glossary definitions for money, balance, credit, invoice, payment, transaction, settlement, reconciliation, usage event, meter, entitlement, and subscription.

## Definition of done for financial roadmap items

A financial roadmap item is done only when all applicable checks are true:

- [ ] Data model and migrations are complete.
- [ ] Service logic is implemented with idempotency.
- [ ] State transitions are validated.
- [ ] Ledger impact is defined and tested.
- [ ] Webhook and retry behavior is documented.
- [ ] Reconciliation behavior is documented.
- [ ] Metrics, logs, and traces are added.
- [ ] Tests cover success, duplicate, failure, delayed, and out-of-order scenarios.
- [ ] Documentation and examples are updated.
- [ ] Rollout and rollback plan exists.
