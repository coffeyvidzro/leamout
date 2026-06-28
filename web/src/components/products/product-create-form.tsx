"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { apiFetch } from "@/lib/api";
import type { PriceInterval, PriceType, Product } from "@/types/product";

export function ProductCreateForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [priceNickname, setPriceNickname] = useState("Monthly access");
  const [amount, setAmount] = useState("50");
  const [currency, setCurrency] = useState("GHS");
  const [type, setType] = useState<PriceType>("recurring");
  const [interval, setInterval] = useState<PriceInterval>("month");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const parsedAmount = Number(amount);
    if (!Number.isFinite(parsedAmount) || parsedAmount <= 0) {
      setError("Enter a valid amount greater than zero.");
      return;
    }

    try {
      setSubmitting(true);
      setError(null);

      await apiFetch<Product>("/products", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim() || undefined,
          active: true,
          metadata: {},
          prices: [
            {
              nickname: priceNickname.trim(),
              type,
              unit_amount: Math.round(parsedAmount * 100),
              currency: currency.trim().toUpperCase(),
              interval: type === "recurring" ? interval : undefined,
              metadata: {},
            },
          ],
        }),
      });

      router.push("/products");
      router.refresh();
    } catch {
      setError("Could not create product. Check the details and try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card className="max-w-2xl">
      <CardHeader>
        <CardTitle className="text-base">Product details</CardTitle>
      </CardHeader>

      <CardContent>
        <form className="space-y-5" onSubmit={handleSubmit}>
          <div className="space-y-2">
            <Label htmlFor="name">Product name</Label>
            <Input
              id="name"
              maxLength={160}
              onChange={(event) => setName(event.target.value)}
              placeholder="Creator Membership"
              required
              value={name}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              maxLength={1000}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="What customers receive when they subscribe."
              value={description}
            />
          </div>

          <div className="grid gap-5 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="price-nickname">Price nickname</Label>
              <Input
                id="price-nickname"
                maxLength={160}
                onChange={(event) => setPriceNickname(event.target.value)}
                required
                value={priceNickname}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="amount">Amount</Label>
              <Input
                id="amount"
                min="0.01"
                onChange={(event) => setAmount(event.target.value)}
                required
                step="0.01"
                type="number"
                value={amount}
              />
            </div>
          </div>

          <div className="grid gap-5 md:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="currency">Currency</Label>
              <Input
                id="currency"
                maxLength={3}
                minLength={3}
                onChange={(event) => setCurrency(event.target.value)}
                required
                value={currency}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="type">Type</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
                id="type"
                onChange={(event) => setType(event.target.value as PriceType)}
                value={type}
              >
                <option value="recurring">Recurring</option>
                <option value="one_time">One-time</option>
              </select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="interval">Interval</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={type !== "recurring"}
                id="interval"
                onChange={(event) =>
                  setInterval(event.target.value as PriceInterval)
                }
                value={interval}
              >
                <option value="day">Day</option>
                <option value="week">Week</option>
                <option value="month">Month</option>
                <option value="year">Year</option>
              </select>
            </div>
          </div>

          {error ? <p className="text-sm text-destructive">{error}</p> : null}

          <div className="flex items-center gap-3">
            <Button disabled={submitting} type="submit">
              {submitting ? "Creating..." : "Create product"}
            </Button>

            <Button asChild type="button" variant="outline">
              <Link href="/products">Cancel</Link>
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
