"use client";

import {
  CoreApiClient,
  type Blueprint,
	type Capability,
  type Collection,
  type Incubation,
  type Opportunity,
	type Product,
	type ProductDetail,
	type Provider,
	type Session
} from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");

const nav = ["运行概览", "机会审核", "孵化项目", "业务蓝图", "产品发布", "工作流运行", "对账队列"].map(label => ({ label, href: "#workspace" }));
const statusLabel: Record<string, string> = {
  detected: "待补充证据", enriched: "待评分", scored: "待送审", under_review: "审核中",
  approved: "已批准", incubating: "孵化中", rejected: "已拒绝", archived: "已归档",
	draft: "草稿", researching: "研究中", validating: "验证中", analyzing: "分析中",
	ready: "待发布", published: "已发布", suspended: "已暂停", retired: "已退役", configuring: "配置中", launched: "已上线"
};

function tone(status: string): "green" | "amber" | "gray" {
	if (["approved", "incubating", "ready", "launched", "published"].includes(status)) return "green";
  if (["enriched", "scored", "under_review", "researching", "validating", "analyzing"].includes(status)) return "amber";
  return "gray";
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "操作未完成";
}

export default function Page() {
	const [session, setSession] = useState<Session | null>(null);
	const [authLoading, setAuthLoading] = useState(true);
	const [authBusy, setAuthBusy] = useState(false);
	const [authError, setAuthError] = useState<string | null>(null);
  const [opportunities, setOpportunities] = useState<Opportunity[]>([]);
  const [incubations, setIncubations] = useState<Incubation[]>([]);
  const [blueprints, setBlueprints] = useState<Blueprint[]>([]);
	const [capabilities, setCapabilities] = useState<Capability[]>([]);
	const [providers, setProviders] = useState<Provider[]>([]);
	const [products, setProducts] = useState<ProductDetail[]>([]);
  const [selectedID, setSelectedID] = useState<string | null>(null);
	const [selectedProductID, setSelectedProductID] = useState<string | null>(null);
	const [view, setView] = useState<"opportunities" | "incubations" | "blueprints" | "products">("opportunities");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
		const [opportunityResult, incubationResult, blueprintResult, capabilityResult, providerResult, productResult] = await Promise.all([
      api.get<Collection<Opportunity>>("/v1/opportunities"),
      api.get<Collection<Incubation>>("/v1/incubations"),
			api.get<Collection<Blueprint>>("/v1/blueprints"),
			api.get<Collection<Capability>>("/v1/capabilities"),
			api.get<Collection<Provider>>("/v1/providers"),
			api.get<Collection<Product>>("/v1/products")
    ]);
		const productDetails = await Promise.all(productResult.items.map(item => api.get<ProductDetail>(`/v1/products/${item.id}`)));
    setOpportunities(opportunityResult.items);
    setIncubations(incubationResult.items);
    setBlueprints(blueprintResult.items);
		setCapabilities(capabilityResult.items);
		setProviders(providerResult.items);
		setProducts(productDetails);
    setSelectedID(current => current ?? opportunityResult.items[0]?.id ?? null);
		setSelectedProductID(current => current && productDetails.some(item => item.id === current) ? current : productDetails[0]?.id ?? null);
  }, []);

  useEffect(() => {
		api.session().then(setSession).catch(() => setSession(null)).finally(() => setAuthLoading(false));
	}, []);

	useEffect(() => {
		if (!session) return;
		setLoading(true);
		reload().catch(error => setError(errorMessage(error))).finally(() => setLoading(false));
	}, [reload, session]);

	async function login(email: string, password: string) {
		setAuthBusy(true);
		setAuthError(null);
		try {
			setSession(await api.login(email, password));
		} catch (error) {
			setAuthError(errorMessage(error));
		} finally {
			setAuthBusy(false);
		}
	}

	async function logout() {
		await api.logout().catch(() => undefined);
		setSession(null);
		setOpportunities([]);
		setIncubations([]);
		setBlueprints([]);
		setCapabilities([]);
		setProviders([]);
		setProducts([]);
	}

  const selected = useMemo(() => opportunities.find(item => item.id === selectedID) ?? null, [opportunities, selectedID]);
	const selectedProduct = useMemo(() => products.find(item => item.id === selectedProductID) ?? null, [products, selectedProductID]);
  const pendingReviews = opportunities.filter(item => item.status === "under_review").length;
	if (authLoading) return <div className="auth-screen"><div className="loading">正在恢复会话...</div></div>;
	if (!session) return <LoginScreen title="运营控制台" error={authError} busy={authBusy} onLogin={login} />;

  async function execute(action: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      await reload();
    } catch (error) {
      setError(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function createOpportunity(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = new FormData(form);
    await execute(() => api.command("/v1/opportunities", {
      name: values.get("name"), description: values.get("description")
    }));
    form.reset();
  }

  async function addEvidence(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    const form = event.currentTarget;
    const values = new FormData(form);
    await execute(() => api.command(`/v1/opportunities/${selected.id}/evidence`, {
      kind: values.get("kind"), summary: values.get("summary"), confidence: Number(values.get("confidence"))
    }));
    form.reset();
  }

  async function scoreOpportunity(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    const values = new FormData(event.currentTarget);
    await execute(() => api.command(`/v1/opportunities/${selected.id}/score`, { score: Number(values.get("score")) }));
  }

  async function reviewOpportunity(rationale: string, decision: "approved" | "rejected") {
    if (!selected) return;
    await execute(() => api.command(`/v1/opportunities/${selected.id}/reviews`, { decision, rationale }));
  }

  async function createIncubation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    const form = event.currentTarget;
    const values = new FormData(form);
    await execute(() => api.command(`/v1/opportunities/${selected.id}/incubations`, { name: values.get("name") }));
    form.reset();
  }

  async function createBlueprint(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    const form = event.currentTarget;
    const values = new FormData(form);
    const name = String(values.get("name") ?? "");
    await execute(() => api.command(`/v1/opportunities/${selected.id}/blueprints`, {
      name,
      description: values.get("description"),
      value_proposition: values.get("value_proposition"),
      required_capabilities: [String(values.get("capability") ?? "Test Capability")],
      product_definitions: [{ name: `${name} Product` }],
      workflow_definitions: [{ name: `${name} Delivery` }],
      metering_definitions: [{ unit: "execution" }],
      pricing_definitions: [{ currency: "CNY" }],
      compliance_profile: { classification: "internal" },
      business_model: {}, target_market_definition: {}, revenue_model: {}, delivery_model: {}
    }));
    form.reset();
  }

	async function submitForm(event: FormEvent<HTMLFormElement>, action: (values: FormData) => Promise<unknown>) {
		event.preventDefault();
		const form = event.currentTarget;
		await execute(() => action(new FormData(form)));
		form.reset();
	}

	async function createProductVersion(event: FormEvent<HTMLFormElement>, productID: string) {
		event.preventDefault();
		const values = new FormData(event.currentTarget);
		let inputSchema: Record<string, unknown>;
		let outputSchema: Record<string, unknown>;
		let complianceProfile: Record<string, unknown>;
		try {
			inputSchema = JSON.parse(String(values.get("input_schema")));
			outputSchema = JSON.parse(String(values.get("output_schema")));
			complianceProfile = JSON.parse(String(values.get("compliance_profile")));
		} catch {
			setError("Schema 与合规配置必须是有效 JSON");
			return;
		}
		await execute(() => api.command(`/v1/products/${productID}/versions`, {
			input_schema: inputSchema, output_schema: outputSchema, form_schema: inputSchema,
			capability_ids: [String(values.get("capability_id"))],
			workflow: {
				name: values.get("workflow_name"), version: 1,
				nodes: [{ id: "start", type: "start" }, { id: "validate", type: "validate" }, { id: "execute", type: "realtime_call" }, { id: "meter", type: "meter" }, { id: "end", type: "end" }],
				edges: [{ from: "start", to: "validate" }, { from: "validate", to: "execute" }, { from: "execute", to: "meter" }, { from: "meter", to: "end" }]
			},
			metering: { name: values.get("meter_name"), unit: values.get("meter_unit"), field: values.get("meter_field"), version: 1 },
			price_book: { currency: values.get("currency"), version: 1, rules: [{ kind: "flat", flat_minor: Number(values.get("flat_minor")) }] },
			route_policy: { name: values.get("route_name"), strategy: values.get("route_strategy"), version: 1 },
			delivery_mode: values.get("delivery_mode"), compliance_profile: complianceProfile, growth_playbook: {}
		}));
	}

	return <PortalShell title="运营控制台" role="Operator Console" navigation={nav} tenantLabel={session.tenant_name} userLabel={`${session.display_name} · ${session.role}`} onLogout={logout}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics">
      <Metric label="机会总数" value={String(opportunities.length)} detail="PostgreSQL 实时数据" />
      <Metric label="待人工审核" value={String(pendingReviews)} detail="需要批准或拒绝" />
      <Metric label="孵化项目" value={String(incubations.length)} detail="版本化阶段门" />
			<Metric label="已发布产品" value={String(products.filter(item => item.status === "published").length)} detail={`${products.length} 个产品定义`} />
    </div>

    <section className="panel" id="workspace">
      <div className="tabs" role="tablist" aria-label="运营对象">
        <button className={`tab ${view === "opportunities" ? "active" : ""}`} onClick={() => setView("opportunities")}>机会</button>
        <button className={`tab ${view === "incubations" ? "active" : ""}`} onClick={() => setView("incubations")}>孵化项目</button>
        <button className={`tab ${view === "blueprints" ? "active" : ""}`} onClick={() => setView("blueprints")}>业务蓝图</button>
				<button className={`tab ${view === "products" ? "active" : ""}`} onClick={() => setView("products")}>产品工厂</button>
      </div>
      {loading ? <div className="loading">正在读取 Core API...</div> : view === "opportunities" ?
        <OpportunityWorkspace opportunities={opportunities} selected={selected} selectedID={selectedID} busy={busy}
          onSelect={setSelectedID} onCreate={createOpportunity} onEvidence={addEvidence} onScore={scoreOpportunity}
          onTransition={to => selected && execute(() => api.command(`/v1/opportunities/${selected.id}/transitions`, { to }))}
          onReview={reviewOpportunity} onIncubation={createIncubation} onBlueprint={createBlueprint} /> :
				view === "incubations" ? <ObjectRecords items={incubations.map(item => ({ id: item.id, title: item.name, status: item.status, detail: `Opportunity ${item.opportunity_id} · v${item.version}` }))} /> :
					view === "blueprints" ? <BlueprintRecords items={blueprints} busy={busy} onAdvance={(id, to) => execute(() => api.command(`/v1/blueprints/${id}/transitions`, { to }))} /> :
						<ProductFactory blueprints={blueprints} capabilities={capabilities} providers={providers} products={products} selected={selectedProduct} selectedID={selectedProductID} busy={busy} onSelect={setSelectedProductID}
							onCapability={event => submitForm(event, values => api.command("/v1/capabilities", { name: values.get("name"), description: values.get("description"), definition: {} }))}
							onProvider={event => submitForm(event, values => api.command("/v1/providers", { name: values.get("name") }))}
							onEndpoint={event => submitForm(event, values => api.command(`/v1/providers/${values.get("provider_id")}/endpoints`, { capability_id: values.get("capability_id"), adapter_type: values.get("adapter_type"), adapter_version: "v1" }))}
							onProduct={event => submitForm(event, values => api.command(`/v1/blueprints/${values.get("blueprint_id")}/products`, { name: values.get("name") }))}
							onVersion={createProductVersion}
							onSKU={event => selectedProduct && submitForm(event, values => api.command(`/v1/products/${selectedProduct.id}/skus`, { code: values.get("code"), name: values.get("name") }))}
							onSKUVersion={(skuID, productVersionID) => execute(() => api.command(`/v1/skus/${skuID}/versions`, { product_version_id: productVersionID, entitlements: {} }))}
							onPublish={(productID, productVersionID) => execute(() => api.command(`/v1/products/${productID}/publications`, { product_version_id: productVersionID }))} />}
    </section>
  </PortalShell>;
}

