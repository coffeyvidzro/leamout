import { Geist, Geist_Mono, Mona_Sans } from "next/font/google";

export const fontSans = Geist({
  variable: "--font-sans",
  subsets: ["latin"],
});

export const fontMono = Geist_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
});

export const fontHeading = Mona_Sans({
  variable: "--font-heading",
  subsets: ["latin"],
});
