"use client";

import {
  CoreApiClient,
  type Blueprint,
	type Capability,
  type Collection,
	type FinanceOverview,
	type GrowthOverview,
  type Incubation,
  type Opportunity,
	type Product,
	type ProductDetail,
	type Provider,
	type Quote,
	type Order,
	type Session
} from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");

const emptyFinance: FinanceOverview = {
	wallets: [], accounts: [], transactions: [], holds: [], refunds: [], commissions: [],
	provider_payables: [], settlements: [], reconciliation_runs: []
};

const emptyGrowth: GrowthOverview = {
	segments: [], icps: [], leads: [], evidence: [], contacts: [], proof_templates: [], proof_requests: [],
	proof_instances: [], campaigns: [], suppressions: [], outreach: [], conversations: [], deals: [], experiments: []
};

const nav = ["运行概览", "机会审核", "孵化项目", "业务蓝图", "产品发布", "增长销售", "交易执行", "财务对账"].map(label => ({ label, href: "#workspace" }));
const statusLabel: Record<string, string> = {
  detected: "待补充证据", enriched: "待评分", scored: "待送审", under_review: "审核中",
  approved: "已批准", incubating: "孵化中", rejected: "已拒绝", archived: "已归档",
	draft: "草稿", researching: "研究中", validating: "验证中", analyzing: "分析中",
	ready: "待发布", published: "已发布", suspended: "已暂停", retired: "已退役", configuring: "配置中", launched: "已上线",
	sent: "已发送", accepted: "已接受", expired: "已过期", cancelled: "已取消",
	created: "已创建", awaiting_payment: "待支付", paid: "已支付", provisioning: "履约中", active: "已生效", completed: "已完成",
	reserved: "已预留", queued: "已排队", submitted: "已提交", processing: "处理中", succeeded: "已成功", failed: "失败", reconciling: "对账中", settled: "已结算",
	in_progress: "进行中", waiting: "等待中", calculated: "已计算", pending: "待生效", posted: "已入账", reversed: "已冲退",
	open: "待结算", partially_settled: "部分结算", matched: "已匹配", discrepancy: "有差异", released: "已释放", captured: "已扣取", partially_captured: "部分扣取",
	discovered: "待丰富", qualified: "已认定", proof_requested: "Proof 待生成", proof_ready: "Proof 就绪", approved_for_outreach: "可计划触达",
	contacted: "已联系", replied: "已回复", meeting: "会谈中", proposal: "提案中", won: "已赢单", lost: "已流失", suppressed: "禁止联系",
	pending_approval: "待审批", paused: "已暂停", planned: "已计划", blocked: "已阻止", running: "运行中"
};