type WorkspaceProps = {
  opportunities: Opportunity[]; selected: Opportunity | null; selectedID: string | null; busy: boolean;
  onSelect: (id: string) => void; onCreate: (event: FormEvent<HTMLFormElement>) => void;
  onEvidence: (event: FormEvent<HTMLFormElement>) => void; onScore: (event: FormEvent<HTMLFormElement>) => void;
  onTransition: (to: string) => void; onReview: (rationale: string, decision: "approved" | "rejected") => void;
  onIncubation: (event: FormEvent<HTMLFormElement>) => void; onBlueprint: (event: FormEvent<HTMLFormElement>) => void;
};

function OpportunityWorkspace(props: WorkspaceProps) {
  return <div className="workspace-grid">
    <div>
      <form className="form-grid" onSubmit={props.onCreate}>
        <label className="field"><span>机会名称</span><input name="name" required maxLength={120} placeholder="输入可验证的商业机会" /></label>
        <label className="field"><span>初始描述</span><input name="description" maxLength={500} placeholder="问题、客户与预期结果" /></label>
        <div className="form-actions wide"><button className="button primary" disabled={props.busy}>创建机会</button></div>
      </form>
      <div className="panel-title"><h2>机会队列</h2><span className="muted">选择一项执行领域命令</span></div>
      <div style={{ overflowX: "auto" }}>
        <table className="data-grid"><thead><tr><th>机会</th><th>阶段</th><th>评分</th><th>证据</th><th>版本</th></tr></thead>
          <tbody>{props.opportunities.map(item => <tr key={item.id} className={`select-row ${item.id === props.selectedID ? "active" : ""}`} onClick={() => props.onSelect(item.id)}>
            <td><strong>{item.name}</strong><div className="muted">{item.description || item.id}</div></td>
            <td><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge></td>
            <td>{item.score}</td><td>{item.evidence?.length ?? 0}</td><td>v{item.version}</td>
          </tr>)}</tbody></table>
      </div>
    </div>
    <OpportunityDetail {...props} />
  </div>;
}

