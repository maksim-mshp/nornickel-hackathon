"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

export type DemoRole =
  | "researcher"
  | "manager"
  | "expert"
  | "admin"
  | "partner";

export const ROLE_LABELS: Record<DemoRole, string> = {
  researcher: "Исследователь",
  manager: "Руководитель",
  expert: "Эксперт",
  admin: "Администратор",
  partner: "Внешний партнёр",
};

const RESEARCHER_ROUTES = ["/", "/experiments", "/experts"];

export const ROLE_ROUTES: Record<DemoRole, string[]> = {
  researcher: RESEARCHER_ROUTES,
  manager: [...RESEARCHER_ROUTES, "/coverage"],
  expert: [...RESEARCHER_ROUTES, "/coverage", "/review"],
  admin: [
    ...RESEARCHER_ROUTES,
    "/coverage",
    "/review",
    "/documents",
    "/dictionaries",
  ],
  partner: ["/"],
};

export const DEFAULT_ROLE: DemoRole = "partner";

const PUBLIC_ROUTES = ["/login"];
const DETAIL_ROUTES = ["/entity", "/answers"];

type RoleStore = {
  role: DemoRole;
  token: string;
  setRole: (role: DemoRole) => void;
  setAuth: (role: DemoRole, token: string) => void;
};

export const useRole = create<RoleStore>()(
  persist(
    (set) => ({
      role: DEFAULT_ROLE,
      token: `demo-${DEFAULT_ROLE}`,
      setRole: (role) => set({ role, token: `demo-${role}` }),
      setAuth: (role, token) => set({ role, token }),
    }),
    { name: "kmap-role" },
  ),
);

function matchesPrefix(route: string, href: string): boolean {
  if (route === "/") return href === "/";
  return href === route || href.startsWith(`${route}/`);
}

export function routeAllowed(role: DemoRole, href: string): boolean {
  if (PUBLIC_ROUTES.some((route) => matchesPrefix(route, href))) return true;
  if (DETAIL_ROUTES.some((route) => matchesPrefix(route, href))) return true;
  return ROLE_ROUTES[role].some((route) => matchesPrefix(route, href));
}

export function authHeaders(): Record<string, string> {
  const token = useRole.getState().token;
  return token ? { Authorization: `Bearer ${token}` } : {};
}

const ROLE_PRIORITY: { claim: string; role: DemoRole }[] = [
  { claim: "admin", role: "admin" },
  { claim: "expert", role: "expert" },
  { claim: "manager", role: "manager" },
  { claim: "analyst", role: "researcher" },
  { claim: "researcher", role: "researcher" },
  { claim: "partner", role: "partner" },
];

function decodeBase64Url(segment: string): string {
  let base64 = segment.replace(/-/g, "+").replace(/_/g, "/");
  while (base64.length % 4) base64 += "=";
  const binary = atob(base64);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

function decodeRoles(token: string): string[] {
  const parts = token.split(".");
  if (parts.length < 2) return [];
  try {
    const payload = JSON.parse(decodeBase64Url(parts[1])) as {
      realm_access?: { roles?: string[] };
    };
    return payload.realm_access?.roles ?? [];
  } catch {
    return [];
  }
}

export function roleFromToken(token: string): DemoRole {
  const roles = new Set(decodeRoles(token));
  const match = ROLE_PRIORITY.find((entry) => roles.has(entry.claim));
  return match?.role ?? "researcher";
}

export async function loginOIDC(
  username: string,
  password: string,
): Promise<DemoRole> {
  const body = new URLSearchParams({
    grant_type: "password",
    client_id: "kmap-ui",
    username,
    password,
  });
  const response = await fetch(
    "/kc/realms/kmap/protocol/openid-connect/token",
    {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
    },
  );
  if (!response.ok) {
    throw new Error("Keycloak отклонил вход");
  }
  const data = (await response.json()) as { access_token?: string };
  const token = data.access_token ?? "";
  if (!token) {
    throw new Error("Keycloak не вернул токен доступа");
  }
  const role = roleFromToken(token);
  useRole.getState().setAuth(role, token);
  return role;
}
