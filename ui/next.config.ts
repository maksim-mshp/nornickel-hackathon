import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

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

export default withNextIntl(nextConfig);