function OpportunityDetail(props: WorkspaceProps) {
  const item = props.selected;
  if (!item) return <div className="detail-panel"><div className="empty-feature">创建或选择一个机会</div></div>;
  return <div className="detail-panel">
    <div className="panel-title"><h2>{item.name}</h2><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge></div>
    <div className="detail-body">
      <div className="detail-meta"><div><span>评分</span><strong>{item.score} / 100</strong></div><div><span>版本</span><strong>v{item.version}</strong></div></div>
    </div>
    {item.status === "detected" && <Command title="录入验证证据"><form className="inline-form" onSubmit={props.onEvidence}><select name="kind" defaultValue="customer_interview"><option value="customer_interview">客户访谈</option><option value="market_signal">市场信号</option><option value="experiment">验证实验</option></select><textarea name="summary" required placeholder="记录可追溯的事实和结论" /><input name="confidence" type="number" min="0" max="100" defaultValue="80" required /><button className="button primary" disabled={props.busy}>保存证据</button></form></Command>}
    {item.status === "enriched" && <Command title="提交机会评分"><form className="inline-form" onSubmit={props.onScore}><input name="score" type="number" min="0" max="100" defaultValue="75" required /><button className="button primary" disabled={props.busy}>确认评分</button></form></Command>}
    {item.status === "scored" && <Command title="进入人工审核"><p className="muted">送审后只能由明确的审核命令批准或拒绝。</p><button className="button primary" disabled={props.busy} onClick={() => props.onTransition("under_review")}>提交审核</button></Command>}
    {item.status === "under_review" && <Command title="审核决策"><form className="inline-form" onSubmit={event => { event.preventDefault(); props.onReview(String(new FormData(event.currentTarget).get("rationale") ?? ""), "approved"); }}><textarea name="rationale" required placeholder="填写判断依据和风险说明" /><div className="button-row"><button className="button primary" disabled={props.busy}>批准</button><button type="button" className="button danger" disabled={props.busy} onClick={event => { const form = event.currentTarget.form; if (form?.reportValidity()) props.onReview(String(new FormData(form).get("rationale") ?? ""), "rejected"); }}>拒绝</button></div></form></Command>}
    {item.status === "approved" && <Command title="创建孵化项目"><form className="inline-form" onSubmit={props.onIncubation}><input name="name" required placeholder="孵化项目名称" /><button className="button primary" disabled={props.busy}>进入孵化</button></form></Command>}
    {item.status === "incubating" && <Command title="建立业务蓝图"><form className="inline-form" onSubmit={props.onBlueprint}><input name="name" required placeholder="蓝图名称" /><textarea name="value_proposition" required placeholder="价值主张" /><input name="capability" required defaultValue="Test Capability" /><input name="description" placeholder="蓝图说明" /><button className="button primary" disabled={props.busy}>创建蓝图草稿</button></form></Command>}
    {!!item.evidence?.length && <Command title="证据记录">{item.evidence.map(record => <div key={record.id} className="notice"><strong>{record.kind} · {record.confidence}%</strong><div>{record.summary}</div></div>)}</Command>}
  </div>;
}

