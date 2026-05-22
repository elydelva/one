# Stripe

Stripe REST API via `one stripe <action>`. Auth: PAT (secret key `sk_live_...` or `sk_test_...`).

## Body encoding (READ FIRST)

Stripe accepts **only `application/x-www-form-urlencoded`**, never JSON. Every write
action takes a single `body_raw` input: a pre-encoded form string you build yourself.

```
body_raw: "amount=2000&currency=usd&customer=cus_123&metadata[order_id]=42"
```

Nested params use bracket notation (`metadata[key]=val`, `card[number]=...`).
Arrays use `items[0][price]=...&items[0][quantity]=1`. URL-encode values that
contain `&`, `=`, or spaces.

Path and query inputs are declared normally. Idempotency: pass `Idempotency-Key`
as a header input on write calls when retrying.

## Customers
- `customers.list`, `customers.read`, `customers.create`, `customers.update`, `customers.delete`
- `customers.payment_methods.list`

## Payment Intents
- `payment_intents.list`, `payment_intents.read`, `payment_intents.create`
- `payment_intents.update`, `payment_intents.confirm`, `payment_intents.capture`, `payment_intents.cancel`

## Setup Intents
- `setup_intents.create`, `setup_intents.confirm`

## Payment Methods
- `payment_methods.list`, `payment_methods.attach`, `payment_methods.detach`

## Charges + Refunds
- `charges.list`, `charges.read`
- `refunds.create`, `refunds.list`, `refunds.read`

## Invoices
- `invoices.list`, `invoices.read`, `invoices.create`
- `invoices.finalize`, `invoices.pay`, `invoices.send`, `invoices.void`

## Subscriptions
- `subscriptions.list`, `subscriptions.read`, `subscriptions.create`, `subscriptions.update`, `subscriptions.cancel`

## Products + Prices
- `products.list`, `products.create`, `products.update`
- `prices.list`, `prices.create`

## Checkout + Billing Portal
- `checkout.sessions.create`, `checkout.sessions.read`, `checkout.sessions.expire`
- `billing_portal.sessions.create`

## Coupons
- `coupons.list`, `coupons.read`, `coupons.create`, `coupons.update`, `coupons.delete`

## Disputes
- `disputes.list`, `disputes.read`, `disputes.update`

## Payouts + Balance
- `payouts.list`, `payouts.create`
- `balance.read`

## Webhooks
- `webhook_endpoints.list`, `webhook_endpoints.create`, `webhook_endpoints.delete`

## Tax + Connect
- `tax_rates.list`, `tax_rates.create`, `tax_rates.update`
- `accounts.list`, `accounts.read`, `accounts.create`
- `transfers.create`, `transfers.list`
- `application_fees.list`

## Pagination

Stripe uses cursor pagination: `?limit=N&starting_after=<id>`. Response shape:
`{data:[...], has_more, url}`. List actions return raw payloads — paginate by
re-calling with the last `data[].id` in `starting_after`.
