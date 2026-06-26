import { ProductDetail } from "@/components/dashboard/product-detail";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const routeParams = await params;

  return <ProductDetail productId={routeParams.id} />;
}
