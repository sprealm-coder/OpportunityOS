"use client";

import type { FormEvent, ReactNode } from "react";

export type NavigationItem = { label: string; href: string; enabled?: boolean };

export function PortalShell({ title, role, navigation, tenantLabel = "OpportunityOS Development", userLabel, onLogout, children }: { title: string; role: string; navigation: NavigationItem[]; tenantLabel?: string; userLabel?: string; onLogout?: () => void; children: ReactNode }) {
  return <div className="portal-shell"><aside><div className="portal-brand"><span>O</span><strong>OpportunityOS</strong></div><p className="portal-role">{role}</p><nav>{navigation.map(item => <a key={item.label} href={item.href} aria-disabled={item.enabled === false} className={item.enabled === false ? "disabled" : ""}>{item.label}{item.enabled === false && <small>受功能开关控制</small>}</a>)}</nav></aside><main><header><div><small>CORE PLATFORM</small><h1>{title}</h1></div><div className="identity-actions"><div className="tenant-chip"><strong>{tenantLabel}</strong>{userLabel && <span>{userLabel}</span>}</div>{onLogout && <button className="icon-command" type="button" onClick={onLogout} title="退出当前会话" aria-label="退出当前会话">退出</button>}</div></header>{children}</main></div>;
}

export function LoginScreen({ title, error, busy, onLogin }: { title: string; error?: string | null; busy?: boolean; onLogin: (email: string, password: string) => void | Promise<void> }) {
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    void onLogin(String(data.get("email") ?? ""), String(data.get("password") ?? ""));
  }
  return <main className="auth-screen"><div className="auth-brand"><span>O</span><strong>OpportunityOS</strong></div><form className="auth-panel" onSubmit={submit}><small>CORE PLATFORM</small><h1>{title}</h1>{error && <div className="notice error" role="alert">{error}</div>}<label className="field"><span>邮箱</span><input name="email" type="email" autoComplete="username" defaultValue="admin@opportunity.local" required /></label><label className="field"><span>密码</span><input name="password" type="password" autoComplete="current-password" required /></label><button className="button primary" disabled={busy}>{busy ? "登录中..." : "登录"}</button></form></main>;
}

export function StatusBadge({ children, tone = "green" }: { children: ReactNode; tone?: "green" | "amber" | "gray" }) { return <span className={`status-badge ${tone}`}>{children}</span>; }
export function Metric({ label, value, detail }: { label: string; value: string; detail: string }) { return <div className="metric"><span>{label}</span><strong>{value}</strong><small>{detail}</small></div>; }
