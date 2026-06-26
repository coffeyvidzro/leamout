"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiFetch } from "@/lib/api";
import type { Customer } from "@/types/customer";
import type { DunningAttempt } from "@/types/dunning";
import type { Product } from "@/types/product";
import type { Subscription } from "@/types/subscription";

type SubscriptionDetailProps = {
  subscriptionId: string;
};

function formatDate(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleDateString();
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleString();
}

function getStatusClassName(status: Subscription["status"]) {
  if (status === "active") {
    return "text-emerald-600";
  }

  if (status === "past_due" || status === "incomplete") {
    return "text-amber-600";
  }

  return "text-muted-foreground";
}

export function SubscriptionDetail({ subscriptionId }: SubscriptionDetailProps) {
  const [subscription, setSubscription] = useState<Subscription | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [dunningAttempts, setDunningAttempts] = useState<DunningAttempt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const customer = useMemo(
    () =>
      customers.find((item) => item.id === subscription?.customer_id) ?? null,
    [customers, subscription?.customer_id],
  );

  const productPrice = useMemo(() => {
    for (const product of products) {
      const price = product.prices.find(
        (item) => item.id === subscription?.price_id,
      );

      if (price) {
        return { product, price };
      }
    }

    return null;
  }, [products, subscription?.price_id]);

  const relatedDunningAttempts = useMemo(
    () =>
      dunningAttempts.filter(
        (attempt) => attempt.subscription_id === subscription?.id,
      ),
    [dunningAttempts, subscription?.id],
  );

  useEffect(() => {
    async function loadSubscription() {
      try {
        setError(null);

        const [subscriptionData, customerData, productData, dunningData] =
          await Promise.all([
            apiFetch<Subscription>(`/subscriptions/${subscriptionId}`),
            apiFetch<Customer[]>("/customers"),
            apiFetch<Product[]>("/products"),
            apiFetch<DunningAttempt[]>("/dunning-events"),
          ]);

        setSubscription(subscriptionData);
        setCustomers(customerData);
        setProducts(productData);
        setDunningAttempts(dunningData);
      } catch {
        setError("Could not load subscription details.");
      } finally {
        setLoading(false);
      }
    }

    loadSubscription();
  }, [subscriptionId]);

  if (loading) {
    return (
      <p className="text-sm text-muted-foreground">Loading subscription...</p>
    );
  }

  if (error || !subscription) {
    return (
      <div>
        <PageHeader
          title="Subscription not available"
          description="We could not load this subscription."
        />
        <p className="text-sm text-destructive">{error}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Subscription"
          description="Subscription period, customer, price, and recovery activity."
        />

        <Button asChild variant="outline">
          <Link href="/subscriptions">Back to subscriptions</Link>
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div
              className={`text-2xl font-semibold capitalize ${getStatusClassName(
                subscription.status,
              )}`}
            >
              {subscription.status.replace("_", " ")}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Period start</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">
              {formatDate(subscription.current_period_start)}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Period end</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">
              {formatDate(subscription.current_period_end)}
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Customer</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div>
              <div className="text-muted-foreground">Name</div>
              {customer ? (
                <Button asChild className="h-auto p-0" variant="link">
                  <Link href={`/customers/${customer.id}`}>{customer.name}</Link>
                </Button>
              ) : (
                <div className="font-mono text-xs">
                  {subscription.customer_id ?? "—"}
                </div>
              )}
            </div>
            <div>
              <div className="text-muted-foreground">Phone</div>
              <div>{customer?.phone ?? "—"}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Email</div>
              <div>{customer?.email ?? "—"}</div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Product price</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div>
              <div className="text-muted-foreground">Product</div>
              {productPrice ? (
                <Button asChild className="h-auto p-0" variant="link">
                  <Link href={`/products/${productPrice.product.id}`}>
                    {productPrice.product.name}
                  </Link>
                </Button>
              ) : (
                <div>—</div>
              )}
            </div>
            <div>
              <div className="text-muted-foreground">Price</div>
              <div>{productPrice?.price.nickname ?? subscription.price_id}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Renewal</div>
              <div>
                {subscription.cancel_at_period_end ? "Canceling" : "Renews"}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Dunning events</CardTitle>
        </CardHeader>
        <CardContent>
          {relatedDunningAttempts.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No dunning events for this subscription yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Reason</TableHead>
                  <TableHead>Period end</TableHead>
                  <TableHead>Sent</TableHead>
                  <TableHead>Clicked</TableHead>
                  <TableHead>Paid</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {relatedDunningAttempts.map((attempt) => (
                  <TableRow key={attempt.id}>
                    <TableCell className="capitalize">{attempt.status}</TableCell>
                    <TableCell className="capitalize">
                      {attempt.reason.replace("_", " ")}
                    </TableCell>
                    <TableCell>{formatDate(attempt.period_end)}</TableCell>
                    <TableCell>{formatDateTime(attempt.sent_at)}</TableCell>
                    <TableCell>{formatDateTime(attempt.clicked_at)}</TableCell>
                    <TableCell>{formatDateTime(attempt.paid_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
