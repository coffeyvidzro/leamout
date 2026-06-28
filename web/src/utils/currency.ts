// utils/currency.ts

/**
 * Formats a monetary amount from minor units.
 *
 * Example:
 * formatMoney(10000) => "GHS 100.00"
 * formatMoney(257) => "GHS 2.57"
 */
export function formatMoney(amount?: number, currency = "GHS") {
  return new Intl.NumberFormat("en-GH", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format((amount ?? 0) / 100);
}

/**
 * Helper for adding money values stored in minor units safely.
 */
export function addMoney(...amounts: Array<number | undefined>): number {
  return amounts.reduce<number>((total, amount) => total + (amount ?? 0), 0);
}
