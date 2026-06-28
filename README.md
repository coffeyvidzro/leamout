# Leamout Backend

Leamout is an MVP billing and monetization platform for African digital creators built on PawaPay and Tola Mobile.

The backend implements the core renewal lifecycle for creator subscriptions: a creator creates products, prices, customers, subscriptions, and communication credits; the scheduler finds subscriptions nearing expiry; the worker sends paid SMS renewal reminders; customers open secure recovery links; checkout sessions are created; payment confirmation renews subscriptions; credits are debited; and dunning attempts are marked paid.

## Current status

| Area | Status |
| --- | --- |
| Backend MVP core | Achieved |
| Product, price, customer, and subscription APIs | Achieved |
| Dunning scanner, queue, worker, and recovery links | Achieved |
| Communication credits and SMS debit ledger | Achieved |
| Checkout session lifecycle | Achieved |
| PawaPay and Tola Mobile payment foundation | MVP target |
| Next.js checkout UI | Next |
| Production short-domain routing | Next |
| Full creator dashboard | Later |

## Product thesis

Many African digital creators need subscription billing and renewal automation without assuming card-first payment rails.

Leamout focuses on mobile-money-first monetization for creators who sell memberships, communities, courses, media access, digital services, or recurring content products. The MVP direction is simple:

1. A creator defines products and prices.
2. A customer has an active subscription.
3. A scanner finds subscriptions nearing expiry.
4. A River worker sends a renewal SMS.
5. SMS cost is deducted from the creator's prepaid communication credits.
6. The customer opens a secure short recovery link.
7. The backend creates a checkout session.
8. The checkout is confirmed through the payment layer.
9. The subscription renews.
10. The dunning attempt is marked paid and the token cannot be reused.

## Core flow

```text
Creator auth / PAT
  ↓
Create product + recurring price
  ↓
Create customer
  ↓
Create active subscription
  ↓
Top up communication credits
  ↓
robfig/cron scanner finds subscription due within window
  ↓
River job is enqueued
  ↓
Worker creates/reuses dunning attempt and token
  ↓
SMS service routes message and debits credits
  ↓
Customer receives short renewal link
  ↓
GET /v1/dunning/:token creates checkout session
  ↓
Backend redirects to frontend checkout URL
  ↓
Customer confirms checkout
  ↓
Payment confirmation renews subscription
  ↓
Dunning attempt becomes paid
  ↓
Dunning token is revoked and cannot be reused
```

## Tech stack

| Concern | Technology | Use |
| --- | --- | --- |
| Language | Go | Backend services, workers, scheduler, HTTP handlers |
| API router | Gin | JSON APIs and public recovery/checkout endpoints |
| Database | PostgreSQL | Users, sessions, products, prices, customers, subscriptions, dunning, checkout, credits, PATs |
| Queue | River | Durable background jobs for dunning reminders |
| Scheduler | robfig/cron | Subscription scanner |
| Payments | PawaPay, Tola Mobile | Mobile-money-first checkout and renewal foundation |
| SMS | Internal SMS orchestration | Mock provider locally, provider routing for production SMS |
| Auth | OAuth sessions + PATs | Browser login and API testing/integration auth |
| Geo/IP/security | MaxMind/IPinfo + Arcjet | Request context and protection middleware |
| Container | Docker Compose | Local PostgreSQL and Redis dependencies |

## Domain model

Leamout separates these concepts:

- **Products**: creator-owned things being sold.
- **Prices**: one-time, recurring, or usage price definitions. MVP renewal uses recurring prices.
- **Customers**: user-scoped customer records with phone numbers.
- **Subscriptions**: active customer subscriptions tied to a recurring price.
- **Dunning attempts**: system-managed renewal reminder attempts for subscriptions nearing expiry.
- **Dunning tokens**: short-lived hashed recovery tokens used in SMS links.
- **Checkout sessions**: payment or renewal attempts created when a customer starts checkout.
- **Communication credits**: prepaid creator balance used for outbound SMS.
- **Personal access tokens**: user-scoped API tokens for local testing and future API access.

Checkout links and dunning links are intentionally separate:

- **Checkout links** are user-managed product/payment entry points.
- **Dunning links** are system-managed recovery links created for renewal reminders.

## Local setup

Start dependencies:

```powershell
docker compose up -d
```

Set local environment variables in `.env`:

