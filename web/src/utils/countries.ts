export type CountryCode = "BEN" | "BFA" | "CMR" | "GHA" | "CIV" | "SLE";

export type CountryOption = {
  code: CountryCode;
  name: string;
  currency: string;
  callingCode: string;
};

export const COUNTRIES: CountryOption[] = [
  {
    code: "BEN",
    name: "Benin",
    currency: "XOF",
    callingCode: "229",
  },
  {
    code: "BFA",
    name: "Burkina Faso",
    currency: "XOF",
    callingCode: "226",
  },
  {
    code: "CMR",
    name: "Cameroon",
    currency: "XAF",
    callingCode: "237",
  },
  {
    code: "GHA",
    name: "Ghana",
    currency: "GHS",
    callingCode: "233",
  },
  {
    code: "CIV",
    name: "Ivory Coast",
    currency: "XOF",
    callingCode: "225",
  },
  {
    code: "SLE",
    name: "Sierra Leone",
    currency: "SLE",
    callingCode: "232",
  },
];

export function getCountryByCode(code?: string) {
  return COUNTRIES.find((country) => country.code === code);
}