function tone(status: string): "green" | "amber" | "gray" {
	if (["approved", "incubating", "ready", "launched", "published", "accepted", "paid", "active", "completed", "succeeded", "settled", "calculated", "posted", "matched", "released", "captured"].includes(status)) return "green";
	if (["enriched", "scored", "under_review", "researching", "validating", "analyzing", "sent", "awaiting_payment", "provisioning", "reserved", "queued", "submitted", "processing", "reconciling", "in_progress", "waiting", "pending", "open", "partially_settled", "partially_captured", "discrepancy"].includes(status)) return "amber";
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
	const [quotes, setQuotes] = useState<Quote[]>([]);
	const [orders, setOrders] = useState<Order[]>([]);
	const [finance, setFinance] = useState<FinanceOverview>(emptyFinance);
	const [growth, setGrowth] = useState<GrowthOverview>(emptyGrowth);
  const [selectedID, setSelectedID] = useState<string | null>(null);
	const [selectedProductID, setSelectedProductID] = useState<string | null>(null);
	const [selectedOrderID, setSelectedOrderID] = useState<string | null>(null);
	const [view, setView] = useState<"opportunities" | "incubations" | "blueprints" | "products" | "growth" | "transactions" | "finance">("opportunities");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
		const [opportunityResult, incubationResult, blueprintResult, capabilityResult, providerResult, productResult, quoteResult, orderResult, financeResult, growthResult] = await Promise.all([
      api.get<Collection<Opportunity>>("/v1/opportunities"),
      api.get<Collection<Incubation>>("/v1/incubations"),
			api.get<Collection<Blueprint>>("/v1/blueprints"),
			api.get<Collection<Capability>>("/v1/capabilities"),
			api.get<Collection<Provider>>("/v1/providers"),
			api.get<Collection<Product>>("/v1/products"),
			api.get<Collection<Quote>>("/v1/quotes"),
			api.get<Collection<Order>>("/v1/orders"),
			api.get<FinanceOverview>("/v1/finance"),
			api.get<GrowthOverview>("/v1/growth")
    ]);
		const productDetails = await Promise.all(productResult.items.map(item => api.get<ProductDetail>(`/v1/products/${item.id}`)));
    setOpportunities(opportunityResult.items);
    setIncubations(incubationResult.items);
    setBlueprints(blueprintResult.items);
		setCapabilities(capabilityResult.items);
		setProviders(providerResult.items);
		setProducts(productDetails);
		setQuotes(quoteResult.items);
		setOrders(orderResult.items);
		setFinance(financeResult);
		setGrowth(growthResult);
    setSelectedID(current => current ?? opportunityResult.items[0]?.id ?? null);
		setSelectedProductID(current => current && productDetails.some(item => item.id === current) ? current : productDetails[0]?.id ?? null);
		setSelectedOrderID(current => current && orderResult.items.some(item => item.id === current) ? current : orderResult.items[0]?.id ?? null);
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
		setQuotes([]);
		setOrders([]);
		setFinance(emptyFinance);
		setGrowth(emptyGrowth);
	}

  const selected = useMemo(() => opportunities.find(item => item.id === selectedID) ?? null, [opportunities, selectedID]);
	const selectedProduct = useMemo(() => products.find(item => item.id === selectedProductID) ?? null, [products, selectedProductID]);
	const selectedOrder = useMemo(() => orders.find(item => item.id === selectedOrderID) ?? null, [orders, selectedOrderID]);
	const orderableSKUs = useMemo(() => products.flatMap(product => {
		const publishedVersions = new Set(product.publications.filter(item => item.status === "published").map(item => item.product_version_id));
		return product.skus.flatMap(sku => sku.versions.filter(version => publishedVersions.has(version.product_version_id)).map(version => ({ id: version.id, label: `${product.name} · ${sku.code} v${version.version}` })));
	}), [products]);
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

	async function createQuote(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		const form = event.currentTarget;
		const values = new FormData(form);
		let input: Record<string, unknown>;
		try {
			input = JSON.parse(String(values.get("input") ?? "{}"));
		} catch {
			setError("订单输入必须是有效 JSON");
			return;
		}
		const validUntil = new Date(Date.now() + Number(values.get("valid_days") ?? 7) * 86400000).toISOString();
		await execute(() => api.command("/v1/quotes", {
			deal_id: values.get("deal_id"), customer_id: values.get("customer_id"), currency: values.get("currency"), valid_until: validUntil,
			items: [{ sku_version_id: values.get("sku_version_id"), quantity: Number(values.get("quantity")), input }]
		}));
		form.reset();
	}

	return <PortalShell title="运营控制台" role="Operator Console" navigation={nav} tenantLabel={session.tenant_name} userLabel={`${session.display_name} · ${session.role}`} onLogout={logout}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics">
      <Metric label="机会总数" value={String(opportunities.length)} detail="PostgreSQL 实时数据" />
      <Metric label="待人工审核" value={String(pendingReviews)} detail="需要批准或拒绝" />
      <Metric label="孵化项目" value={String(incubations.length)} detail="版本化阶段门" />
			<Metric label="已发布产品" value={String(products.filter(item => item.status === "published").length)} detail={`${products.length} 个产品定义`} />
			<Metric label="增长管道" value={String(growth.leads.length)} detail={`${growth.deals.filter(item => ["open", "proposal"].includes(item.status)).length} 个进行中 Deal`} />
    </div>

    <section className="panel" id="workspace">
      <div className="tabs" role="tablist" aria-label="运营对象">
        <button className={`tab ${view === "opportunities" ? "active" : ""}`} onClick={() => setView("opportunities")}>机会</button>
        <button className={`tab ${view === "incubations" ? "active" : ""}`} onClick={() => setView("incubations")}>孵化项目</button>
        <button className={`tab ${view === "blueprints" ? "active" : ""}`} onClick={() => setView("blueprints")}>业务蓝图</button>
				<button className={`tab ${view === "products" ? "active" : ""}`} onClick={() => setView("products")}>产品工厂</button>
				<button className={`tab ${view === "growth" ? "active" : ""}`} onClick={() => setView("growth")}>增长销售</button>
				<button className={`tab ${view === "transactions" ? "active" : ""}`} onClick={() => setView("transactions")}>交易执行</button>
				<button className={`tab ${view === "finance" ? "active" : ""}`} onClick={() => setView("finance")}>财务</button>
      </div>
      {loading ? <div className="loading">正在读取 Core API...</div> : view === "opportunities" ?
        <OpportunityWorkspace opportunities={opportunities} selected={selected} selectedID={selectedID} busy={busy}
          onSelect={setSelectedID} onCreate={createOpportunity} onEvidence={addEvidence} onScore={scoreOpportunity}
          onTransition={to => selected && execute(() => api.command(`/v1/opportunities/${selected.id}/transitions`, { to }))}
          onReview={reviewOpportunity} onIncubation={createIncubation} onBlueprint={createBlueprint} /> :
				view === "incubations" ? <ObjectRecords items={incubations.map(item => ({ id: item.id, title: item.name, status: item.status, detail: `Opportunity ${item.opportunity_id} · v${item.version}` }))} /> :
					view === "blueprints" ? <BlueprintRecords items={blueprints} busy={busy} onAdvance={(id, to) => execute(() => api.command(`/v1/blueprints/${id}/transitions`, { to }))} /> :
						view === "products" ? <ProductFactory blueprints={blueprints} capabilities={capabilities} providers={providers} products={products} selected={selectedProduct} selectedID={selectedProductID} busy={busy} onSelect={setSelectedProductID}
							onCapability={event => submitForm(event, values => api.command("/v1/capabilities", { name: values.get("name"), description: values.get("description"), definition: {} }))}
							onProvider={event => submitForm(event, values => api.command("/v1/providers", { name: values.get("name") }))}
							onEndpoint={event => submitForm(event, values => api.command(`/v1/providers/${values.get("provider_id")}/endpoints`, { capability_id: values.get("capability_id"), adapter_type: values.get("adapter_type"), adapter_version: "v1" }))}
							onProduct={event => submitForm(event, values => api.command(`/v1/blueprints/${values.get("blueprint_id")}/products`, { name: values.get("name") }))}
							onVersion={createProductVersion}
							onSKU={event => selectedProduct && submitForm(event, values => api.command(`/v1/products/${selectedProduct.id}/skus`, { code: values.get("code"), name: values.get("name") }))}
							onSKUVersion={(skuID, productVersionID) => execute(() => api.command(`/v1/skus/${skuID}/versions`, { product_version_id: productVersionID, entitlements: {} }))}
							onPublish={(productID, productVersionID) => execute(() => api.command(`/v1/products/${productID}/publications`, { product_version_id: productVersionID }))} /> :
							view === "growth" ? <GrowthWorkspace growth={growth} products={products} orderableSKUs={orderableSKUs} busy={busy} canWrite={["admin", "operator"].includes(session.role)} canReview={["admin", "reviewer"].includes(session.role)} onCommand={(path, body) => execute(() => api.command(path, body))} /> :
							view === "transactions" ? <TransactionWorkspace quotes={quotes} orders={orders} holds={finance.holds} selected={selectedOrder} selectedID={selectedOrderID} orderableSKUs={orderableSKUs} providers={providers} busy={busy} onSelect={setSelectedOrderID} onQuote={createQuote}
								onQuoteTransition={(id, to) => execute(() => api.command(`/v1/quotes/${id}/transitions`, { to }))}
								onOrder={(quoteVersionID) => execute(() => api.command("/v1/orders", { quote_version_id: quoteVersionID }))}
								onOrderTransition={(id, to) => execute(() => api.command(`/v1/orders/${id}/transitions`, { to }))}
								onExecution={(id, to, providerEndpointID) => execute(() => api.command(`/v1/executions/${id}/transitions`, { to, provider_endpoint_id: providerEndpointID || undefined }))}
								onDelivery={(id, to) => execute(() => api.command(`/v1/deliveries/${id}/transitions`, { to }))}
								onUsage={(id, quantity) => execute(() => api.command(`/v1/executions/${id}/usage`, { quantity, occurred_at: new Date().toISOString() }))}
								onCost={(id, providerEndpointID, currency, amountMinor) => execute(() => api.command(`/v1/executions/${id}/provider-costs`, { provider_endpoint_id: providerEndpointID, currency, amount_minor: amountMinor }))}
								onCharge={id => execute(() => api.command(`/v1/executions/${id}/customer-charges`, {}))} /> :
								<FinanceWorkspace finance={finance} orders={orders} busy={busy} canAdjust={session.role === "admin"} onCommand={(path, body) => execute(() => api.command(path, body))} />}
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

type GrowthWorkspaceProps = {
	growth: GrowthOverview;
	products: ProductDetail[];
	orderableSKUs: { id: string; label: string }[];
	busy: boolean;
	canWrite: boolean;
	canReview: boolean;
	onCommand: (path: string, body: unknown) => Promise<unknown>;
};

