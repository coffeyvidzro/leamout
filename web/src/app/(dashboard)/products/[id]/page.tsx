import { ProductDetail } from "@/components/products/product-detail";
import { serverApiFetch } from "@/lib/server-api";
import type { Product } from "@/types/product";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const product = await serverApiFetch<Product>(`/products/${id}`);

  return <ProductDetail product={product} />;
}
