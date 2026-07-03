"use client";

import { useRouter } from "next/navigation";
import { ROLE_LABELS, useRole, type DemoRole } from "@/shared/lib/role";
import { Isolines } from "@/shared/ui/isolines";

export default function LoginPage() {
  const router = useRouter();
  const setRole = useRole((store) => store.setRole);

  const enter = (role: DemoRole) => {
    setRole(role);
    router.push("/");
  };

  return (
    <div className="glow-panel relative flex min-h-full flex-col items-center justify-center gap-8 px-6 py-16">
      <Isolines />
      <div className="flex flex-col items-center gap-3">
        <span className="stamp-frame flex h-14 w-14 items-center justify-center bg-bg-1 font-display text-2xl font-extrabold text-electrolyte">
          k
        </span>
        <h1 className="font-display text-lg font-bold text-ink-0">kmap</h1>
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          единая карта знаний R&D
        </p>
      </div>
      <div className="flex w-full max-w-xs flex-col gap-2">
        <button
          type="button"
          disabled
          title="Keycloak подключается в prod-режиме"
          className="h-11 rounded-sm bg-electrolyte font-medium text-bg-0 opacity-40"
        >
          Войти через OIDC
        </button>
        <div className="my-2 flex items-center gap-3 text-ink-2">
          <span className="h-px flex-1 bg-line" />
          <span className="font-mono text-[10px] uppercase tracking-wider">
            demo-режим
          </span>
          <span className="h-px flex-1 bg-line" />
        </div>
        {(Object.keys(ROLE_LABELS) as DemoRole[]).map((role) => (
          <button
            key={role}
            type="button"
            onClick={() => enter(role)}
            className="h-10 rounded-sm border border-line bg-bg-1 text-[13px] text-ink-1 transition-colors hover:border-electrolyte hover:text-ink-0"
          >
            {ROLE_LABELS[role]}
          </button>
        ))}
      </div>
    </div>
  );
}
