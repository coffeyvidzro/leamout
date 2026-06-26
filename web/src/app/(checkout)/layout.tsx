import type { Metadata } from "next";

export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default async function CheckoutLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div translate="no">{children}</div>;
}
