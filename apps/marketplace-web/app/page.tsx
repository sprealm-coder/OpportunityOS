"use client";

import { CoreApiClient, type ChannelOverview, type Listing, type Session } from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");
const emptyMarketplace = { developers: [], publishers: [], listings: [], reviews: [], sandbox_runs: [], quality_scores: [], incidents: [], revenue_share_rules: [], payout_reserves: [], disputes: [], takedowns: [] };
const emptyChannels: ChannelOverview = { reseller_levels: [], resellers: [], attribution_rules: [], lead_ownerships: [], customer_ownerships: [], transfer_requests: [], conflicts: [], commission_rules: [], commission_locks: [], settlement_cycles: [], suppliers: [], supplier_capabilities: [], provider_bindings: [], supplier_contracts: [], supplier_rates: [], supplier_quality: [], provider_payables: [], reseller_settlements: [], supplier_settlements: [], marketplace: emptyMarketplace };
const navigation = [{ label: "市场总览", href: "#overview" }, { label: "开发者与发布者", href: "#publishers" }, { label: "Listing 工作流", href: "#listings" }, { label: "沙箱与质量", href: "#release" }, { label: "争议与下架", href: "#governance" }];
const stages = ["draft", "submitted", "automated_review", "manual_review", "sandbox_testing", "limited_release", "published"];
function message(error: unknown) { return error instanceof Error ? error.message : "请求失败"; }
function tone(status: string): "green" | "amber" | "gray" { return ["approved", "succeeded", "published", "resolved"].includes(status) ? "green" : ["submitted", "automated_review", "manual_review", "sandbox_testing", "limited_release", "requested", "open"].includes(status) ? "amber" : "gray"; }
function submit(event: FormEvent<HTMLFormElement>, action: (values: FormData) => Promise<void>) { event.preventDefault(); const form = event.currentTarget; const values = new FormData(form); void Promise.resolve().then(() => action(values)).then(() => form.reset()).catch(() => undefined); }

