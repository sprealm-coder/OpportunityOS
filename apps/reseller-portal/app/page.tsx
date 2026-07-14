"use client";

import { CoreApiClient, type ChannelOverview, type FinanceOverview, type GrowthOverview, type Session } from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");
const emptyMarketplace = { developers: [], publishers: [], listings: [], reviews: [], sandbox_runs: [], quality_scores: [], incidents: [], revenue_share_rules: [], payout_reserves: [], disputes: [], takedowns: [] };
const emptyChannels: ChannelOverview = { reseller_levels: [], resellers: [], attribution_rules: [], lead_ownerships: [], customer_ownerships: [], transfer_requests: [], conflicts: [], commission_rules: [], commission_locks: [], settlement_cycles: [], suppliers: [], supplier_capabilities: [], provider_bindings: [], supplier_contracts: [], supplier_rates: [], supplier_quality: [], provider_payables: [], reseller_settlements: [], supplier_settlements: [], marketplace: emptyMarketplace };
const emptyGrowth: GrowthOverview = { segments: [], icps: [], leads: [], evidence: [], contacts: [], proof_templates: [], proof_requests: [], proof_instances: [], campaigns: [], suppressions: [], outreach: [], conversations: [], deals: [], experiments: [] };
const emptyFinance: FinanceOverview = { wallets: [], accounts: [], transactions: [], holds: [], refunds: [], commissions: [], provider_payables: [], settlements: [], reconciliation_runs: [] };
const navigation = [{ label: "渠道总览", href: "#overview" }, { label: "潜客归属", href: "#ownership" }, { label: "转移审批", href: "#transfers" }, { label: "佣金锁定", href: "#commission" }, { label: "结算周期", href: "#settlement" }];

function message(error: unknown) { return error instanceof Error ? error.message : "请求失败"; }
function tone(status: string): "green" | "amber" | "gray" { return ["active", "approved", "protected", "posted", "completed", "settled"].includes(status) ? "green" : ["pending", "open", "conflicted"].includes(status) ? "amber" : "gray"; }
function submit(event: FormEvent<HTMLFormElement>, action: (values: FormData) => Promise<void>) { event.preventDefault(); const form = event.currentTarget; const values = new FormData(form); void Promise.resolve().then(() => action(values)).then(() => form.reset()).catch(() => undefined); }

