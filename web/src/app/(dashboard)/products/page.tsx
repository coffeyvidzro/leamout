import { ProductList } from "@/components/products/product-list";
import { serverApiFetch } from "@/lib/server-api";
import type { Product } from "@/types/product";

export default async function ProductsPage() {
  const products = await serverApiFetch<Product[]>("/products");

  return <ProductList products={products} />;
}