export default function Page() {
  const [session, setSession] = useState<Session | null>(null);
  const [channels, setChannels] = useState<ChannelOverview>(emptyChannels);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [authLoading, setAuthLoading] = useState(true);
  const [authBusy, setAuthBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const reload = useCallback(async () => { const result = await api.get<ChannelOverview>("/v1/channels"); setChannels(result); setSelectedID(current => current ?? result.marketplace.listings[0]?.id ?? null); }, []);
  useEffect(() => { api.session().then(setSession).catch(() => setSession(null)).finally(() => setAuthLoading(false)); }, []);
  useEffect(() => { if (session) reload().catch(error => setError(message(error))); }, [reload, session]);
  async function login(email: string, password: string) { setAuthBusy(true); setError(null); try { setSession(await api.login(email, password)); } catch (error) { setError(message(error)); } finally { setAuthBusy(false); } }
  async function logout() { await api.logout().catch(() => undefined); setSession(null); setChannels(emptyChannels); }
  async function command(path: string, body: unknown) { setBusy(true); setError(null); try { await api.command(path, body); await reload(); } catch (error) { setError(message(error)); } finally { setBusy(false); } }
  const selected = useMemo(() => channels.marketplace.listings.find(item => item.id === selectedID) ?? null, [channels.marketplace.listings, selectedID]);
  const latestVersion = selected?.versions[0] ?? null;
  const versionReviews = latestVersion ? channels.marketplace.reviews.filter(item => item.listing_version_id === latestVersion.id) : [];
  const sandbox = latestVersion ? channels.marketplace.sandbox_runs.find(item => item.listing_version_id === latestVersion.id && item.status === "succeeded") : null;
  const quality = latestVersion ? channels.marketplace.quality_scores.find(item => item.listing_version_id === latestVersion.id) : null;
  if (authLoading) return <div className="auth-screen"><div className="loading">正在恢复会话...</div></div>;
  if (!session) return <LoginScreen title="内部开发者市场" error={error} busy={authBusy} onLogin={login} />;

  return <PortalShell title="内部开发者市场" role="Marketplace" navigation={navigation} tenantLabel={session.tenant_name} userLabel={`${session.display_name} · ${session.role}`} onLogout={logout}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics" id="overview">
      <Metric label="Listing" value={String(channels.marketplace.listings.length)} detail={`${channels.marketplace.publishers.length} 个发布者`} />
      <Metric label="审核中" value={String(channels.marketplace.listings.filter(item => item.status.includes("review") || item.status === "sandbox_testing").length)} detail={`${channels.marketplace.reviews.length} 条审核记录`} />
      <Metric label="已发布" value={String(channels.marketplace.listings.filter(item => item.status === "published").length)} detail={`${channels.marketplace.quality_scores.length} 个质量评分`} />
      <Metric label="治理事项" value={String(channels.marketplace.disputes.filter(item => item.status === "open").length + channels.marketplace.takedowns.filter(item => item.status === "requested").length)} detail="争议与下架" />
    </div>
    <div className="object-chain">{stages.map(stage => <div key={stage}><span>{stage}</span><small>{channels.marketplace.listings.filter(item => item.status === stage).length}</small></div>)}</div>

    <section className="panel" id="publishers">
      <div className="panel-title"><h2>开发者与发布者</h2><button className="button" disabled={busy} onClick={() => void reload()}>刷新</button></div>
      <div className="factory-columns">
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/developers", { name: values.get("name") }))}><input name="name" required placeholder="开发者名称" /><button className="button primary" disabled={busy}>创建开发者</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/publishers", { developer_id: values.get("developer_id"), name: values.get("name") }))}><select name="developer_id" required defaultValue=""><option value="" disabled>选择开发者</option>{channels.marketplace.developers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="name" required placeholder="发布者名称" /><button className="button primary" disabled={busy || !channels.marketplace.developers.length}>创建发布者</button></form>
        <form className="inline-form" onSubmit={event => submit(event, values => command("/v1/listings", { publisher_id: values.get("publisher_id"), name: values.get("name"), listing_type: values.get("listing_type") }))}><select name="publisher_id" required defaultValue=""><option value="" disabled>选择发布者</option>{channels.marketplace.publishers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="name" required placeholder="Listing 名称" /><select name="listing_type" defaultValue="workflow"><option value="adapter">Adapter</option><option value="capability">Capability</option><option value="workflow">Workflow</option><option value="agent">Agent</option><option value="mcp">MCP</option><option value="business_blueprint">Business Blueprint</option><option value="pricing_template">Pricing Template</option><option value="growth_playbook">Growth Playbook</option></select><button className="button primary" disabled={busy || !channels.marketplace.publishers.length}>创建 Listing</button></form>
      </div>
    </section>

    <section className="panel" id="listings">
      <div className="panel-title"><h2>Listing 工作流</h2><span className="muted">{selected?.name ?? "未选择"}</span></div>
      <div className="workspace-grid">
        <div className="record-list">{channels.marketplace.listings.map(item => <button type="button" className={`transaction-row transaction-select ${item.id === selectedID ? "active" : ""}`} key={item.id} onClick={() => setSelectedID(item.id)}><div><strong>{item.name}</strong><small>{item.listing_type} · {item.versions.length} 个版本</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><span>v{item.version}</span></button>)}</div>
        <ListingControl listing={selected} latestVersion={latestVersion} reviews={versionReviews} sandboxPassed={Boolean(sandbox)} qualityScore={quality?.score_bps ?? 0} role={session.role} busy={busy} onCommand={command} />
      </div>
    </section>

    <section className="panel" id="release">
      <div className="panel-title"><h2>沙箱与质量</h2><span className="muted">受控策略，不执行上传代码</span></div>
      <div className="growth-two-column">
        <div><h3>沙箱运行</h3><div className="record-list">{channels.marketplace.sandbox_runs.map(item => <div className="record-row" key={item.id}><div><strong>{item.listing_version_id.slice(0, 8)}</strong><small>{String(item.result.contract_tests ?? "-")}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge><span>{new Date(item.started_at).toLocaleDateString("zh-CN")}</span></div>)}</div></div>
        <div><h3>质量评分</h3><div className="record-list">{channels.marketplace.quality_scores.map(item => <div className="record-row" key={item.id}><div><strong>{item.score_bps}</strong><small>{item.listing_version_id.slice(0, 8)}</small></div><StatusBadge tone={item.score_bps >= 8000 ? "green" : "amber"}>{item.score_bps >= 8000 ? "达标" : "未达标"}</StatusBadge><span>{new Date(item.created_at).toLocaleDateString("zh-CN")}</span></div>)}</div></div>
      </div>
    </section>

    <section className="panel" id="governance">
      <div className="panel-title"><h2>争议与下架</h2><StatusBadge tone="amber">审计保留</StatusBadge></div>
      <div className="growth-two-column">
        <div><h3>争议</h3><div className="record-list">{channels.marketplace.disputes.map(item => <div className="record-row" key={item.id}><div><strong>{item.reason}</strong><small>{item.claimant_type} · {item.listing_id.slice(0, 8)}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge>{item.status === "open" && session.role !== "operator" ? <button className="button" disabled={busy} onClick={() => void command(`/v1/marketplace-disputes/${item.id}/resolutions`, { decision: "resolved", resolution: { outcome: "remediated" } })}>解决</button> : <span>{item.resolved_by?.slice(0, 8) ?? "-"}</span>}</div>)}</div></div>
        <div><h3>下架</h3><div className="record-list">{channels.marketplace.takedowns.map(item => <div className="record-row" key={item.id}><div><strong>{item.reason}</strong><small>{item.listing_id.slice(0, 8)}</small></div><StatusBadge tone={tone(item.status)}>{item.status}</StatusBadge>{item.status === "requested" && session.role !== "operator" ? <button className="button danger" disabled={busy} onClick={() => void command(`/v1/takedowns/${item.id}/reviews`, { decision: "approved", rationale: "Policy violation verified" })}>批准下架</button> : <span>{item.reviewed_by?.slice(0, 8) ?? "-"}</span>}</div>)}</div></div>
      </div>
    </section>
  </PortalShell>;
}

function ListingControl({ listing, latestVersion, reviews, sandboxPassed, qualityScore, role, busy, onCommand }: { listing: Listing | null; latestVersion: Listing["versions"][number] | null; reviews: ChannelOverview["marketplace"]["reviews"]; sandboxPassed: boolean; qualityScore: number; role: string; busy: boolean; onCommand: (path: string, body: unknown) => Promise<void> }) {
  if (!listing) return <div className="empty-feature">暂无 Listing</div>;
  const reviewed = (type: string) => reviews.some(item => item.review_type === type && item.decision === "approved");
  function transition(to: string) { void onCommand(`/v1/listings/${listing!.id}/transitions`, { to }); }
  function review(type: string) { if (!latestVersion) return; void onCommand(`/v1/listings/${listing!.id}/reviews`, { listing_version_id: latestVersion.id, review_type: type, decision: "approved", rationale: `${type} policy verified` }); }
  return <div className="detail-panel">
    <div className="detail-body"><div className="detail-meta"><div><span>类型</span><strong>{listing.listing_type}</strong></div><div><span>状态</span><strong>{listing.status}</strong></div><div><span>聚合版本</span><strong>v{listing.version}</strong></div><div><span>内容版本</span><strong>{latestVersion ? `v${latestVersion.version}` : "-"}</strong></div></div></div>
    {listing.status === "draft" && !latestVersion && <form className="inline-form command-section" onSubmit={event => submit(event, values => onCommand(`/v1/listings/${listing.id}/versions`, { capability_manifest: JSON.parse(String(values.get("capability_manifest"))), permission_manifest: JSON.parse(String(values.get("permission_manifest"))), content_ref: values.get("content_ref"), checksum: values.get("checksum") }))}><textarea name="capability_manifest" defaultValue={'{"capabilities":[]}'} required /><textarea name="permission_manifest" defaultValue={'{"scopes":[]}'} required /><input name="content_ref" required placeholder="内容引用" /><input name="checksum" required placeholder="内容校验和" /><button className="button primary" disabled={busy}>创建不可变版本</button></form>}
    {listing.status === "draft" && latestVersion && <div className="command-section"><button className="button primary" disabled={busy} onClick={() => transition("submitted")}>提交 Listing</button></div>}
    {listing.status === "submitted" && <div className="command-section"><button className="button primary" disabled={busy} onClick={() => transition("automated_review")}>进入自动审核</button></div>}
    {listing.status === "automated_review" && <div className="command-section"><div className="button-row">{role !== "operator" && !reviewed("automated") && <button className="button primary" disabled={busy} onClick={() => review("automated")}>批准自动审核</button>}{reviewed("automated") && <button className="button primary" disabled={busy} onClick={() => transition("manual_review")}>进入人工审核</button>}</div></div>}
    {listing.status === "manual_review" && <div className="command-section"><div className="button-row">{["security", "license", "manual"].map(type => !reviewed(type) && role !== "operator" ? <button className="button" key={type} disabled={busy} onClick={() => review(type)}>批准 {type}</button> : <StatusBadge key={type} tone={reviewed(type) ? "green" : "gray"}>{type}</StatusBadge>)}</div>{["security", "license", "manual"].every(reviewed) && <button className="button primary" disabled={busy} onClick={() => transition("sandbox_testing")}>进入沙箱</button>}</div>}
    {listing.status === "sandbox_testing" && latestVersion && <div className="command-section"><div className="button-row">{!sandboxPassed && role !== "operator" && <button className="button" disabled={busy} onClick={() => void onCommand("/v1/sandbox-runs", { listing_version_id: latestVersion.id, status: "succeeded", policy: { network: "denied", filesystem: "ephemeral" }, result: { contract_tests: "passed" } })}>记录沙箱通过</button>}{qualityScore === 0 && role !== "operator" && <button className="button" disabled={busy} onClick={() => void onCommand("/v1/quality-scores", { listing_version_id: latestVersion.id, score_bps: 9000, dimensions: { reliability: 9000, security: 9000 } })}>记录质量评分</button>}{sandboxPassed && qualityScore >= 8000 && <button className="button primary" disabled={busy} onClick={() => transition("limited_release")}>有限发布</button>}</div></div>}
    {listing.status === "limited_release" && <div className="command-section"><button className="button primary" disabled={busy} onClick={() => transition("published")}>正式发布</button></div>}
    {["published", "suspended"].includes(listing.status) && <div className="command-section"><form className="inline-form" onSubmit={event => submit(event, values => onCommand("/v1/marketplace-disputes", { listing_id: listing.id, claimant_type: "platform", claimant_id: "OpportunityOS", reason: values.get("reason") }))}><input name="reason" required placeholder="争议原因" /><button className="button" disabled={busy}>创建争议</button></form><form className="inline-form" onSubmit={event => submit(event, values => onCommand("/v1/takedowns", { listing_id: listing.id, reason: values.get("reason") }))}><input name="reason" required placeholder="下架原因" /><button className="button danger" disabled={busy}>请求下架</button></form></div>}
  </div>;
}
