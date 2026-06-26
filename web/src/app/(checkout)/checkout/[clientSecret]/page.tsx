"use client";

import { use, useEffect, useState } from "react";
import type { CheckoutSession } from "@/types/checkout";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";

type CheckoutPageProps = {
  params: Promise<{
    clientSecret: string;
  }>;
};

function formatMoney(amount?: number, currency = "GHS") {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format((amount ?? 0) / 100);
}

export default function CheckoutPage({ params }: CheckoutPageProps) {
  const { clientSecret } = use(params);

  const [session, setSession] = useState<CheckoutSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function loadCheckout() {
    try {
      setError(null);
      const data = await apiFetch<CheckoutSession>(`/checkout/${clientSecret}`);
      setSession(data);
    } catch {
      setError("This checkout link is invalid or has expired.");
    } finally {
      setLoading(false);
    }
  }

  async function confirmPayment() {
    try {
      setConfirming(true);
      setError(null);

      const data = await apiFetch<{
        message: string;
        session: CheckoutSession;
      }>(`/checkout/${clientSecret}/confirm`, {
        method: "POST",
      });

      setSession(data.session);
    } catch {
      setError("We could not confirm this payment. Please try again.");
    } finally {
      setConfirming(false);
    }
  }

  useEffect(() => {
    loadCheckout();
  }, [clientSecret]);

  if (loading) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6">
        <p className="text-muted-foreground text-sm">Loading checkout...</p>
      </main>
    );
  }

  if (error && !session) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle>Checkout unavailable</CardTitle>
            <CardDescription>{error}</CardDescription>
          </CardHeader>
        </Card>
      </main>
    );
  }

  if (!session) {
    return null;
  }

  const isCompleted = session.status === "completed";
  const isOpen = session.status === "open";

  return (
    <main className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>
            {isCompleted ? "Payment confirmed" : "Complete your renewal"}
          </CardTitle>
          <CardDescription>
            {isCompleted
              ? "Your subscription has been renewed successfully."
              : "Review your subscription renewal and continue."}
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-5">
          <div className="space-y-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Amount</span>
              <span className="font-medium">
                {formatMoney(session.amount, session.currency)}
              </span>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Status</span>
              <span className="font-medium capitalize">{session.status}</span>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Mode</span>
              <span className="font-medium capitalize">{session.mode}</span>
            </div>
          </div>

          <Separator />

          {error ? <p className="text-destructive text-sm">{error}</p> : null}

          {isOpen ? (
            <Button
              className="w-full"
              disabled={confirming}
              onClick={confirmPayment}
            >
              {confirming ? "Confirming..." : "Confirm payment"}
            </Button>
          ) : (
            <Button className="w-full" disabled>
              {isCompleted ? "Renewed" : "Unavailable"}
            </Button>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
