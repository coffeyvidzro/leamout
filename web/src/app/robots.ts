import type { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  const baseUrl = "https://leamout.com";

  return {
    rules: {
      userAgent: "*",
      allow: ["/", "/features", "/pricing"],
      
      disallow: [
        "/login",
        "/register",
        "/checkout/",
        "/dashboard/",  
      ],
    },
    sitemap: `${baseUrl}/sitemap.xml`,
  };
}