```env
DATABASE_URL=postgres://leamout:leamout@localhost:5432/leamout?sslmode=disable
REDIS_URL=redis://localhost:6379/0
APP_ENV=development
PORT=8080

API_BASE_URL=http://localhost:8080
FRONTEND_BASE_URL=http://localhost:3000
SHORT_BASE_URL=http://localhost:3000

TRUSTED_PROXIES=127.0.0.1,::1
GEOIP_DATABASE_PATH=./assets/GeoLite2-City.mmdb
```

Run app and River migrations:

```powershell
go run ./cmd/migrate up
```

`cmd/migrate up` should run both Leamout migrations and River's internal queue migrations, so workers can start without missing tables such as `river_queue` or `river_leader`.

Start the three processes in separate terminals:

```powershell
# Terminal 1
go run ./cmd/server

# Terminal 2
go run ./cmd/worker

# Terminal 3
go run ./cmd/scheduler
```

For local testing, the scheduler can temporarily use `platformcron.ScheduleMin` instead of the hourly schedule.

## Auth for local API testing

Login in the browser:

```text
http://localhost:8080/v1/auth/google
```

After login, create a PAT from the browser console:

```js
const res = await fetch("http://localhost:8080/v1/personal-access-tokens", {
  method: "POST",
  credentials: "include",
  headers: {
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "local-product-flow-test",
    metadata: { purpose: "testing real product flow" },
  }),
});

const data = await res.json();
console.log(data.raw_token);
```

Then use it in PowerShell:

```powershell
$PAT = "PASTE_RAW_TOKEN_HERE"
$API = "http://localhost:8080/v1"
$Headers = @{
  Authorization = "Bearer $PAT"
  "Content-Type" = "application/json"
}
```

Do not commit or share raw PATs. Revoke test PATs after use.

## Manual renewal flow test

### 1. Create product + recurring price

```powershell
$stamp = Get-Date -Format "yyyyMMddHHmmss"

$productBody = @{
  name = "Leamout Test Membership $stamp"
  description = "Monthly creator membership for dunning test"
  prices = @(
    @{
      nickname = "Monthly"
      type = "recurring"
      unit_amount = 5000
      currency = "GHS"
      interval = "month"
    }
  )
} | ConvertTo-Json -Depth 10

$product = Invoke-RestMethod -Method Post -Uri "$API/products" -Headers $Headers -Body $productBody
$PriceID = $product.prices[0].id
```

### 2. Create customer

Use a mock-routed number locally when testing SMS without a production provider.

```powershell
$rand = Get-Random -Minimum 1000000 -Maximum 9999999
$phone = "+234803$rand"

$customerBody = @{
  name = "Test Customer $stamp"
  email = "customer+$stamp@example.com"
  phone = $phone
  external_id = "cust_$stamp"
  address = @{
    city = "Accra"
    country = "GH"
  }
  metadata = @{
    test = "dunning-flow"
  }
} | ConvertTo-Json -Depth 10

$customer = Invoke-RestMethod -Method Post -Uri "$API/customers" -Headers $Headers -Body $customerBody
$CustomerID = $customer.id
```

### 3. Create subscription due within the scan window

```powershell
$periodStart = (Get-Date).ToUniversalTime().AddDays(-29).ToString("o")
$periodEnd = (Get-Date).ToUniversalTime().AddHours(24).ToString("o")

$subBody = @{
  customer_id = $CustomerID
  price_id = $PriceID
  status = "active"
  current_period_start = $periodStart
  current_period_end = $periodEnd
  metadata = @{
    test = "dunning-flow"
  }
} | ConvertTo-Json -Depth 10

$subscription = Invoke-RestMethod -Method Post -Uri "$API/subscriptions" -Headers $Headers -Body $subBody
$SubscriptionID = $subscription.id
```

### 4. Top up communication credits

```powershell
$topupBody = @{
  amount = 1000
  reference = "local_topup_$stamp"
  description = "Local dunning flow test topup"
  metadata = @{
    test = "dunning-flow"
  }
} | ConvertTo-Json -Depth 10

$balance = Invoke-RestMethod -Method Post -Uri "$API/credits/topup" -Headers $Headers -Body $topupBody
```

### 5. Wait for scheduler + worker

The scheduler should log something like:

```text
dunning scanner completed scanned=1 enqueued=1 skipped=0
```

The worker should log a mock SMS:

```text
[MOCK SMS] to=+234... message="Your Leamout subscription expires soon. Renew here: http://localhost:3000/r/<token>"
```

Copy the token after `/r/`.

### 6. Open recovery link through backend

Until the Next.js `/r/:token` page or rewrite exists, call the backend directly:

```powershell
$DunningToken = "PASTE_TOKEN_HERE"
curl.exe -i "http://localhost:8080/v1/dunning/$DunningToken"
```

Expected response before the token is used:

```text
HTTP/1.1 302 Found
Location: http://localhost:3000/checkout/<client_secret>
```

Copy the client secret from the `Location` header.

### 7. Fetch checkout session

```powershell
$ClientSecret = "PASTE_CLIENT_SECRET_HERE"
$checkout = Invoke-RestMethod -Method Get -Uri "$API/checkout/$ClientSecret"
$checkout
```

Expected fields:

```text
mode   = renewal
source = dunning
status = open
amount = 5000
currency = GHS
```

### 8. Confirm checkout

```powershell
$confirm = Invoke-RestMethod -Method Post -Uri "$API/checkout/$ClientSecret/confirm"
$confirm
```

### 9. Verify renewal

```powershell
$renewedSub = Invoke-RestMethod -Method Get -Uri "$API/subscriptions/$SubscriptionID" -Headers $Headers
$renewedSub
```

For a monthly price, `current_period_end` should move forward by one month.

### 10. Verify dunning state

```powershell
$dunningEvents = Invoke-RestMethod -Method Get -Uri "$API/dunning-events" -Headers $Headers
$dunningEvents
```

Expected state:

```text
status = paid
sent_at set
clicked_at set
paid_at set
```

Opening the same dunning token after payment should return:

```text
404 recovery link not found or expired
```

That proves the token cannot be reused.

### 11. Verify credit ledger

```powershell
$ledger = Invoke-RestMethod -Method Get -Uri "$API/credits/ledger" -Headers $Headers
$ledger | Format-Table type, amount, balance_after, provider, destination, description, created_at
```

Expected local mock route result:

```text
topup   1000   1000
debit    -15    985   mock   +234   Dunning SMS
```

## Important routes

### Auth

```text
GET  /v1/auth/google
GET  /v1/auth/google/callback
GET  /v1/auth/github
GET  /v1/auth/github/callback
POST /v1/auth/logout
```

### PATs

```text
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

### Credits

```text
GET  /v1/credits
GET  /v1/credits/ledger
POST /v1/credits/topup
```

### Dunning and checkout

```text
GET  /v1/dunning/:token
GET  /v1/dunning-events
GET  /v1/dunning-events/:id

GET  /v1/checkout/:clientSecret
POST /v1/checkout/:clientSecret/confirm
```

## Environment design

Use separate base URLs for separate responsibilities:

```env
API_BASE_URL=https://api.leamout.com
FRONTEND_BASE_URL=https://leamout.com
SHORT_BASE_URL=https://lmt.com
```

Local development:

```env
API_BASE_URL=http://localhost:8080
FRONTEND_BASE_URL=http://localhost:3000
SHORT_BASE_URL=http://localhost:3000
```

Usage:

- `API_BASE_URL`: OAuth callbacks and backend-owned URLs.
- `FRONTEND_BASE_URL`: checkout page redirects, for example `/checkout/<client_secret>`.
- `SHORT_BASE_URL`: SMS links, for example `/r/<token>`.

Production target:

```text
SMS:      https://lmt.com/r/<token>
Proxy:    https://api.leamout.com/v1/dunning/<token>
Checkout: https://leamout.com/checkout/<client_secret>
API:      https://api.leamout.com/v1/checkout/<client_secret>
```

## Payment direction

Leamout is designed around mobile money as the primary collection rail.

- **PawaPay**: mobile money payment collection and checkout confirmation foundation.
- **Tola Mobile**: mobile money and local payment infrastructure foundation.
- **Local development**: checkout confirmation can be simulated while provider callbacks and production settlement behavior are being wired.

The MVP backend keeps checkout, subscription renewal, and dunning state separate so provider-specific confirmation logic can be added without changing the product/customer/subscription model.

## What is intentionally not built yet

- Next.js checkout UI.
- Production payment provider callback hardening.
- Production short-domain deployment.
- Full creator dashboard.
- Full organization/team support.
- Advanced retry/backoff policies.
- Usage billing prototype.
- WhatsApp Business API.

## Next milestone

The next milestone is the frontend MVP path:

```text
Next.js /r/:token or short-domain rewrite
  ↓
Next.js /checkout/:client_secret
  ↓
GET /v1/checkout/:clientSecret
  ↓
show customer/product/subscription details
  ↓
confirm checkout
  ↓
show renewal success
```
