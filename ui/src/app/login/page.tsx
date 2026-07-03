"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { loginOIDC, ROLE_LABELS, useRole, type DemoRole } from "@/shared/lib/role";
import { Isolines } from "@/shared/ui/isolines";

export default function LoginPage() {
  const router = useRouter();
  const setRole = useRole((store) => store.setRole);

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const enter = (role: DemoRole) => {
    setRole(role);
    router.push("/");
  };

  const submitOIDC = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await loginOIDC(username.trim(), password);
      router.push("/");
    } catch {
      setError("Неверный логин или Keycloak недоступен");
      setBusy(false);
    }
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
        <form onSubmit={submitOIDC} className="flex flex-col gap-2">
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Keycloak-логин (напр. expert)"
            autoComplete="username"
            className="h-10 rounded-sm border border-line bg-bg-1 px-3 text-[13px] text-ink-0 placeholder:text-ink-2 focus:border-electrolyte focus:outline-none"
          />
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Пароль"
            autoComplete="current-password"
            className="h-10 rounded-sm border border-line bg-bg-1 px-3 text-[13px] text-ink-0 placeholder:text-ink-2 focus:border-electrolyte focus:outline-none"
          />
          <button
            type="submit"
            disabled={busy || username === "" || password === ""}
            className="h-11 rounded-sm bg-electrolyte font-medium text-bg-0 transition-opacity disabled:opacity-40"
          >
            {busy ? "Вход…" : "Войти через OIDC (Keycloak)"}
          </button>
          {error && (
            <p className="text-[11px] text-melt" role="alert">
              {error}
            </p>
          )}
        </form>
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
