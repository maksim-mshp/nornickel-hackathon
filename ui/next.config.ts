import type { NextConfig } from "next";

const GATEWAY_ORIGIN = "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
  async rewrites() {
    if (process.env.NODE_ENV !== "development") {
      return [];
    }
    return [
      {
        source: "/v1/:path*",
        destination: `${GATEWAY_ORIGIN}/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
