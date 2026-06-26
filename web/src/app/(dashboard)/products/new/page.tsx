import { PageHeader } from "@/components/dashboard/page-header";
import { ProductCreateForm } from "@/components/products/product-create-form";

export default function NewProductPage() {
  return (
    <div>
      <PageHeader
        title="Add product"
        description="Create a product with its first price for checkout and subscriptions."
      />

      <ProductCreateForm />
    </div>
  );
}
