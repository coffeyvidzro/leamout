"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { useState } from "react";
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

export default function NewCustomerPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [externalId, setExternalId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    try {
      setSubmitting(true);
      setError(null);

      await apiFetch<Customer>("/customers", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          email: email.trim() || undefined,
          phone: phone.trim(),
          external_id: externalId.trim() || undefined,
          address: {},
          metadata: {},
        }),
      });

      router.push("/customers");
      router.refresh();
    } catch {
      setError("Could not create customer. Check the details and try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Add customer"
        description="Create a customer that can be attached to subscriptions and renewal reminders."
      />

      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle className="text-base">Customer details</CardTitle>
        </CardHeader>

        <CardContent>
          <form className="space-y-5" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                maxLength={160}
                onChange={(event) => setName(event.target.value)}
                placeholder="Kwame Mensah"
                required
                value={name}
              />
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="customer@example.com"
                  type="email"
                  value={email}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="phone">Phone</Label>
                <Input
                  id="phone"
                  maxLength={40}
                  minLength={3}
                  onChange={(event) => setPhone(event.target.value)}
                  placeholder="+233501234567"
                  required
                  value={phone}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="external-id">External ID</Label>
              <Input
                id="external-id"
                maxLength={160}
                onChange={(event) => setExternalId(event.target.value)}
                placeholder="Optional ID from your own system"
                value={externalId}
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}

            <div className="flex items-center gap-3">
              <Button disabled={submitting} type="submit">
                {submitting ? "Creating..." : "Create customer"}
              </Button>

              <Button asChild type="button" variant="outline">
                <Link href="/customers">Cancel</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
