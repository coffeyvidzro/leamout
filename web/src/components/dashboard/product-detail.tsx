"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiFetch } from "@/lib/api";
import type { Product } from "@/types/product";

type ProductDetailProps = {
  productId: string;
};

function formatMoney(amount: number, currency: string) {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency,
  }).format(amount / 100);
}

function formatDate(value?: string | null) {
  if (!value) {
    return "—";
  }

  return new Date(value).toLocaleDateString();
}

export function ProductDetail({ productId }: ProductDetailProps) {
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadProduct() {
      try {
        setError(null);
        const productData = await apiFetch<Product>(`/products/${productId}`);
        setProduct(productData);
      } catch {
        setError("Could not load product details.");
      } finally {
        setLoading(false);
      }
    }

    loadProduct();
  }, [productId]);

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading product...</p>;
  }

  if (error || !product) {
    return (
      <div>
        <PageHeader
          title="Product not available"
          description="We could not load this product."
        />
        <p className="text-sm text-destructive">{error}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title={product.name}
          description={product.description ?? "Product and pricing details."}
        />

        <Button asChild variant="outline">
          <Link href="/products">Back to products</Link>
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold capitalize">
              {product.active ? "Active" : "Inactive"}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Prices</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">
              {product.prices.length}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Created</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">
              {formatDate(product.created_at)}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Prices</CardTitle>
        </CardHeader>
        <CardContent>
          {product.prices.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No prices have been configured for this product.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nickname</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Interval</TableHead>
                  <TableHead>Lookup key</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {product.prices.map((price) => (
                  <TableRow key={price.id}>
                    <TableCell className="font-medium">
                      {price.nickname}
                    </TableCell>
                    <TableCell className="capitalize">
                      {price.type.replace("_", " ")}
                    </TableCell>
                    <TableCell>
                      {formatMoney(price.unit_amount, price.currency)}
                    </TableCell>
                    <TableCell>{price.interval ?? "—"}</TableCell>
                    <TableCell>{price.lookup_key ?? "—"}</TableCell>
                    <TableCell>{formatDate(price.created_at)}</TableCell>
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