function GrowthWorkspace(props: GrowthWorkspaceProps) {
	const [formError, setFormError] = useState<string | null>(null);
	const qualifiedLeads = props.growth.leads.filter(item => !["discovered", "enriched", "won", "lost", "suppressed"].includes(item.status));
	const outreachLeads = props.growth.leads.filter(item => ["approved_for_outreach", "contacted", "replied"].includes(item.status));
	const openDeals = props.growth.deals.filter(item => ["open", "proposal"].includes(item.status));
	const workflowIDs = Array.from(new Set(props.products.flatMap(product => product.versions.map(version => String(version.workflow.id ?? ""))).filter(Boolean)));
	const leadNext: Record<string, string> = { enriched: "qualified", proof_ready: "approved_for_outreach", replied: "meeting", meeting: "proposal" };

	async function submit(event: FormEvent<HTMLFormElement>, path: string, build: (values: FormData) => unknown) {
		event.preventDefault();
		const form = event.currentTarget;
		setFormError(null);
		try {
			await props.onCommand(path, build(new FormData(form)));
			form.reset();
		} catch (error) {
			setFormError(errorMessage(error));
		}
	}

	function objectValue(values: FormData, name: string): Record<string, unknown> {
		const value = JSON.parse(String(values.get(name) ?? "{}"));
		if (!value || Array.isArray(value) || typeof value !== "object") throw new Error(`${name} 必须是 JSON 对象`);
		return value as Record<string, unknown>;
	}

	function entityOptions() {
		return [
			...props.growth.segments.map(item => ({ value: `market_segment:${item.id}`, label: `市场细分 · ${item.name}` })),
			...props.growth.leads.map(item => ({ value: `lead:${item.id}`, label: `潜客 · ${item.name}` })),
			...props.growth.campaigns.map(item => ({ value: `campaign:${item.id}`, label: `Campaign · ${item.name}` })),
			...props.growth.deals.map(item => ({ value: `deal:${item.id}`, label: `Deal · ${item.name}` }))
		];
	}

	return <div className="factory-layout growth-layout">
		{formError && <div className="notice error">{formError}</div>}
		<section className="factory-band">
			<div className="panel-title"><h2>市场细分与潜客</h2><span className="muted">{props.growth.segments.length} 个细分 · {props.growth.leads.length} 条潜客</span></div>
			<div className="factory-columns growth-controls">
				<form className="inline-form" onSubmit={event => submit(event, "/v1/market-segments", values => ({ name: values.get("name"), definition: objectValue(values, "definition") }))}>
					<input name="name" required placeholder="市场细分名称" /><textarea name="definition" defaultValue={'{"criteria":[]}'} required /><button className="button primary" disabled={props.busy || !props.canWrite}>创建细分</button>
				</form>
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void submit(event, `/v1/market-segments/${values.get("segment_id")}/icps`, data => ({ name: data.get("name"), definition: objectValue(data, "definition") })); }}>
					<select name="segment_id" required defaultValue=""><option value="" disabled>选择市场细分</option>{props.growth.segments.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="name" required placeholder="ICP 名称" /><textarea name="definition" defaultValue={'{"signals":[],"constraints":[]}'} required /><button className="button" disabled={props.busy || !props.canWrite || !props.growth.segments.length}>创建 ICP</button>
				</form>
				<form className="inline-form" onSubmit={event => submit(event, "/v1/leads", values => ({ market_segment_id: values.get("segment_id"), icp_definition_id: values.get("icp_id"), name: values.get("name") }))}>
					<select name="segment_id" required defaultValue=""><option value="" disabled>市场细分</option>{props.growth.segments.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="icp_id" defaultValue=""><option value="">不绑定 ICP</option>{props.growth.icps.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="name" required placeholder="潜客名称" /><button className="button primary" disabled={props.busy || !props.canWrite || !props.growth.segments.length}>创建潜客</button>
				</form>
			</div>
			<div className="growth-two-column">
				<div>
					<h3>潜客队列</h3>
					<div className="transaction-rows">{props.growth.leads.length ? props.growth.leads.map(item => <div className="transaction-row" key={item.id}><div><strong>{item.name}</strong><small>评分 {item.score} · v{item.version}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge>{leadNext[item.status] ? <button className="button" disabled={props.busy || !props.canWrite} onClick={() => void props.onCommand(`/v1/leads/${item.id}/transitions`, { to: leadNext[item.status] })}>推进</button> : <span className="muted">{item.id.slice(0, 8)}</span>}</div>) : <div className="empty-feature">暂无潜客</div>}</div>
				</div>
				<div className="growth-stack">
					<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void submit(event, `/v1/leads/${values.get("lead_id")}/evidence`, data => ({ kind: data.get("kind"), summary: data.get("summary"), confidence: Number(data.get("confidence")), source_ref: data.get("source_ref") })); }}>
						<select name="lead_id" required defaultValue=""><option value="" disabled>选择潜客</option>{props.growth.leads.filter(item => !["won", "lost", "suppressed"].includes(item.status)).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="kind" required placeholder="证据类型" /><input name="summary" required placeholder="证据摘要" /><input name="confidence" type="number" min="0" max="100" defaultValue="80" required /><input name="source_ref" placeholder="来源引用" /><button className="button" disabled={props.busy || !props.canWrite}>添加证据</button>
					</form>
					<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void submit(event, `/v1/leads/${values.get("lead_id")}/contacts`, data => ({ channel: data.get("channel"), value: data.get("value"), consent_status: data.get("consent_status"), source_type: "manual", source_ref: "operator", evidence: {} })); }}>
						<select name="lead_id" required defaultValue=""><option value="" disabled>选择潜客</option>{props.growth.leads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="channel" defaultValue="email"><option value="email">Email</option><option value="phone">Phone</option><option value="web">Web</option><option value="custom">Custom</option></select><input name="value" required placeholder="联系值" /><select name="consent_status" defaultValue="unknown"><option value="unknown">未知</option><option value="opted_in">已同意</option><option value="opted_out">已退出</option></select><button className="button" disabled={props.busy || !props.canWrite}>记录联系人</button>
					</form>
				</div>
			</div>
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>Proof 管理</h2><span className="muted">Schema + 工作流 + 人工审核</span></div>
			<div className="factory-columns growth-controls">
				<form className="inline-form" onSubmit={event => submit(event, "/v1/proof-templates", values => ({ name: values.get("name"), proof_type: values.get("proof_type"), workflow_version_id: values.get("workflow_id"), input_schema: objectValue(values, "input_schema"), output_schema: objectValue(values, "output_schema"), access_policy: { scope: "tenant" }, retention_days: Number(values.get("retention_days")) }))}>
					<input name="name" required placeholder="Proof 模板名称" /><select name="proof_type" defaultValue="analysis"><option value="analysis">分析</option><option value="report">报告</option><option value="sample">样本</option><option value="comparison">对比</option><option value="prototype">原型</option><option value="custom">自定义</option></select><select name="workflow_id" required defaultValue=""><option value="" disabled>选择工作流版本</option>{workflowIDs.map(id => <option key={id} value={id}>{id.slice(0, 12)}</option>)}</select><textarea name="input_schema" defaultValue={'{"type":"object","properties":{}}'} required /><textarea name="output_schema" defaultValue={'{"type":"object","properties":{}}'} required /><input name="retention_days" type="number" min="1" defaultValue="30" required /><button className="button primary" disabled={props.busy || !props.canWrite || !workflowIDs.length}>创建模板</button>
				</form>
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void submit(event, `/v1/leads/${values.get("lead_id")}/proof-requests`, data => ({ template_id: data.get("template_id"), deal_id: data.get("deal_id"), input: objectValue(data, "input") })); }}>
					<select name="lead_id" required defaultValue=""><option value="" disabled>已认定潜客</option>{qualifiedLeads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="template_id" required defaultValue=""><option value="" disabled>Proof 模板</option>{props.growth.proof_templates.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="deal_id" defaultValue=""><option value="">不绑定 Deal</option>{openDeals.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><textarea name="input" defaultValue="{}" required /><button className="button" disabled={props.busy || !props.canWrite || !qualifiedLeads.length || !props.growth.proof_templates.length}>请求 Proof</button>
				</form>
			</div>
			<div className="transaction-rows">{props.growth.proof_requests.map(request => { const instance = props.growth.proof_instances.find(item => item.proof_request_id === request.id); return <div className="transaction-row" key={request.id}><div><strong>{props.growth.leads.find(item => item.id === request.lead_id)?.name ?? request.lead_id.slice(0, 8)}</strong><small>{props.growth.proof_templates.find(item => item.id === request.template_id)?.name ?? "Proof"} · 到期 {new Date(request.expires_at).toLocaleDateString("zh-CN")}</small></div><StatusBadge tone={tone(request.status)}>{statusLabel[request.status] ?? request.status}</StatusBadge><div className="button-row">{request.status === "requested" && <button className="button" disabled={props.busy || !props.canWrite} onClick={() => void props.onCommand(`/v1/proof-requests/${request.id}/generate`, { result: { summary: "Test Proof Artifact" }, artifact_ref: `proof://${request.id}` })}>生成 Proof</button>}{request.status === "review" && instance && props.canReview && <><button className="button primary" disabled={props.busy} onClick={() => void props.onCommand(`/v1/proof-requests/${request.id}/reviews`, { decision: "approved", rationale: "Proof output verified" })}>批准</button><button className="button" disabled={props.busy} onClick={() => void props.onCommand(`/v1/proof-requests/${request.id}/reviews`, { decision: "rejected", rationale: "Proof requires revision" })}>拒绝</button></>}</div></div>; })}</div>
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>Campaign 与触达计划</h2><StatusBadge tone="amber">外部发送关闭</StatusBadge></div>
			<div className="factory-columns growth-controls">
				<form className="inline-form" onSubmit={event => submit(event, "/v1/campaigns", values => ({ market_segment_id: values.get("segment_id"), name: values.get("name"), channel: values.get("channel"), purpose: values.get("purpose") }))}>
					<select name="segment_id" defaultValue=""><option value="">全部细分</option>{props.growth.segments.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="name" required placeholder="Campaign 名称" /><select name="channel" defaultValue="email"><option value="email">Email</option><option value="phone">Phone</option><option value="web">Web</option><option value="custom">Custom</option></select><input name="purpose" required placeholder="触达目的" /><button className="button primary" disabled={props.busy || !props.canWrite}>创建 Campaign</button>
				</form>
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void submit(event, `/v1/campaigns/${values.get("campaign_id")}/steps`, data => ({ position: Number(data.get("position")), kind: data.get("kind"), definition: objectValue(data, "definition") })); }}>
					<select name="campaign_id" required defaultValue=""><option value="" disabled>草稿 Campaign</option>{props.growth.campaigns.filter(item => item.status === "draft").map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="position" type="number" min="1" defaultValue="1" required /><select name="kind" defaultValue="message"><option value="message">消息计划</option><option value="wait">等待</option><option value="condition">条件</option><option value="proof_request">Proof 请求</option><option value="manual_task">人工任务</option></select><textarea name="definition" defaultValue={'{"template":"tenant-controlled"}'} required /><button className="button" disabled={props.busy || !props.canWrite}>添加步骤</button>
				</form>
			</div>
			<div className="transaction-rows">{props.growth.campaigns.map(item => <div className="transaction-row" key={item.id}><div><strong>{item.name}</strong><small>{item.channel} · {item.steps.length} 步 · v{item.version}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge><div className="button-row">{item.status === "draft" && <button className="button" disabled={props.busy || !props.canWrite || !item.steps.length} onClick={() => void props.onCommand(`/v1/campaigns/${item.id}/transitions`, { to: "pending_approval" })}>提交审批</button>}{item.status === "pending_approval" && props.canReview && <button className="button primary" disabled={props.busy} onClick={() => void props.onCommand(`/v1/campaigns/${item.id}/reviews`, { decision: "approved", rationale: "Campaign scope and controls verified" })}>批准</button>}{item.status === "approved" && <button className="button primary" disabled={props.busy || !props.canWrite} onClick={() => void props.onCommand(`/v1/campaigns/${item.id}/transitions`, { to: "active" })}>启用计划</button>}</div></div>)}</div>
			<div className="growth-two-column">
				<form className="inline-form" onSubmit={event => submit(event, "/v1/suppressions", values => ({ lead_id: values.get("lead_id"), contact_id: values.get("contact_id"), channel: values.get("channel"), reason: values.get("reason"), source_ref: "operator" }))}>
					<select name="lead_id" defaultValue=""><option value="">不按潜客抑制</option>{props.growth.leads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="contact_id" defaultValue=""><option value="">不按联系人抑制</option>{props.growth.contacts.filter(item => item.status === "active").map(item => <option key={item.id} value={item.id}>{item.value}</option>)}</select><select name="channel" defaultValue="all"><option value="all">全部渠道</option><option value="email">Email</option><option value="phone">Phone</option><option value="web">Web</option><option value="custom">Custom</option></select><select name="reason" defaultValue="do_not_contact"><option value="do_not_contact">禁止联系</option><option value="opt_out">主动退出</option><option value="bounce">退信</option><option value="complaint">投诉</option><option value="risk">风险</option></select><button className="button" disabled={props.busy || !props.canWrite}>加入抑制</button>
				</form>
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); const campaign = props.growth.campaigns.find(item => item.id === values.get("campaign_id")); void submit(event, `/v1/campaigns/${campaign?.id}/outreach`, data => ({ lead_id: data.get("lead_id"), step_id: data.get("step_id"), contact_id: data.get("contact_id"), content: { subject: data.get("subject"), body: data.get("body") } })); }}>
					<select name="campaign_id" required defaultValue=""><option value="" disabled>已启用 Campaign</option>{props.growth.campaigns.filter(item => item.status === "active").map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="step_id" required defaultValue=""><option value="" disabled>消息步骤</option>{props.growth.campaigns.flatMap(item => item.steps.filter(step => step.kind === "message")).map(item => <option key={item.id} value={item.id}>{item.campaign_id.slice(0, 8)} · 第 {item.position} 步</option>)}</select><select name="lead_id" required defaultValue=""><option value="" disabled>可计划潜客</option>{outreachLeads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="contact_id" defaultValue=""><option value="">无需联系人</option>{props.growth.contacts.filter(item => item.status === "active").map(item => <option key={item.id} value={item.id}>{item.value}</option>)}</select><input name="subject" required placeholder="主题" /><textarea name="body" required placeholder="计划内容" /><button className="button primary" disabled={props.busy || !props.canWrite}>检查并计划</button>
				</form>
			</div>
			<div className="transaction-rows">{props.growth.outreach.map(item => <div className="transaction-row" key={item.id}><div><strong>{props.growth.leads.find(lead => lead.id === item.lead_id)?.name ?? item.lead_id.slice(0, 8)}</strong><small>{item.block_reason || "已通过抑制与额度检查"}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge>{item.status === "planned" ? <button className="button" disabled={props.busy || !props.canWrite} onClick={() => void props.onCommand(`/v1/outreach/${item.id}/transitions`, { to: "cancelled" })}>取消计划</button> : <span className="muted">不执行发送</span>}</div>)}</div>
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>Conversation、Deal 与实验</h2><span className="muted">Deal 绑定既有 Quote，Order 保持独立</span></div>
			<div className="factory-columns growth-controls">
				<form className="inline-form" onSubmit={event => submit(event, "/v1/deals", values => ({ lead_id: values.get("lead_id"), name: values.get("name"), customer_id: values.get("customer_id"), currency: values.get("currency"), value_minor: Number(values.get("value_minor")) }))}>
					<select name="lead_id" required defaultValue=""><option value="" disabled>选择已认定潜客</option>{qualifiedLeads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><input name="name" required placeholder="Deal 名称" /><input name="customer_id" required defaultValue="Test Customer" /><input name="currency" required minLength={3} maxLength={3} defaultValue="CNY" /><input name="value_minor" type="number" min="0" defaultValue="10000" required /><button className="button primary" disabled={props.busy || !props.canWrite || !qualifiedLeads.length}>创建 Deal</button>
				</form>
				<form className="inline-form" onSubmit={event => submit(event, "/v1/conversations", values => ({ lead_id: values.get("lead_id"), deal_id: values.get("deal_id"), channel: values.get("channel") }))}>
					<select name="lead_id" required defaultValue=""><option value="" disabled>选择潜客</option>{props.growth.leads.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="deal_id" defaultValue=""><option value="">不绑定 Deal</option>{openDeals.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="channel" defaultValue="email"><option value="email">Email</option><option value="phone">Phone</option><option value="web">Web</option><option value="custom">Custom</option></select><button className="button" disabled={props.busy || !props.canWrite}>创建 Conversation</button>
				</form>
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); const direction = String(values.get("direction")); void submit(event, `/v1/conversations/${values.get("conversation_id")}/messages`, data => ({ direction, status: direction === "inbound" ? "received" : "draft", content: { text: data.get("text") } })); }}>
					<select name="conversation_id" required defaultValue=""><option value="" disabled>选择 Conversation</option>{props.growth.conversations.map(item => <option key={item.id} value={item.id}>{props.growth.leads.find(lead => lead.id === item.lead_id)?.name ?? item.id.slice(0, 8)}</option>)}</select><select name="direction" defaultValue="inbound"><option value="inbound">收到回复</option><option value="outbound">外发草稿</option><option value="system">系统记录</option></select><textarea name="text" required placeholder="消息内容" /><button className="button" disabled={props.busy || !props.canWrite}>记录消息</button>
				</form>
			</div>
			<div className="growth-two-column">
				<div><h3>Deal 与规范报价</h3><div className="transaction-rows">{props.growth.deals.map(item => <div className="transaction-row" key={item.id}><div><strong>{item.name}</strong><small>{item.customer_id} · {item.currency} {item.value_minor}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge><span className="muted">{item.id.slice(0, 8)}</span></div>)}</div>
					<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void submit(event, `/v1/deals/${values.get("deal_id")}/quotes`, data => ({ currency: data.get("currency"), valid_until: new Date(Date.now() + Number(data.get("valid_days")) * 86400000).toISOString(), items: [{ sku_version_id: data.get("sku_version_id"), quantity: Number(data.get("quantity")), input: {} }] })); }}><select name="deal_id" required defaultValue=""><option value="" disabled>开放 Deal</option>{openDeals.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><select name="sku_version_id" required defaultValue=""><option value="" disabled>已发布 SKU</option>{props.orderableSKUs.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}</select><input name="currency" defaultValue="CNY" minLength={3} maxLength={3} required /><input name="quantity" type="number" min="1" defaultValue="1" required /><input name="valid_days" type="number" min="1" defaultValue="7" required /><button className="button primary" disabled={props.busy || !props.canWrite || !openDeals.length || !props.orderableSKUs.length}>创建规范报价</button></form>
				</div>
				<div><h3>增长实验</h3><form className="inline-form" onSubmit={event => submit(event, "/v1/experiments", values => { const [entity_type, entity_id] = String(values.get("entity")).split(":"); return { name: values.get("name"), entity_type, entity_id, hypothesis: values.get("hypothesis"), allocation_basis_points: Number(values.get("allocation")), metrics_definition: objectValue(values, "metrics") }; })}><input name="name" required placeholder="实验名称" /><select name="entity" required defaultValue=""><option value="" disabled>实验对象</option>{entityOptions().map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select><textarea name="hypothesis" required placeholder="可证伪假设" /><input name="allocation" type="number" min="0" max="10000" defaultValue="5000" required /><textarea name="metrics" defaultValue={'{"conversion":{"type":"count"}}'} required /><button className="button primary" disabled={props.busy || !props.canWrite || !entityOptions().length}>创建实验</button></form><div className="transaction-rows">{props.growth.experiments.map(item => <div className="transaction-row" key={item.id}><div><strong>{item.name}</strong><small>{item.hypothesis}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge>{item.status === "draft" ? <button className="button" disabled={props.busy || !props.canWrite} onClick={() => void props.onCommand(`/v1/experiments/${item.id}/transitions`, { to: "running", result: {} })}>启动</button> : item.status === "running" ? <button className="button" disabled={props.busy || !props.canWrite} onClick={() => void props.onCommand(`/v1/experiments/${item.id}/transitions`, { to: "completed", result: { outcome: "recorded" } })}>完成</button> : <span className="muted">v{item.version}</span>}</div>)}</div></div>
			</div>
		</section>
	</div>;
}

type TransactionWorkspaceProps = {
	quotes: Quote[];
	orders: Order[];
	holds: FinanceOverview["holds"];
	selected: Order | null;
	selectedID: string | null;
	orderableSKUs: { id: string; label: string }[];
	providers: Provider[];
	busy: boolean;
	onSelect: (id: string) => void;
	onQuote: (event: FormEvent<HTMLFormElement>) => void;
	onQuoteTransition: (id: string, to: string) => void;
	onOrder: (quoteVersionID: string) => void;
	onOrderTransition: (id: string, to: string) => void;
	onExecution: (id: string, to: string, providerEndpointID?: string) => void;
	onDelivery: (id: string, to: string) => void;
	onUsage: (id: string, quantity: number) => void;
	onCost: (id: string, providerEndpointID: string, currency: string, amountMinor: number) => void;
	onCharge: (id: string) => void;
};

function TransactionWorkspace(props: TransactionWorkspaceProps) {
	const endpoints = props.providers.flatMap(provider => provider.endpoints.filter(endpoint => endpoint.status === "healthy").map(endpoint => ({ id: endpoint.id, label: `${provider.name} · ${endpoint.adapter_type}` })));
	const orderNext: Record<string, string> = { created: "awaiting_payment", awaiting_payment: "paid", paid: "provisioning", provisioning: "active", active: "completed" };
	const executionNext: Record<string, string> = { created: "validating", validating: "reserved", reserved: "queued", queued: "submitted", submitted: "processing", processing: "succeeded", succeeded: "reconciling", reconciling: "settled" };
	const deliveryNext: Record<string, string> = { created: "in_progress", in_progress: "completed", waiting: "in_progress", failed: "in_progress" };
	const selected = props.selected;
	const selectedNext = selected ? orderNext[selected.status] : undefined;
	const paymentReady = selected ? props.holds.filter(item => item.order_id === selected.id && ["active", "partially_captured"].includes(item.status)).reduce((total, item) => total + item.remaining_minor, 0) >= selected.amount_minor : false;
	const activationReady = selected ? selected.executions.length > 0 && selected.executions.every(item => ["succeeded", "reconciling", "settled"].includes(item.status)) && selected.deliveries.every(item => item.status === "completed") : false;
	const completionReady = selected ? selected.executions.length > 0 && selected.executions.every(item => item.status === "settled") : false;
	const orderAdvanceReady = selectedNext === "paid" ? paymentReady : selectedNext === "active" ? activationReady : selectedNext === "completed" ? completionReady : true;

	return <div className="transaction-layout">
		<section className="factory-band">
			<div className="panel-title"><h2>报价配置</h2><span className="muted">{props.orderableSKUs.length} 个可报价 SKU 版本</span></div>
			<form className="form-grid" onSubmit={props.onQuote}>
				<label className="field"><span>Deal 标识</span><input name="deal_id" required /></label>
				<label className="field"><span>客户</span><input name="customer_id" required /></label>
				<label className="field wide"><span>SKU 版本</span><select name="sku_version_id" required defaultValue=""><option value="" disabled>选择已发布 SKU</option>{props.orderableSKUs.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>
				<label className="field"><span>数量</span><input name="quantity" type="number" min="1" defaultValue="1" required /></label>
				<label className="field"><span>币种</span><input name="currency" minLength={3} maxLength={3} defaultValue="USD" required /></label>
				<label className="field"><span>有效天数</span><input name="valid_days" type="number" min="1" max="365" defaultValue="7" required /></label>
				<label className="field wide"><span>订单输入</span><textarea name="input" defaultValue={'{"input":""}'} required /></label>
				<div className="form-actions wide"><button className="button primary" disabled={props.busy || !props.orderableSKUs.length}>创建报价</button></div>
			</form>
		</section>

		<div className="transaction-columns">
			<section>
				<div className="panel-title"><h2>报价</h2><span className="muted">{props.quotes.length} 条</span></div>
				<div className="transaction-rows">{props.quotes.length ? props.quotes.map(quote => {
					const latest = quote.versions.reduce((current, version) => !current || version.version > current.version ? version : current, quote.versions[0]);
					const ordered = latest ? props.orders.some(order => order.quote_version_id === latest.id) : false;
					return <div className="transaction-row" key={quote.id}><div><strong>{quote.customer_id}</strong><small>{quote.deal_id} · {latest ? `${latest.currency} ${latest.amount_minor}` : "-"}</small></div><StatusBadge tone={tone(quote.status)}>{statusLabel[quote.status] ?? quote.status}</StatusBadge><div className="button-row">{quote.status === "draft" && <button className="button" disabled={props.busy} onClick={() => props.onQuoteTransition(quote.id, "accepted")}>接受报价</button>}{quote.status === "accepted" && latest && !ordered && <button className="button primary" disabled={props.busy} onClick={() => props.onOrder(latest.id)}>生成订单</button>}{ordered && <span className="muted">已生成订单</span>}</div></div>;
				}) : <div className="empty-feature">暂无报价</div>}</div>
			</section>
			<section>
				<div className="panel-title"><h2>订单</h2><span className="muted">{props.orders.length} 条</span></div>
				<div className="transaction-rows">{props.orders.length ? props.orders.map(order => <button type="button" className={`transaction-row transaction-select ${order.id === props.selectedID ? "active" : ""}`} key={order.id} onClick={() => props.onSelect(order.id)}><div><strong>{order.customer_id}</strong><small>{order.currency} {order.amount_minor} · {order.id.slice(0, 8)}</small></div><StatusBadge tone={tone(order.status)}>{statusLabel[order.status] ?? order.status}</StatusBadge><span className="muted">v{order.version}</span></button>) : <div className="empty-feature">暂无订单</div>}</div>
			</section>
		</div>

		{selected && <section className="factory-band">
			<div className="panel-title"><h2>订单 {selected.id.slice(0, 8)}</h2><div className="button-row"><StatusBadge tone={tone(selected.status)}>{statusLabel[selected.status] ?? selected.status}</StatusBadge>{selectedNext && <button className="button primary" title={selectedNext === "paid" && !paymentReady ? "请先在财务工作区冻结足额资金" : undefined} disabled={props.busy || !orderAdvanceReady} onClick={() => props.onOrderTransition(selected.id, selectedNext)}>推进至{statusLabel[selectedNext] ?? selectedNext}</button>}</div></div>
			<div className="release-strip"><div><span>订单金额</span><strong>{selected.currency} {selected.amount_minor}</strong></div><div><span>订阅 / 权益</span><strong>{selected.subscriptions.length} / {selected.entitlements.length}</strong></div><div><span>用量 / 成本 / 收费</span><strong>{selected.usage.length} / {selected.provider_costs.length} / {selected.customer_charges.length}</strong></div></div>

			<div className="transaction-section"><h3>执行订单</h3>{selected.executions.length ? selected.executions.map(execution => {
				const next = executionNext[execution.status];
				const usageRecorded = selected.usage.some(item => item.execution_order_id === execution.id);
				const costRecorded = selected.provider_costs.some(item => item.execution_order_id === execution.id);
				const chargeRecorded = selected.customer_charges.some(item => item.execution_order_id === execution.id);
				return <div className="execution-row" key={execution.id}><div className="execution-heading"><div><strong>{execution.id.slice(0, 8)}</strong><small>{execution.bindings.sku_version_id.slice(0, 8)} · attempt {execution.attempt}</small></div><StatusBadge tone={tone(execution.status)}>{statusLabel[execution.status] ?? execution.status}</StatusBadge>{next && <form className="button-row" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); props.onExecution(execution.id, next, String(values.get("provider_endpoint_id") ?? "")); }}>{next === "reserved" && <select name="provider_endpoint_id" required defaultValue=""><option value="" disabled>选择 Endpoint</option>{endpoints.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}</select>}<button className="button" disabled={props.busy || (next === "reserved" && !endpoints.length)}>推进至{statusLabel[next] ?? next}</button></form>}</div>
					{["succeeded", "reconciling", "settled"].includes(execution.status) && <div className="fact-actions">{!usageRecorded && <form className="inline-form" onSubmit={event => { event.preventDefault(); props.onUsage(execution.id, Number(new FormData(event.currentTarget).get("quantity"))); }}><label className="field"><span>用量</span><input name="quantity" type="number" min="0" defaultValue="1" required /></label><button className="button" disabled={props.busy}>记录用量</button></form>}{!costRecorded && <form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); props.onCost(execution.id, String(values.get("provider_endpoint_id")), String(values.get("currency")), Number(values.get("amount_minor"))); }}><label className="field"><span>Endpoint</span><select name="provider_endpoint_id" required defaultValue={execution.provider_endpoint_id ?? ""}><option value="" disabled>选择 Endpoint</option>{endpoints.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label><label className="field"><span>成本</span><input name="amount_minor" type="number" min="0" required /></label><label className="field"><span>币种</span><input name="currency" minLength={3} maxLength={3} defaultValue={selected.currency} required /></label><button className="button" disabled={props.busy || !endpoints.length}>记录成本</button></form>}{usageRecorded && !chargeRecorded && <button className="button primary" disabled={props.busy} onClick={() => props.onCharge(execution.id)}>计算客户收费</button>}{usageRecorded && costRecorded && chargeRecorded && <span className="muted">运营事实已齐备</span>}</div>}
				</div>;
			}) : <div className="empty-feature">订单进入履约后生成执行记录</div>}</div>

			<div className="transaction-section"><h3>交付项目</h3>{selected.deliveries.length ? selected.deliveries.map(delivery => { const next = deliveryNext[delivery.status]; return <div className="transaction-row" key={delivery.id}><div><strong>{delivery.mode}</strong><small>{delivery.id.slice(0, 8)}</small></div><StatusBadge tone={tone(delivery.status)}>{statusLabel[delivery.status] ?? delivery.status}</StatusBadge>{next ? <button className="button" disabled={props.busy} onClick={() => props.onDelivery(delivery.id, next)}>推进至{statusLabel[next] ?? next}</button> : <span className="muted">已结束</span>}</div>; }) : <div className="empty-feature">订单进入履约后生成交付记录</div>}</div>
		</section>}
	</div>;
}

