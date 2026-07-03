import type { NextConfig } from "next";

const gatewayOrigin =
  process.env.NODE_ENV === "development"
    ? "http://localhost:8080"
    : "http://gateway:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: "/v1/:path*",
        destination: `${gatewayOrigin}/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
