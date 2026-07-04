import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

function readYaml(relativePath: string): Record<string, string> {
  for (const base of ["configs", "../configs"]) {
    const path = resolve(process.cwd(), base, relativePath);
    if (!existsSync(path)) continue;
    const result: Record<string, string> = {};
    for (const line of readFileSync(path, "utf8").split("\n")) {
      const match = line.match(/^([A-Za-z0-9_]+):\s*(.+?)\s*$/);
      if (match) result[match[1]] = match[2].replace(/^["']|["']$/g, "");
    }
    return result;
  }
  return {};
}

const uiConfig = { ...readYaml("base/ui.yml"), ...readYaml("dev/ui.yml") };
const gatewayOrigin = uiConfig.gateway_origin;
const keycloakOrigin = uiConfig.keycloak_origin;

if (!gatewayOrigin || !keycloakOrigin) {
  throw new Error(
    "configs/base/ui.yml must define gateway_origin and keycloak_origin",
  );
}

const csp = [
  "default-src 'self'",
  "base-uri 'self'",
  "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob:",
  "font-src 'self' data:",
  "connect-src 'self'",
  "frame-ancestors 'none'",
  "form-action 'self'",
  "object-src 'none'",
].join("; ");

const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
  poweredByHeader: false,
  async rewrites() {
    return [
      { source: "/v1/:path*", destination: `${gatewayOrigin}/v1/:path*` },
      { source: "/kc/:path*", destination: `${keycloakOrigin}/:path*` },
    ];
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "Content-Security-Policy", value: csp },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          {
            key: "Permissions-Policy",
            value: "camera=(), microphone=(), geolocation=()",
          },
        ],
      },
    ];
  },
};

export default withNextIntl(nextConfig);
