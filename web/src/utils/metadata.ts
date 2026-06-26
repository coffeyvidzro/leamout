import type { Metadata } from "next";

export function constructMetadata({
  title,
  description = "Leamout is a billing and monetization platform for Africa.",
  image = "/assets/leamout.png",
  url,
  noIndex = false,
}: {
  title?: string;
  description?: string;
  image?: string;
  url?: string;
  noIndex?: boolean;
} = {}): Metadata {
  const pageTitle = title
    ? `${title} | Leamout`
    : "Leamout | Revenue platform for Africa";

  return {
    title: pageTitle,
    description,
    metadataBase: new URL("https://leamout.com"),

    alternates: {
      canonical: url,
    },

    openGraph: {
      title: pageTitle,
      description,
      url,
      siteName: "Leamout",
      locale: "en_GH",
      type: "website",
      images: [
        {
          url: image,
          width: 1200,
          height: 630,
        },
      ],
    },

    twitter: {
      card: "summary_large_image",
      title: pageTitle,
      description,
      images: [image],
      creator: "@leamout",
    },

    robots: {
      index: !noIndex,
      follow: !noIndex,
      googleBot: {
        index: !noIndex,
        follow: !noIndex,
        "max-video-preview": -1,
        "max-image-preview": "large",
        "max-snippet": -1,
      },
    },
  };
}
