# Access + Usage Credits

Leamout exposes simple product language to developers:

- **Access Check** answers: can this customer use this feature right now?
- **Usage Events** record what the customer did.
- **Usage Credits** track credit grants, debits, and balances.
- **Customer Meters** show the current balance for a customer and meter.

Internally, the server may still use the `entitlement` package name, but public docs should prefer **Access**.

## Access Check

Use Access Check when an application needs to allow or block a feature, content item, or credit-backed action.

```http
POST /v1/access/check
```

Example request:

```json
{
  "external_customer_id": "user_123",
  "code": "premium_downloads"
}
```

Example response:

```json
{
  "allowed": true,
  "code": "premium_downloads",
  "type": "feature",
  "reason": "active_grant",
  "required": 1
}
```

For credit-backed features, include `quantity`.

```json
{
  "external_customer_id": "user_123",
  "code": "api_calls",
  "quantity": 500
}
```

Access Check is read-only. It does not deduct credits.

## Backward compatibility

The older endpoint remains available:

```http
POST /v1/entitlements/check
```

Use `/v1/access/check` in new docs, examples, SDKs, and public product language.

## Usage Events

Use Usage Events when the customer actually used something and Leamout should record usage.

```http
POST /v1/events/ingest
```

Example request:

```json
{
  "events": [
    {
      "name": "api_call",
      "external_customer_id": "user_123",
      "external_id": "evt_001",
      "metadata": {
        "quantity": 500
      }
    }
  ]
}
```

If the event matches a credited meter, Leamout deducts from usage credits and refreshes the customer meter balance.

## Do not expose Entitlement Consume

Avoid introducing this public endpoint for now:

```http
POST /v1/entitlements/consume
```

For the MVP, keep the split simple:

```text
/v1/access/check = can the customer use it?
/v1/events/ingest = the customer used it, record usage and deduct credits if applicable
```

This keeps Leamout focused on prepaid usage credits instead of full usage billing.

## Naming guide

| Avoid in public docs | Use instead |
| --- | --- |
| Entitlement Check | Access Check |
| Entitlements | Access |
| Entitlement Consume | Usage Events / Usage Credit Deduction |
| Metered Entitlements | Usage Credits |

## Core flow

```text
Developer creates a product
  ↓
Developer attaches benefits
  ↓
Customer pays
  ↓
Leamout grants access and usage credits
  ↓
Developer calls /v1/access/check before allowing access
  ↓
Developer sends usage events to /v1/events/ingest
  ↓
Leamout deducts usage credits and updates customer meters
```
