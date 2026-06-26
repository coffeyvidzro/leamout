"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiFetch } from "@/lib/api";
import type { Customer } from "@/types/customer";
import type { Price, Product } from "@/types/product";
import type { Subscription } from "@/types/subscription";

function defaultPeriodEnd() {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() + 30);
  return date.toISOString().slice(0, 10);
}

function formatMoney(amount: number, currency: string) {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency,
  }).format(amount / 100);
}

function formatPriceLabel(product: Product, price: Price) {
  const amount = formatMoney(price.unit_amount, price.currency);
  const interval =
    price.type === "recurring" && price.interval ? ` / ${price.interval}` : "";

  return `${product.name} — ${price.nickname} (${amount}${interval})`;
}

export default function NewSubscriptionPage() {
  const router = useRouter();
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [customerId, setCustomerId] = useState("");
  const [priceId, setPriceId] = useState("");
  const [periodEnd, setPeriodEnd] = useState(defaultPeriodEnd());
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const priceOptions = useMemo(
    () =>
      products.flatMap((product) =>
        product.prices.map((price) => ({ product, price })),
      ),
    [products],
  );

  useEffect(() => {
    async function loadFormData() {
      try {
        setError(null);

        const [customerData, productData] = await Promise.all([
          apiFetch<Customer[]>("/customers"),
          apiFetch<Product[]>("/products"),
        ]);

        setCustomers(customerData);
        setProducts(productData);
        setCustomerId(customerData[0]?.id ?? "");
        setPriceId(productData.flatMap((product) => product.prices)[0]?.id ?? "");
      } catch {
        setError("Could not load customers and products.");
      } finally {
        setLoading(false);
      }
    }

    loadFormData();
  }, []);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!customerId || !priceId || !periodEnd) {
      setError("Choose a customer, price, and period end date.");
      return;
    }

    try {
      setSubmitting(true);
      setError(null);

      await apiFetch<Subscription>("/subscriptions", {
        method: "POST",
        body: JSON.stringify({
          customer_id: customerId,
          price_id: priceId,
          current_period_end: new Date(
            `${periodEnd}T23:59:59.000Z`,
          ).toISOString(),
          metadata: {},
        }),
      });

      router.push("/subscriptions");
      router.refresh();
    } catch {
      setError("Could not create subscription. Check the details and try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Add subscription"
        description="Attach a customer to a product price and set the first renewal date."
      />

      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle className="text-base">Subscription details</CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">
              Loading customers and prices...
            </p>
          ) : null}

          {!loading && customers.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Create a customer before creating a subscription.
            </p>
          ) : null}

          {!loading && priceOptions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Create a product with a price before creating a subscription.
            </p>
          ) : null}

          {!loading && customers.length > 0 && priceOptions.length > 0 ? (
            <form className="space-y-5" onSubmit={handleSubmit}>
              <div className="space-y-2">
                <Label htmlFor="customer">Customer</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
                  id="customer"
                  onChange={(event) => setCustomerId(event.target.value)}
                  required
                  value={customerId}
                >
                  {customers.map((customer) => (
                    <option key={customer.id} value={customer.id}>
                      {customer.name} — {customer.phone ?? "no phone"}
                    </option>
                  ))}
                </select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="price">Product price</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
                  id="price"
                  onChange={(event) => setPriceId(event.target.value)}
                  required
                  value={priceId}
                >
                  {priceOptions.map(({ product, price }) => (
                    <option key={price.id} value={price.id}>
                      {formatPriceLabel(product, price)}
                    </option>
                  ))}
                </select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="period-end">Current period end</Label>
                <Input
                  id="period-end"
                  onChange={(event) => setPeriodEnd(event.target.value)}
                  required
                  type="date"
                  value={periodEnd}
                />
              </div>

              {error ? <p className="text-sm text-destructive">{error}</p> : null}

              <div className="flex items-center gap-3">
                <Button disabled={submitting} type="submit">
                  {submitting ? "Creating..." : "Create subscription"}
                </Button>

                <Button asChild type="button" variant="outline">
                  <Link href="/subscriptions">Cancel</Link>
                </Button>
              </div>
            </form>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
