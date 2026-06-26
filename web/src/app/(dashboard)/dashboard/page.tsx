"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiFetch } from "@/lib/api";
import type { CreditBalance } from "@/types/credits";
import type { Customer } from "@/types/customer";
import type { DunningAttempt } from "@/types/dunning";
import type { Product } from "@/types/product";
import type { Subscription } from "@/types/subscription";

function formatMoney(amount: number, currency = "GHS") {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency,
  }).format(amount / 100);
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleString();
}

export default function DashboardPage() {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [balance, setBalance] = useState<CreditBalance | null>(null);
  const [dunningAttempts, setDunningAttempts] = useState<DunningAttempt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const activeSubscriptionCount = useMemo(
    () =>
      subscriptions.filter((subscription) => subscription.status === "active")
        .length,
    [subscriptions],
  );

  const recentDunningAttempts = dunningAttempts.slice(0, 5);

  useEffect(() => {
    async function loadDashboard() {
      try {
        setError(null);

        const [
          customerData,
          productData,
          subscriptionData,
          balanceData,
          dunningData,
        ] = await Promise.all([
          apiFetch<Customer[]>("/customers"),
          apiFetch<Product[]>("/products"),
          apiFetch<Subscription[]>("/subscriptions"),
          apiFetch<CreditBalance>("/credits"),
          apiFetch<DunningAttempt[]>("/dunning-events"),
        ]);

        setCustomers(customerData);
        setProducts(productData);
        setSubscriptions(subscriptionData);
        setBalance(balanceData);
        setDunningAttempts(dunningData);
      } catch {
        setError("Could not load dashboard overview.");
      } finally {
        setLoading(false);
      }
    }

    loadDashboard();
  }, []);

  const stats = [
    {
      label: "Customers",
      value: customers.length.toString(),
      href: "/customers",
    },
    {
      label: "Products",
      value: products.length.toString(),
      href: "/products",
    },
    {
      label: "Active subscriptions",
      value: activeSubscriptionCount.toString(),
      href: "/subscriptions",
    },
    {
      label: "Credit balance",
      value: balance ? formatMoney(balance.balance, balance.currency) : "—",
      href: "/credits",
    },
  ];

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Dashboard"
          description="A simple overview of your Leamout account."
        />

        <div className="flex gap-2">
          <Button asChild variant="outline">
            <Link href="/customers/new">Add customer</Link>
          </Button>

          <Button asChild>
            <Link href="/subscriptions/new">Add subscription</Link>
          </Button>
        </div>
      </div>

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading dashboard...</p>
      ) : null}

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.label}>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {stat.label}
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="text-2xl font-semibold">{stat.value}</div>
              <Button asChild className="mt-4 h-auto p-0" variant="link">
                <Link href={stat.href}>View</Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Recent dunning events</CardTitle>
        </CardHeader>

        <CardContent>
          {!loading && !error && recentDunningAttempts.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No dunning events yet.
            </p>
          ) : null}

          {!loading && !error && recentDunningAttempts.length > 0 ? (
            <div className="space-y-4">
              {recentDunningAttempts.map((attempt) => (
                <div
                  className="flex flex-col justify-between gap-1 border-b pb-4 last:border-b-0 last:pb-0 md:flex-row md:items-center"
                  key={attempt.id}
                >
                  <div>
                    <div className="text-sm font-medium capitalize">
                      {attempt.status} · {attempt.reason.replace("_", " ")}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Subscription {attempt.subscription_id}
                    </div>
                  </div>

                  <div className="text-xs text-muted-foreground">
                    {formatDateTime(attempt.updated_at)}
                  </div>
                </div>
              ))}
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