function Command({ title, children }: { title: string; children: import("react").ReactNode }) {
  return <section className="command-section"><h3>{title}</h3>{children}</section>;
}

function ObjectRecords({ items }: { items: { id: string; title: string; status: string; detail: string }[] }) {
  if (!items.length) return <div className="empty-feature">暂无记录</div>;
  return <div className="record-list">{items.map(item => <div className="record-row" key={item.id}><div><strong>{item.title}</strong><small>{item.detail}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge><span className="muted">{item.id.slice(0, 8)}</span></div>)}</div>;
}

function BlueprintRecords({ items, busy, onAdvance }: { items: Blueprint[]; busy: boolean; onAdvance: (id: string, to: string) => void }) {
	const nextStatus: Record<string, string> = { draft: "analyzing", analyzing: "validating", validating: "approved", approved: "configuring", configuring: "ready", ready: "launched" };
	if (!items.length) return <div className="empty-feature">暂无记录</div>;
	return <div className="record-list">{items.map(item => <div className="record-row blueprint-row" key={item.id}><div><strong>{item.name}</strong><small>{item.value_proposition || "待补充价值主张"} · v{item.version}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge>{nextStatus[item.status] ? <button className="button" disabled={busy} onClick={() => onAdvance(item.id, nextStatus[item.status])}>推进至{statusLabel[nextStatus[item.status]] ?? nextStatus[item.status]}</button> : <span className="muted">{item.id.slice(0, 8)}</span>}</div>)}</div>;
}

