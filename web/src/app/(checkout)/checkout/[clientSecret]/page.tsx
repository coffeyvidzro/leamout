"use client";

import type { FormEvent } from "react";
import { use, useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { apiFetch } from "@/lib/api";
import type { CheckoutSession } from "@/types/checkout";

type CheckoutPageProps = {
  params: Promise<{
    clientSecret: string;
  }>;
};

type MobileMoneyOperator = "mtn" | "telecel" | "at";

type PayResponse = {
  checkout_session_id: string;
  external_ref: string;
  provider_id: string;
  provider_reference?: string;
  status: string;
  next_action_type: string;
  next_action_url?: string;
  customer_message?: string;
};

const operatorLabels: Record<MobileMoneyOperator, string> = {
  mtn: "MTN MoMo",
  telecel: "Telecel Cash",
  at: "AT Money",
};

function formatMoney(amount?: number, currency = "GHS") {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format((amount ?? 0) / 100);
}

function readableStatus(status: string) {
  return status.replaceAll("_", " ");
}

export default function CheckoutPage({ params }: CheckoutPageProps) {
  const { clientSecret } = use(params);

  const [session, setSession] = useState<CheckoutSession | null>(null);
  const [phone, setPhone] = useState("");
  const [operator, setOperator] = useState<MobileMoneyOperator>("mtn");
  const [payment, setPayment] = useState<PayResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [paying, setPaying] = useState(false);
  const [awaitingApproval, setAwaitingApproval] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadCheckout = useCallback(async () => {
    try {
      setError(null);
      const data = await apiFetch<CheckoutSession>(`/checkout/${clientSecret}`);
      setSession(data);

      if (data.status === "completed") {
        setAwaitingApproval(false);
      }
    } catch {
      setError("This checkout link is invalid or has expired.");
    } finally {
      setLoading(false);
    }
  }, [clientSecret]);

  useEffect(() => {
    loadCheckout();
  }, [loadCheckout]);

  useEffect(() => {
    if (!awaitingApproval || session?.status !== "open") {
      return;
    }

    const timer = window.setInterval(() => {
      loadCheckout();
    }, 4000);

    return () => window.clearInterval(timer);
  }, [awaitingApproval, loadCheckout, session?.status]);

  async function sendPaymentPrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const normalizedPhone = phone.trim();
    if (normalizedPhone.length < 9) {
      setError("Enter a valid Mobile Money number.");
      return;
    }

    try {
      setPaying(true);
      setError(null);
      setPayment(null);

      const data = await apiFetch<PayResponse>(`/checkout/${clientSecret}/pay`, {
        method: "POST",
        body: JSON.stringify({
          country: "GH",
          phone: normalizedPhone,
          operator,
          preferred_provider: "moolre",
        }),
      });

      setPayment(data);
      setAwaitingApproval(true);
      await loadCheckout();
    } catch {
      setError(
        "We could not send the payment prompt. Check the number and try again.",
      );
    } finally {
      setPaying(false);
    }
  }

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
  const title = isCompleted
    ? "Payment successful"
    : session.mode === "renewal"
      ? "Complete your renewal"
      : "Complete payment";
  const description = isCompleted
    ? "Your payment has been received successfully."
    : "Pay securely with Mobile Money.";
  const label = session.label?.trim() || "Leamout payment";

  return (
    <main className="flex min-h-screen items-center justify-center bg-muted/30 p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="mb-4">
            <div className="font-semibold text-lg tracking-tight">Leamout</div>
            <p className="text-muted-foreground text-xs">Secure checkout</p>
          </div>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>

        <CardContent className="space-y-5">
          <div className="space-y-3 text-sm">
            <div className="flex items-center justify-between gap-4">
              <span className="text-muted-foreground">Amount</span>
              <span className="font-semibold text-lg">
                {formatMoney(session.amount, session.currency)}
              </span>
            </div>

            <div className="flex items-center justify-between gap-4">
              <span className="text-muted-foreground">For</span>
              <span className="max-w-56 truncate text-right font-medium">
                {label}
              </span>
            </div>

            <div className="flex items-center justify-between gap-4">
              <span className="text-muted-foreground">Status</span>
              <span className="font-medium capitalize">
                {readableStatus(session.status)}
              </span>
            </div>
          </div>

          <Separator />

          {isCompleted ? (
            <div className="space-y-4">
              <p className="text-muted-foreground text-sm">
                Your payment is complete. You can close this page.
              </p>
              <Button className="w-full" disabled>
                Paid
              </Button>
            </div>
          ) : null}

          {!isCompleted && !isOpen ? (
            <div className="space-y-4">
              <p className="text-muted-foreground text-sm">
                This checkout is no longer available.
              </p>
              <Button className="w-full" disabled>
                Unavailable
              </Button>
            </div>
          ) : null}

          {isOpen ? (
            <form className="space-y-4" onSubmit={sendPaymentPrompt}>
              <div className="space-y-2">
                <Label htmlFor="phone">Mobile Money number</Label>
                <Input
                  autoComplete="tel"
                  id="phone"
                  inputMode="tel"
                  onChange={(event) => setPhone(event.target.value)}
                  placeholder="024 123 4567"
                  required
                  value={phone}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="operator">Network</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={paying || awaitingApproval}
                  id="operator"
                  onChange={(event) =>
                    setOperator(event.target.value as MobileMoneyOperator)
                  }
                  value={operator}
                >
                  {Object.entries(operatorLabels).map(([value, name]) => (
                    <option key={value} value={value}>
                      {name}
                    </option>
                  ))}
                </select>
              </div>

              {payment ? (
                <div className="rounded-md border bg-muted/50 p-3 text-sm">
                  <p className="font-medium">Payment request sent</p>
                  <p className="mt-1 text-muted-foreground">
                    Check your phone and approve the Mobile Money prompt.
                  </p>
                  <p className="mt-2 text-muted-foreground capitalize">
                    Provider status: {readableStatus(payment.status)}
                  </p>
                </div>
              ) : null}

              {awaitingApproval ? (
                <p className="text-muted-foreground text-sm">
                  Waiting for confirmation. This page will update automatically
                  after the payment is captured.
                </p>
              ) : null}

              {error ? <p className="text-destructive text-sm">{error}</p> : null}

              <Button
                className="w-full"
                disabled={paying || awaitingApproval}
                type="submit"
              >
                {paying
                  ? "Sending prompt..."
                  : awaitingApproval
                    ? "Waiting for approval..."
                    : "Send payment prompt"}
              </Button>
            </form>
          ) : null}
        </CardContent>
      </Card>
    </main>
  );
}
