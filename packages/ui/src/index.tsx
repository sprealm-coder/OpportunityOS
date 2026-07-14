import type { ReactNode } from "react";

export type NavigationItem = { label: string; href: string; enabled?: boolean };

export function PortalShell({ title, role, navigation, tenantLabel = "OpportunityOS Development", children }: { title: string; role: string; navigation: NavigationItem[]; tenantLabel?: string; children: ReactNode }) {
  return <div className="portal-shell"><aside><div className="portal-brand"><span>O</span><strong>OpportunityOS</strong></div><p className="portal-role">{role}</p><nav>{navigation.map(item => <a key={item.label} href={item.href} aria-disabled={item.enabled === false} className={item.enabled === false ? "disabled" : ""}>{item.label}{item.enabled === false && <small>受功能开关控制</small>}</a>)}</nav></aside><main><header><div><small>CORE PLATFORM</small><h1>{title}</h1></div><div className="tenant-chip">{tenantLabel}</div></header>{children}</main></div>;
}

export function StatusBadge({ children, tone = "green" }: { children: ReactNode; tone?: "green" | "amber" | "gray" }) { return <span className={`status-badge ${tone}`}>{children}</span>; }
export function Metric({ label, value, detail }: { label: string; value: string; detail: string }) { return <div className="metric"><span>{label}</span><strong>{value}</strong><small>{detail}</small></div>; }
