"use client";

import { CoreApiClient, type Capability, type ChannelOverview, type Collection, type Provider, type Session } from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { type FormEvent, useCallback, useEffect, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");
const emptyMarketplace = { developers: [], publishers: [], listings: [], reviews: [], sandbox_runs: [], quality_scores: [], incidents: [], revenue_share_rules: [], payout_reserves: [], disputes: [], takedowns: [] };
const emptyChannels: ChannelOverview = { reseller_levels: [], resellers: [], attribution_rules: [], lead_ownerships: [], customer_ownerships: [], transfer_requests: [], conflicts: [], commission_rules: [], commission_locks: [], settlement_cycles: [], suppliers: [], supplier_capabilities: [], provider_bindings: [], supplier_contracts: [], supplier_rates: [], supplier_quality: [], provider_payables: [], reseller_settlements: [], supplier_settlements: [], marketplace: emptyMarketplace };
const navigation = [{ label: "供应商总览", href: "#overview" }, { label: "能力与 Provider", href: "#capabilities" }, { label: "合同与费率", href: "#contracts" }, { label: "质量记录", href: "#quality" }, { label: "应付与结算", href: "#payables" }];
function message(error: unknown) { return error instanceof Error ? error.message : "请求失败"; }
function tone(status: string): "green" | "amber" | "gray" { return ["active", "approved", "healthy", "completed", "settled"].includes(status) ? "green" : ["draft", "pending_approval", "open", "partially_settled"].includes(status) ? "amber" : "gray"; }
function submit(event: FormEvent<HTMLFormElement>, action: (values: FormData) => Promise<void>) { event.preventDefault(); const form = event.currentTarget; const values = new FormData(form); void Promise.resolve().then(() => action(values)).then(() => form.reset()).catch(() => undefined); }

export default function Page() {
  const [session, setSession] = useState<Session | null>(null);
  const [channels, setChannels] = useState<ChannelOverview>(emptyChannels);
  const [capabilities, setCapabilities] = useState<Capability[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [authLoading, setAuthLoading] = useState(true);
  const [authBusy, setAuthBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const reload = useCallback(async () => {
    const [channelResult, capabilityResult, providerResult] = await Promise.all([api.get<ChannelOverview>("/v1/channels"), api.get<Collection<Capability>>("/v1/capabilities"), api.get<Collection<Provider>>("/v1/providers")]);
    setChannels(channelResult); setCapabilities(capabilityResult.items); setProviders(providerResult.items);
  }, []);
  useEffect(() => { api.session().then(setSession).catch(() => setSession(null)).finally(() => setAuthLoading(false)); }, []);
  useEffect(() => { if (session) reload().catch(error => setError(message(error))); }, [reload, session]);
  async function login(email: string, password: string) { setAuthBusy(true); setError(null); try { setSession(await api.login(email, password)); } catch (error) { setError(message(error)); } finally { setAuthBusy(false); } }
  async function logout() { await api.logout().catch(() => undefined); setSession(null); setChannels(emptyChannels); }
  async function command(path: string, body: unknown) { setBusy(true); setError(null); try { await api.command(path, body); await reload(); } catch (error) { setError(message(error)); } finally { setBusy(false); } }
  const outstandingMinor = channels.provider_payables.reduce((sum, item) => sum + item.amount_minor - item.settled_minor, 0);
  const averageQuality = channels.supplier_quality.length ? Math.round(channels.supplier_quality.reduce((sum, item) => sum + item.score_bps, 0) / channels.supplier_quality.length) : 0;
  if (authLoading) return <div className="auth-screen"><div className="loading">正在恢复会话...</div></div>;
  if (!session) return <LoginScreen title="供应商协作" error={error} busy={authBusy} onLogin={login} />;

  return <PortalShell title="供应商协作" role="Supplier Portal" navigation={navigation} tenantLabel={session.tenant_name} userLabel={`${session.display_name} · ${session.role}`} onLogout={logout}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics" id="overview">
      <Metric label="供应商" value={String(channels.suppliers.length)} detail={`${channels.provider_bindings.length} 个 Provider 绑定`} />
      <Metric label="生效合同" value={String(channels.supplier_contracts.filter(item => item.status === "active").length)} detail={`${channels.supplier_rates.length} 条费率`} />
      <Metric label="质量均分" value={String(averageQuality)} detail="Basis points" />
      <Metric label="待付金额" value={String(outstandingMinor)} detail={`${channels.supplier_settlements.length} 笔规范结算`} />
    </div>

    <section className="panel" id="capabilities">
      <div className="panel-title"><h2>供应商与执行资源</h2><button className="button" disabled={busy} onClick={() => void reload()}>刷新</button></div>
      <div className="factory-columns">
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/suppliers", { name: values.get("name") }))}><input name="name" required placeholder="供应商名称" /><button className="button primary" disabled={busy}>创建供应商</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command(`/v1/suppliers/${values.get("supplier_id")}/capabilities`, { capability_id: values.get("capability_id") }))}><select name="supplier_id" required defaultValue=""><option value="" disabled>选择供应商</option>{channels.suppliers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="capability_id" required defaultValue=""><option value="" disabled>选择能力</option>{capabilities.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><button className="button primary" disabled={busy || !channels.suppliers.length || !capabilities.length}>绑定能力</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command(`/v1/providers/${values.get("provider_id")}/supplier-binding`, { supplier_id: values.get("supplier_id") }))}><select name="provider_id" required defaultValue=""><option value="" disabled>选择 Provider</option>{providers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="supplier_id" required defaultValue=""><option value="" disabled>选择供应商</option>{channels.suppliers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><button className="button primary" disabled={busy || !channels.suppliers.length || !providers.length}>绑定 Provider</button></form>
      </div>
      <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>Provider</th><th>供应商</th><th>Endpoint</th><th>状态</th></tr></thead><tbody>{providers.map(provider => { const binding = channels.provider_bindings.find(item => item.provider_id === provider.id); return <tr key={provider.id}><td><strong>{provider.name}</strong><div className="muted">{provider.id.slice(0, 8)}</div></td><td>{channels.suppliers.find(item => item.id === binding?.supplier_id)?.name ?? "未绑定"}</td><td>{provider.endpoints.length}</td><td><StatusBadge tone={binding ? "green" : "gray"}>{binding ? "已绑定" : "待绑定"}</StatusBadge></td></tr>; })}</tbody></table></div>
    </section>

    <section className="panel" id="contracts">
      <div className="panel-title"><h2>合同与费率</h2><StatusBadge tone="amber">独立审批</StatusBadge></div>
      <div className="growth-two-column">
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/supplier-contracts", { supplier_id: values.get("supplier_id"), provider_id: values.get("provider_id"), name: values.get("name"), currency: values.get("currency"), terms: JSON.parse(String(values.get("terms"))) }))}><label className="field"><span>供应商</span><select name="supplier_id" required defaultValue=""><option value="" disabled>选择供应商</option>{channels.suppliers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>Provider</span><select name="provider_id" required defaultValue=""><option value="" disabled>选择已绑定 Provider</option>{channels.provider_bindings.map(item => <option key={item.provider_id} value={item.provider_id}>{item.provider_name}</option>)}</select></label><label className="field"><span>合同名称</span><input name="name" required /></label><label className="field"><span>币种</span><input name="currency" minLength={3} maxLength={3} defaultValue="USD" required /></label><label className="field"><span>条款</span><textarea name="terms" defaultValue={'{"settlement_days":30}'} required /></label><button className="button primary" disabled={busy || !channels.provider_bindings.length}>创建合同</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/supplier-rates", { contract_id: values.get("contract_id"), capability_id: values.get("capability_id"), unit: values.get("unit"), rate_minor: Number(values.get("rate_minor")) }))}><label className="field"><span>已批准合同</span><select name="contract_id" required defaultValue=""><option value="" disabled>选择合同</option>{channels.supplier_contracts.filter(item => ["approved", "active"].includes(item.status)).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>能力</span><select name="capability_id" required defaultValue=""><option value="" disabled>选择能力</option>{capabilities.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>计费单位</span><input name="unit" required /></label><label className="field"><span>单位成本</span><input name="rate_minor" type="number" min="0" required /></label><button className="button primary" disabled={busy}>创建费率</button></form>
      </div>
      <div className="record-list">{channels.supplier_contracts.map(item => <div className="record-row" key={item.id}><div><strong>{item.name}</strong><small>{item.currency} · v{item.version} · {channels.supplier_rates.filter(rate => rate.contract_id === item.id).length} 条费率</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><div className="button-row">{item.status === "draft" && <button className="button" disabled={busy} onClick={() => void command(`/v1/supplier-contracts/${item.id}/transitions`, { to: "pending_approval" })}>提交</button>}{item.status === "pending_approval" && session.role !== "operator" && <button className="button primary" disabled={busy} onClick={() => void command(`/v1/supplier-contracts/${item.id}/reviews`, { decision: "approved", rationale: "Contract terms verified" })}>批准</button>}{item.status === "approved" && <button className="button primary" disabled={busy || !channels.supplier_rates.some(rate => rate.contract_id === item.id)} onClick={() => void command(`/v1/supplier-contracts/${item.id}/transitions`, { to: "active" })}>生效</button>}</div></div>)}</div>
    </section>

    <section className="panel" id="quality">
      <div className="panel-title"><h2>质量记录</h2><span className="muted">{channels.supplier_quality.length} 条</span></div>
      <form className="form-grid" onSubmit={event => submit(event, values => { const provider = providers.find(item => item.id === values.get("provider_id")); return command("/v1/supplier-quality-records", { supplier_id: values.get("supplier_id"), provider_id: values.get("provider_id"), provider_endpoint_id: provider?.endpoints[0]?.id ?? "", metric: values.get("metric"), score_bps: Number(values.get("score_bps")), evidence: { sample_size: Number(values.get("sample_size")) }, period_start: new Date(String(values.get("start"))).toISOString(), period_end: new Date(String(values.get("end"))).toISOString() }); })}><label className="field"><span>供应商</span><select name="supplier_id" required defaultValue=""><option value="" disabled>选择供应商</option>{channels.suppliers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>Provider</span><select name="provider_id" required defaultValue=""><option value="" disabled>选择 Provider</option>{providers.filter(item => item.endpoints.length).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>指标</span><input name="metric" required /></label><label className="field"><span>质量分 BPS</span><input name="score_bps" type="number" min="0" max="10000" required /></label><label className="field"><span>样本量</span><input name="sample_size" type="number" min="1" required /></label><label className="field"><span>开始日期</span><input name="start" type="date" required /></label><label className="field"><span>结束日期</span><input name="end" type="date" required /></label><div className="form-actions wide"><button className="button primary" disabled={busy}>记录质量</button></div></form>
      <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>供应商</th><th>指标</th><th>得分</th><th>期间</th></tr></thead><tbody>{channels.supplier_quality.map(item => <tr key={item.id}><td>{channels.suppliers.find(value => value.id === item.supplier_id)?.name ?? item.supplier_id.slice(0, 8)}</td><td>{item.metric}</td><td><StatusBadge tone={item.score_bps >= 8000 ? "green" : "amber"}>{item.score_bps}</StatusBadge></td><td>{new Date(item.period_start).toLocaleDateString("zh-CN")} - {new Date(item.period_end).toLocaleDateString("zh-CN")}</td></tr>)}</tbody></table></div>
    </section>

    <section className="panel" id="payables">
      <div className="panel-title"><h2>规范应付与结算</h2><span className="muted">ProviderPayable → Settlement</span></div>
      <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>Provider</th><th>应付金额</th><th>已结</th><th>状态</th></tr></thead><tbody>{channels.provider_payables.map(item => <tr key={item.id}><td>{providers.find(value => value.id === item.provider_id)?.name ?? item.provider_id.slice(0, 8)}</td><td>{item.currency} {item.amount_minor}</td><td>{item.settled_minor}</td><td><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge></td></tr>)}</tbody></table></div>
      <div className="record-list">{channels.supplier_settlements.map(item => <div className="record-row" key={item.id}><div><strong>{item.currency} {item.amount_minor}</strong><small>{item.source_type} · {item.source_id.slice(0, 8)}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><span>{new Date(item.created_at).toLocaleDateString("zh-CN")}</span></div>)}</div>
    </section>
  </PortalShell>;
}