type ProductFactoryProps = {
	blueprints: Blueprint[];
	capabilities: Capability[];
	providers: Provider[];
	products: ProductDetail[];
	selected: ProductDetail | null;
	selectedID: string | null;
	busy: boolean;
	onSelect: (id: string) => void;
	onCapability: (event: FormEvent<HTMLFormElement>) => void;
	onProvider: (event: FormEvent<HTMLFormElement>) => void;
	onEndpoint: (event: FormEvent<HTMLFormElement>) => void;
	onProduct: (event: FormEvent<HTMLFormElement>) => void;
	onVersion: (event: FormEvent<HTMLFormElement>, productID: string) => void;
	onSKU: (event: FormEvent<HTMLFormElement>) => void;
	onSKUVersion: (skuID: string, productVersionID: string) => void;
	onPublish: (productID: string, productVersionID: string) => void;
};

function ProductFactory(props: ProductFactoryProps) {
	const approvedBlueprints = props.blueprints.filter(item => ["approved", "configuring", "ready", "launched"].includes(item.status));
	const latestVersion = props.selected?.versions.length ? props.selected.versions.reduce((latest, item) => item.version > latest.version ? item : latest) : null;
	const releaseSKU = props.selected?.skus[0] ?? null;
	const releaseSKUVersion = latestVersion ? releaseSKU?.versions.find(item => item.product_version_id === latestVersion.id) : null;
	const published = latestVersion ? props.selected?.publications.some(item => item.product_version_id === latestVersion.id) : false;
	return <div className="factory-layout">
		<section className="factory-band">
			<div className="panel-title"><h2>资源与上游</h2><span className="muted">{props.capabilities.length} 项能力 · {props.providers.length} 个 Provider</span></div>
			<div className="factory-columns">
				<form className="inline-form" onSubmit={props.onCapability}><input name="name" required placeholder="能力名称" /><input name="description" placeholder="能力说明" /><button className="button" disabled={props.busy}>创建能力</button></form>
				<form className="inline-form" onSubmit={props.onProvider}><input name="name" required placeholder="Provider 名称" /><button className="button" disabled={props.busy}>注册 Provider</button></form>
				<form className="inline-form" onSubmit={props.onEndpoint}><select name="provider_id" required defaultValue=""><option value="" disabled>选择 Provider</option>{props.providers.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="capability_id" required defaultValue=""><option value="" disabled>选择能力</option>{props.capabilities.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="adapter_type" defaultValue="generic_http"><option value="generic_http">Generic HTTP</option><option value="generic_webhook">Generic Webhook</option><option value="manual_service">Manual Service</option><option value="provisioning">Provisioning</option></select><button className="button" disabled={props.busy || !props.providers.length || !props.capabilities.length}>添加 Endpoint</button></form>
			</div>
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>产品定义</h2><StatusBadge tone={approvedBlueprints.length ? "green" : "amber"}>{approvedBlueprints.length} 个可用蓝图</StatusBadge></div>
			<form className="form-grid" onSubmit={props.onProduct}><label className="field"><span>业务蓝图</span><select name="blueprint_id" required defaultValue=""><option value="" disabled>选择已批准蓝图</option>{approvedBlueprints.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label className="field"><span>产品名称</span><input name="name" required placeholder="输入产品名称" /></label><div className="form-actions wide"><button className="button primary" disabled={props.busy || !approvedBlueprints.length}>创建产品草稿</button></div></form>
			<div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>产品</th><th>状态</th><th>版本</th><th>SKU</th></tr></thead><tbody>{props.products.map(item => <tr key={item.id} className={`select-row ${item.id === props.selectedID ? "active" : ""}`} onClick={() => props.onSelect(item.id)}><td><strong>{item.name}</strong><div className="muted">{item.id.slice(0, 8)}</div></td><td><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge></td><td>{item.versions.length}</td><td>{item.skus.length}</td></tr>)}</tbody></table></div>
		</section>

		{props.selected && <section className="factory-band">
			<div className="panel-title"><h2>{props.selected.name}</h2><StatusBadge tone={tone(props.selected.status)}>{statusLabel[props.selected.status] ?? props.selected.status}</StatusBadge></div>
			<form className="form-grid version-form" onSubmit={event => props.onVersion(event, props.selected!.id)}>
				<label className="field"><span>输入 Schema</span><textarea name="input_schema" required defaultValue={'{"type":"object","properties":{"input":{"type":"string"}},"required":["input"]}'} /></label>
				<label className="field"><span>输出 Schema</span><textarea name="output_schema" required defaultValue={'{"type":"object","properties":{"result":{"type":"string"},"units":{"type":"integer"}}}'} /></label>
				<label className="field"><span>执行能力</span><select name="capability_id" required defaultValue=""><option value="" disabled>选择能力</option>{props.capabilities.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
				<label className="field"><span>工作流名称</span><input name="workflow_name" required /></label>
				<label className="field"><span>计量名称</span><input name="meter_name" required /></label>
				<label className="field"><span>计量单位</span><input name="meter_unit" required /></label>
				<label className="field"><span>计量字段</span><input name="meter_field" required /></label>
				<label className="field"><span>币种</span><input name="currency" required minLength={3} maxLength={3} /></label>
				<label className="field"><span>固定价格（最小货币单位）</span><input name="flat_minor" type="number" min="0" required /></label>
				<label className="field"><span>路由名称</span><input name="route_name" required /></label>
				<label className="field"><span>路由策略</span><select name="route_strategy" defaultValue="priority"><option value="priority">优先级</option><option value="lowest_cost">最低成本</option></select></label>
				<label className="field"><span>交付模式</span><select name="delivery_mode" defaultValue="workflow"><option value="workflow">工作流</option><option value="realtime">实时</option><option value="async">异步</option><option value="provisioning">资源开通</option><option value="manual">人工服务</option></select></label>
				<label className="field wide"><span>合规配置</span><textarea name="compliance_profile" required defaultValue={'{"classification":"internal"}'} /></label>
				<div className="form-actions wide"><button className="button primary" disabled={props.busy || !props.capabilities.length}>创建不可变版本</button></div>
			</form>
			{latestVersion && <div className="release-strip"><div><span>最新产品版本</span><strong>v{latestVersion.version}</strong></div><div><span>工作流</span><strong>{String(latestVersion.workflow.name ?? "-")}</strong></div><div><span>交付</span><strong>{latestVersion.delivery_mode}</strong></div></div>}
			{latestVersion && !releaseSKU && <form className="inline-form release-form" onSubmit={props.onSKU}><input name="code" required placeholder="SKU 编码" /><input name="name" required placeholder="SKU 名称" /><button className="button primary" disabled={props.busy}>创建 SKU</button></form>}
			{latestVersion && releaseSKU && !releaseSKUVersion && <div className="release-action"><span>{releaseSKU.code} 尚未绑定 v{latestVersion.version}</span><button className="button primary" disabled={props.busy} onClick={() => props.onSKUVersion(releaseSKU!.id, latestVersion.id)}>创建 SKU 版本</button></div>}
			{latestVersion && releaseSKUVersion && !published && <div className="release-action"><span>{releaseSKU?.code ?? "SKU"} v{releaseSKUVersion.version} 已绑定全部版本</span><button className="button primary" disabled={props.busy} onClick={() => props.onPublish(props.selected!.id, latestVersion.id)}>发布产品</button></div>}
			{published && <div className="notice">当前产品版本已发布</div>}
		</section>}
	</div>;
}