export default function Page() {
  const [session, setSession] = useState<Session | null>(null);
  const [channels, setChannels] = useState<ChannelOverview>(emptyChannels);
  const [growth, setGrowth] = useState<GrowthOverview>(emptyGrowth);
  const [finance, setFinance] = useState<FinanceOverview>(emptyFinance);
  const [authLoading, setAuthLoading] = useState(true);
  const [authBusy, setAuthBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    const [channelResult, growthResult, financeResult] = await Promise.all([api.get<ChannelOverview>("/v1/channels"), api.get<GrowthOverview>("/v1/growth"), api.get<FinanceOverview>("/v1/finance")]);
    setChannels(channelResult); setGrowth(growthResult); setFinance(financeResult);
  }, []);
  useEffect(() => { api.session().then(setSession).catch(() => setSession(null)).finally(() => setAuthLoading(false)); }, []);
  useEffect(() => { if (session) reload().catch(error => setError(message(error))); }, [reload, session]);

  async function login(email: string, password: string) { setAuthBusy(true); setError(null); try { setSession(await api.login(email, password)); } catch (error) { setError(message(error)); } finally { setAuthBusy(false); } }
  async function logout() { await api.logout().catch(() => undefined); setSession(null); setChannels(emptyChannels); }
  async function command(path: string, body: unknown) { setBusy(true); setError(null); try { await api.command(path, body); await reload(); } catch (error) { setError(message(error)); } finally { setBusy(false); } }

  const activeOwnerships = channels.lead_ownerships.filter(item => item.status === "protected");
  const openCommissions = useMemo(() => finance.commissions.filter(item => item.beneficiary_type === "reseller" && item.status !== "reversed" && !channels.commission_locks.some(lock => lock.commission_id === item.id)), [channels.commission_locks, finance.commissions]);
  const lockedMinor = channels.commission_locks.reduce((total, item) => total + item.amount_minor, 0);
  if (authLoading) return <div className="auth-screen"><div className="loading">正在恢复会话...</div></div>;
  if (!session) return <LoginScreen title="渠道合作伙伴" error={error} busy={authBusy} onLogin={login} />;

  return <PortalShell title="渠道合作伙伴" role="Reseller Portal" navigation={navigation} tenantLabel={session.tenant_name} userLabel={`${session.display_name} · ${session.role}`} onLogout={logout}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics" id="overview">
      <Metric label="受保护潜客" value={String(activeOwnerships.length)} detail={`${channels.customer_ownerships.length} 个客户归属`} />
      <Metric label="待审转移" value={String(channels.transfer_requests.filter(item => item.status === "pending").length)} detail={`${channels.conflicts.filter(item => item.status === "open").length} 个冲突`} />
      <Metric label="已锁佣金" value={String(lockedMinor)} detail="最小货币单位" />
      <Metric label="结算周期" value={String(channels.settlement_cycles.length)} detail={`${channels.reseller_settlements.length} 笔规范结算`} />
    </div>

    <section className="panel" id="ownership">
      <div className="panel-title"><h2>渠道与归因</h2><button className="button" disabled={busy} onClick={() => void reload()}>刷新</button></div>
      <div className="factory-columns">
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/reseller-levels", { name: values.get("name"), rank: Number(values.get("rank")), default_commission_bps: Number(values.get("bps")) }))}><input name="name" required placeholder="等级名称" /><input name="rank" type="number" min="1" required placeholder="等级序号" /><input name="bps" type="number" min="0" max="10000" required placeholder="默认佣金 BPS" /><button className="button primary" disabled={busy}>创建等级</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/resellers", { name: values.get("name"), level_id: values.get("level_id") }))}><input name="name" required placeholder="渠道名称" /><select name="level_id" required defaultValue=""><option value="" disabled>选择等级</option>{channels.reseller_levels.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><button className="button primary" disabled={busy || !channels.reseller_levels.length}>创建渠道</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/attribution-rules", { name: values.get("name"), priority: Number(values.get("priority")), definition: { method: values.get("method") } }))}><input name="name" required placeholder="归因规则名称" /><input name="priority" type="number" min="1" required placeholder="优先级" /><select name="method" defaultValue="first_verified_evidence"><option value="first_verified_evidence">首个有效证据</option><option value="last_verified_touch">最近有效触点</option></select><button className="button primary" disabled={busy}>创建规则</button></form>
      </div>
      <div className="growth-two-column">
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/lead-ownerships", { lead_id: values.get("lead_id"), reseller_id: values.get("reseller_id"), attribution_rule_id: values.get("rule_id"), protection_days: Number(values.get("days")) }))}><label className="field"><span>潜客</span><select name="lead_id" required defaultValue=""><option value="" disabled>选择潜客</option>{growth.leads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>渠道</span><select name="reseller_id" required defaultValue=""><option value="" disabled>选择渠道</option>{channels.resellers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>归因规则</span><select name="rule_id" required defaultValue=""><option value="" disabled>选择规则</option>{channels.attribution_rules.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>保护天数</span><input name="days" type="number" min="1" max="730" defaultValue="30" required /></label><button className="button primary" disabled={busy || !growth.leads.length || !channels.resellers.length || !channels.attribution_rules.length}>分配潜客归属</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/customer-ownerships", { customer_id: values.get("customer_id"), reseller_id: values.get("reseller_id"), source_lead_ownership_id: values.get("source_id"), protection_days: Number(values.get("days")) }))}><label className="field"><span>客户标识</span><input name="customer_id" required /></label><label className="field"><span>渠道</span><select name="reseller_id" required defaultValue=""><option value="" disabled>选择渠道</option>{channels.resellers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>来源归属</span><select name="source_id" defaultValue=""><option value="">无</option>{activeOwnerships.map(item => <option key={item.id} value={item.id}>{item.lead_id.slice(0, 8)}</option>)}</select></label><label className="field"><span>保护天数</span><input name="days" type="number" min="1" max="730" defaultValue="90" required /></label><button className="button primary" disabled={busy || !channels.resellers.length}>创建客户归属</button></form>
      </div>
      <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>对象</th><th>渠道</th><th>状态</th><th>保护截止</th></tr></thead><tbody>{activeOwnerships.map(item => <tr key={item.id}><td>Lead · {item.lead_id.slice(0, 8)}</td><td>{channels.resellers.find(value => value.id === item.reseller_id)?.name ?? item.reseller_id.slice(0, 8)}</td><td><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge></td><td>{new Date(item.protection_expires_at).toLocaleDateString("zh-CN")}</td></tr>)}{channels.customer_ownerships.map(item => <tr key={item.id}><td>Customer · {item.customer_id}</td><td>{channels.resellers.find(value => value.id === item.reseller_id)?.name ?? item.reseller_id.slice(0, 8)}</td><td><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge></td><td>{new Date(item.protection_expires_at).toLocaleDateString("zh-CN")}</td></tr>)}</tbody></table></div>
    </section>

    <section className="panel" id="transfers">
      <div className="panel-title"><h2>归属转移</h2><StatusBadge tone="amber">审批分离</StatusBadge></div>
      <form className="form-grid" onSubmit={event => submit(event, values => { const [type, id] = String(values.get("ownership")).split(":"); return command("/v1/ownership-transfers", { ownership_type: type, ownership_id: id, to_reseller_id: values.get("to_reseller_id"), rationale: values.get("rationale") }); })}><label className="field"><span>当前归属</span><select name="ownership" required defaultValue=""><option value="" disabled>选择归属</option>{channels.lead_ownerships.filter(item => item.status === "protected").map(item => <option key={item.id} value={`lead:${item.id}`}>Lead · {item.lead_id.slice(0, 8)}</option>)}{channels.customer_ownerships.filter(item => item.status === "protected").map(item => <option key={item.id} value={`customer:${item.id}`}>Customer · {item.customer_id}</option>)}</select></label><label className="field"><span>目标渠道</span><select name="to_reseller_id" required defaultValue=""><option value="" disabled>选择渠道</option>{channels.resellers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field wide"><span>转移依据</span><input name="rationale" required /></label><div className="form-actions wide"><button className="button primary" disabled={busy}>提交转移</button></div></form>
      <div className="record-list">{channels.transfer_requests.map(item => <div className="record-row" key={item.id}><div><strong>{item.ownership_type} · {item.ownership_id.slice(0, 8)}</strong><small>{item.rationale}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><span>{item.to_reseller_id.slice(0, 8)}</span></div>)}</div>
    </section>

    <section className="panel" id="commission">
      <div className="panel-title"><h2>规范佣金锁</h2><span className="muted">Commission → CommissionLock</span></div>
      <div className="growth-two-column">
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/commission-rules", { name: values.get("name"), reseller_id: values.get("reseller_id"), basis_points: Number(values.get("bps")) }))}><input name="name" required placeholder="佣金规则名称" /><select name="reseller_id" required defaultValue=""><option value="" disabled>选择渠道</option>{channels.resellers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="bps" type="number" min="0" max="10000" required placeholder="佣金 BPS" /><button className="button primary" disabled={busy || !channels.resellers.length}>创建佣金规则</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => { const commission = openCommissions.find(item => item.id === values.get("commission_id")); return command("/v1/commission-locks", { commission_id: values.get("commission_id"), commission_rule_id: values.get("rule_id"), reseller_id: commission?.beneficiary_id }); })}><select name="commission_id" required defaultValue=""><option value="" disabled>选择规范 Commission</option>{openCommissions.map(item => <option key={item.id} value={item.id}>{item.currency} {item.amount_minor} · {item.beneficiary_id.slice(0, 12)}</option>)}</select><select name="rule_id" required defaultValue=""><option value="" disabled>选择规则</option>{channels.commission_rules.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><button className="button primary" disabled={busy || !openCommissions.length || !channels.commission_rules.length}>锁定佣金</button></form>
      </div>
      <div className="record-list">{channels.commission_locks.map(item => <div className="record-row" key={item.id}><div><strong>{item.currency} {item.amount_minor}</strong><small>Commission {item.commission_id?.slice(0, 8)}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><span>{item.reseller_id.slice(0, 8)}</span></div>)}</div>
    </section>

    <section className="panel" id="settlement">
      <div className="panel-title"><h2>结算周期</h2><span className="muted">{channels.reseller_settlements.length} 笔规范 Settlement</span></div>
      <form className="form-grid" onSubmit={event => submit(event, values => command("/v1/settlement-cycles", { reseller_id: values.get("reseller_id"), name: values.get("name"), period_start: new Date(String(values.get("start"))).toISOString(), period_end: new Date(String(values.get("end"))).toISOString() }))}><label className="field"><span>渠道</span><select name="reseller_id" required defaultValue=""><option value="" disabled>选择渠道</option>{channels.resellers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>周期名称</span><input name="name" required /></label><label className="field"><span>开始日期</span><input name="start" type="date" required /></label><label className="field"><span>结束日期</span><input name="end" type="date" required /></label><div className="form-actions wide"><button className="button primary" disabled={busy}>创建周期</button></div></form>
      <div className="record-list">{channels.settlement_cycles.map(item => <div className="record-row" key={item.id}><div><strong>{item.name}</strong><small>{new Date(item.period_start).toLocaleDateString("zh-CN")} - {new Date(item.period_end).toLocaleDateString("zh-CN")}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><span>{item.reseller_id.slice(0, 8)}</span></div>)}</div>
    </section>
  </PortalShell>;
}