type FinanceWorkspaceProps = {
	finance: FinanceOverview;
	orders: Order[];
	busy: boolean;
	canAdjust: boolean;
	onCommand: (path: string, body: unknown) => Promise<unknown>;
};

function FinanceWorkspace(props: FinanceWorkspaceProps) {
	const payableOrders = props.orders.filter(order => ["created", "awaiting_payment"].includes(order.status));
	const charges = props.orders.flatMap(order => order.customer_charges.map(charge => ({ ...charge, customer_id: order.customer_id })));
	const costs = props.orders.flatMap(order => order.provider_costs.map(cost => ({ ...cost, customer_id: order.customer_id })));
	const openPayableSources = [
		...props.finance.provider_payables.filter(item => ["open", "partially_settled"].includes(item.status)).map(item => ({ type: "provider_payable", id: item.id, label: `Provider ${item.provider_id.slice(0, 8)}`, currency: item.currency, outstanding: item.amount_minor - item.settled_minor })),
		...props.finance.commissions.filter(item => ["open", "partially_settled"].includes(item.status)).map(item => ({ type: "commission", id: item.id, label: `${item.beneficiary_type} ${item.beneficiary_id}`, currency: item.currency, outstanding: item.amount_minor - item.settled_minor }))
	];

	function submit(event: FormEvent<HTMLFormElement>, path: string, build: (values: FormData) => unknown) {
		event.preventDefault();
		const form = event.currentTarget;
		void props.onCommand(path, build(new FormData(form))).then(() => form.reset());
	}

	return <div className="finance-layout">
		<section className="factory-band">
			<div className="panel-title"><h2>钱包与资金冻结</h2><span className="muted">{props.finance.wallets.length} 个钱包 · {props.finance.holds.filter(item => item.remaining_minor > 0).length} 笔在途冻结</span></div>
			<div className="finance-control-grid">
				<form className="inline-form" onSubmit={event => submit(event, "/v1/wallets", values => ({ owner_type: "customer", owner_id: values.get("owner_id"), currency: values.get("currency") }))}>
					<label className="field"><span>客户</span><input name="owner_id" required /></label>
					<label className="field"><span>币种</span><input name="currency" minLength={3} maxLength={3} defaultValue="USD" required /></label>
					<button className="button primary" disabled={props.busy}>创建客户钱包</button>
				</form>
				{props.canAdjust && <form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); const walletID = String(values.get("wallet_id")); void props.onCommand(`/v1/wallets/${walletID}/adjustments`, { direction: "credit", amount_minor: Number(values.get("amount_minor")), reason: values.get("reason") }); }}>
					<label className="field"><span>钱包</span><select name="wallet_id" required defaultValue=""><option value="" disabled>选择钱包</option>{props.finance.wallets.map(item => <option key={item.id} value={item.id}>{item.owner_id} · {item.currency}</option>)}</select></label>
					<label className="field"><span>入账金额</span><input name="amount_minor" type="number" min="1" required /></label>
					<label className="field"><span>调整原因</span><input name="reason" required /></label>
					<button className="button" disabled={props.busy || !props.finance.wallets.length}>管理员入账</button>
				</form>}
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); const orderID = String(values.get("order_id")); void props.onCommand(`/v1/orders/${orderID}/holds`, { wallet_id: values.get("wallet_id"), amount_minor: Number(values.get("amount_minor")) }); }}>
					<label className="field"><span>待支付订单</span><select name="order_id" required defaultValue=""><option value="" disabled>选择订单</option>{payableOrders.map(item => <option key={item.id} value={item.id}>{item.customer_id} · {item.currency} {item.amount_minor}</option>)}</select></label>
					<label className="field"><span>客户钱包</span><select name="wallet_id" required defaultValue=""><option value="" disabled>选择钱包</option>{props.finance.wallets.filter(item => item.owner_type === "customer" && item.status === "active").map(item => <option key={item.id} value={item.id}>{item.owner_id} · 可用 {item.currency} {item.available_minor}</option>)}</select></label>
					<label className="field"><span>冻结金额</span><input name="amount_minor" type="number" min="1" required /></label>
					<button className="button primary" disabled={props.busy || !payableOrders.length || !props.finance.wallets.length}>冻结资金</button>
				</form>
			</div>
			<div style={{ overflowX: "auto" }}><table className="data-grid finance-table"><thead><tr><th>所有者</th><th>币种</th><th>可用</th><th>冻结</th><th>状态</th></tr></thead><tbody>{props.finance.wallets.map(item => <tr key={item.id}><td><strong>{item.owner_id}</strong><div className="muted">{item.owner_type} · {item.id.slice(0, 8)}</div></td><td>{item.currency}</td><td>{item.available_minor}</td><td>{item.held_minor}</td><td><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge></td></tr>)}</tbody></table></div>
			{props.finance.holds.length > 0 && <div className="finance-action-list">{props.finance.holds.map(item => <div className="transaction-row" key={item.id}><div><strong>订单 {item.order_id.slice(0, 8)}</strong><small>{item.currency} {item.amount_minor} · 已扣 {item.captured_minor} · 已释放 {item.released_minor}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge>{item.remaining_minor > 0 ? <button className="button" disabled={props.busy} onClick={() => void props.onCommand(`/v1/holds/${item.id}/releases`, { amount_minor: 0 })}>释放余额 {item.remaining_minor}</button> : <span className="muted">余额 0</span>}</div>)}</div>}
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>收费、应付与佣金</h2><span className="muted">业务事实与账本分录独立关联</span></div>
			<div className="finance-columns">
				<div><h3>客户收费</h3><div className="transaction-rows">{charges.length ? charges.map(charge => {
					const hold = props.finance.holds.find(item => item.order_id === charge.order_id && item.currency === charge.currency && ["active", "partially_captured"].includes(item.status) && item.remaining_minor >= charge.amount_minor);
					const hasCommission = props.finance.commissions.some(item => item.customer_charge_id === charge.id && item.status !== "reversed");
					const wallet = props.finance.wallets.find(item => item.owner_type === "customer" && item.owner_id === charge.customer_id && item.currency === charge.currency);
					return <div className="finance-fact" key={charge.id}><div className="transaction-row"><div><strong>{charge.customer_id}</strong><small>{charge.currency} {charge.amount_minor} · {charge.id.slice(0, 8)}</small></div><StatusBadge tone={tone(charge.status)}>{statusLabel[charge.status] ?? charge.status}</StatusBadge><div className="button-row">{charge.status === "calculated" && hold && <button className="button primary" disabled={props.busy} onClick={() => void props.onCommand(`/v1/customer-charges/${charge.id}/postings`, { hold_id: hold.id })}>正式扣费</button>}{charge.status === "calculated" && !hold && <span className="muted">待足额冻结</span>}{charge.status === "posted" && !hasCommission && <button className="button" disabled={props.busy} onClick={() => void props.onCommand(`/v1/customer-charges/${charge.id}/commissions`, { beneficiary_type: "reseller", beneficiary_id: "Test Reseller", amount_minor: Math.max(1, Math.floor(charge.amount_minor / 10)) })}>记录佣金</button>}</div></div>{charge.status === "posted" && wallet && <form className="finance-refund" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); void props.onCommand(`/v1/customer-charges/${charge.id}/refunds`, { wallet_id: wallet.id, amount_minor: Number(values.get("amount_minor")), reason: values.get("reason") }); }}><input name="amount_minor" type="number" min="1" max={charge.amount_minor} placeholder="退款金额" required /><input name="reason" placeholder="退款原因" required /><button className="button" disabled={props.busy}>退款</button></form>}</div>;
				}) : <div className="empty-feature">暂无客户收费</div>}</div></div>
				<div><h3>供应商成本</h3><div className="transaction-rows">{costs.length ? costs.map(cost => { const payable = props.finance.provider_payables.find(item => item.provider_cost_id === cost.id); return <div className="transaction-row" key={cost.id}><div><strong>{cost.currency} {cost.amount_minor}</strong><small>ProviderCost {cost.id.slice(0, 8)}</small></div><StatusBadge tone={payable ? tone(payable.status) : "amber"}>{payable ? statusLabel[payable.status] ?? payable.status : "待入应付"}</StatusBadge>{payable ? <span className="muted">应付 {payable.amount_minor - payable.settled_minor}</span> : <button className="button primary" disabled={props.busy} onClick={() => void props.onCommand(`/v1/provider-costs/${cost.id}/payables`, {})}>确认应付</button>}</div>; }) : <div className="empty-feature">暂无供应商成本</div>}</div></div>
			</div>
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>结算与对账</h2><span className="muted">{props.finance.settlements.length} 笔结算 · {props.finance.reconciliation_runs.length} 次对账</span></div>
			<div className="finance-control-grid two">
				<form className="inline-form" onSubmit={event => { event.preventDefault(); const values = new FormData(event.currentTarget); const source = openPayableSources.find(item => `${item.type}:${item.id}` === values.get("source")); if (source) void props.onCommand("/v1/settlements", { source_type: source.type, source_id: source.id, amount_minor: Number(values.get("amount_minor")) }); }}>
					<label className="field"><span>未结应付</span><select name="source" required defaultValue=""><option value="" disabled>选择应付</option>{openPayableSources.map(item => <option key={`${item.type}:${item.id}`} value={`${item.type}:${item.id}`}>{item.label} · {item.currency} {item.outstanding}</option>)}</select></label>
					<label className="field"><span>结算金额（0 为全部）</span><input name="amount_minor" type="number" min="0" defaultValue="0" required /></label>
					<button className="button primary" disabled={props.busy || !openPayableSources.length}>执行结算</button>
				</form>
				<form className="inline-form" onSubmit={event => submit(event, "/v1/reconciliation-runs", values => ({ order_id: values.get("order_id") }))}>
					<label className="field"><span>订单范围</span><select name="order_id" required defaultValue=""><option value="" disabled>选择订单</option>{props.orders.map(item => <option key={item.id} value={item.id}>{item.customer_id} · {item.id.slice(0, 8)}</option>)}</select></label>
					<button className="button" disabled={props.busy || !props.orders.length}>运行对账</button>
				</form>
			</div>
			<div className="finance-columns">
				<div><h3>结算记录</h3><div className="transaction-rows">{props.finance.settlements.length ? props.finance.settlements.map(item => <div className="transaction-row" key={item.id}><div><strong>{item.beneficiary_id}</strong><small>{item.source_type} · {item.currency} {item.amount_minor}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge><span className="muted">{item.id.slice(0, 8)}</span></div>) : <div className="empty-feature">暂无结算</div>}</div></div>
				<div><h3>对账记录</h3><div className="transaction-rows">{props.finance.reconciliation_runs.length ? props.finance.reconciliation_runs.map(item => <div className="transaction-row" key={item.id}><div><strong>订单 {item.order_id?.slice(0, 8) ?? "全部"}</strong><small>核对 {item.checked_count} · 差异 {item.discrepancy_count}</small></div><StatusBadge tone={tone(item.status)}>{statusLabel[item.status] ?? item.status}</StatusBadge><span className="muted">{item.id.slice(0, 8)}</span></div>) : <div className="empty-feature">暂无对账</div>}</div></div>
			</div>
		</section>

		<section className="factory-band">
			<div className="panel-title"><h2>只追加账本</h2><span className="muted">{props.finance.accounts.length} 个账户 · {props.finance.transactions.length} 笔交易</span></div>
			<div style={{ overflowX: "auto" }}><table className="data-grid finance-table"><thead><tr><th>交易</th><th>引用</th><th>借方</th><th>贷方</th><th>时间</th></tr></thead><tbody>{props.finance.transactions.map(item => <tr key={item.id}><td><strong>{item.transaction_type}</strong><div className="muted">{item.id.slice(0, 8)}</div></td><td>{item.reference_type}<div className="muted">{item.reference_id.slice(0, 12)}</div></td><td>{item.entries.filter(entry => entry.direction === "debit").reduce((total, entry) => total + entry.amount_minor, 0)}</td><td>{item.entries.filter(entry => entry.direction === "credit").reduce((total, entry) => total + entry.amount_minor, 0)}</td><td>{new Date(item.created_at).toLocaleString("zh-CN")}</td></tr>)}</tbody></table></div>
		</section>
	</div>;
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
