import type { CountryCode } from "./countries";

export type NetworkCode = "MTN" | "MOOV" | "ORANGE" | "AIRTELTIGO" | "TELECEL";

export type NetworkOption = {
  code: NetworkCode;
  name: string;
  country: CountryCode;
};

export const NETWORKS: NetworkOption[] = [
  // Benin
  {
    country: "BEN",
    code: "MOOV",
    name: "Moov Money",
  },
  {
    country: "BEN",
    code: "MTN",
    name: "MTN Mobile Money",
  },

  // Burkina Faso
  {
    country: "BFA",
    code: "MOOV",
    name: "Moov Money",
  },
  {
    country: "BFA",
    code: "ORANGE",
    name: "Orange Money",
  },

  // Cameroon
  {
    country: "CMR",
    code: "MTN",
    name: "MTN Mobile Money",
  },
  {
    country: "CMR",
    code: "ORANGE",
    name: "Orange Money",
  },

  // Ghana
  {
    country: "GHA",
    code: "MTN",
    name: "MTN Mobile Money",
  },
  {
    country: "GHA",
    code: "TELECEL",
    name: "Telecel Cash",
  },
  {
    country: "GHA",
    code: "AIRTELTIGO",
    name: "AirtelTigo Money",
  },

  // Ivory Coast
  {
    country: "CIV",
    code: "MTN",
    name: "MTN Mobile Money",
  },
  {
    country: "CIV",
    code: "ORANGE",
    name: "Orange Money",
  },

  // Sierra Leone
  {
    country: "SLE",
    code: "ORANGE",
    name: "Orange Money",
  },
];

export function getNetworksByCountry(country?: CountryCode | string) {
  if (!country) return [];

  return NETWORKS.filter((network) => network.country === country);
}

export function getNetworkByCode(
  country?: CountryCode | string,
  code?: string,
) {
  if (!country || !code) return undefined;

  return NETWORKS.find(
    (network) => network.country === country && network.code === code,
  );
}
