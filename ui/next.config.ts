import type { NextConfig } from "next";

const isDev = process.env.NODE_ENV === "development";
const gatewayOrigin = isDev ? "http://localhost:8080" : "http://gateway:8080";
const keycloakOrigin = isDev ? "http://localhost:8081" : "http://keycloak:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: "/v1/:path*",
        destination: `${gatewayOrigin}/v1/:path*`,
      },
      {
        source: "/kc/:path*",
        destination: `${keycloakOrigin}/:path*`,
      },
    ];
  },
};

export default nextConfig;
