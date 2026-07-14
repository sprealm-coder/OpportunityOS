"use client";

import { CoreApiClient, type AnalyticsOverview, type AuditRecord, type Collection, type FinanceOverview, type GrowthOverview, type OperationsOverview, type Session } from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");

const nav = ["系统首页", "运行健康", "结果反馈", "租户与品牌", "角色与权限", "功能开关", "审计日志", "增长治理", "财务治理"].map(label => ({ label, href: "#operations" }));
const objects = [
  ["Opportunity", "8 状态", "PostgreSQL + Outbox"],
  ["IncubationProject", "8 状态", "事务阶段门"],
  ["BusinessBlueprint", "9 状态", "JSON 定义"],
  ["ProductVersion", "不可变", "11 版本绑定"],
  ["Wallet / Hold", "资金状态", "可用与冻结分离"],
  ["LedgerTransaction", "只追加", "借贷平衡 + 快照"],
  ["ProviderPayable", "应付结算", "成本事实独立"],
  ["ReconciliationRun", "匹配 / 差异", "运营与账本对照"],
	["Lead / Deal", "增长漏斗", "Lead ≠ Customer · Deal ≠ Order"],
	["Campaign / Suppression", "审批与阻止", "版本审批 + 配额预留"]
];

const emptyFinance: FinanceOverview = { wallets: [], accounts: [], transactions: [], holds: [], refunds: [], commissions: [], provider_payables: [], settlements: [], reconciliation_runs: [] };
const emptyGrowth: GrowthOverview = { segments: [], icps: [], leads: [], evidence: [], contacts: [], proof_templates: [], proof_requests: [], proof_instances: [], campaigns: [], suppressions: [], outreach: [], conversations: [], deals: [], experiments: [] };
const emptyAnalytics: AnalyticsOverview = { feedback: [], projections: [] };
const emptyOperations: OperationsOverview = { outbox: { pending: 0, retry_scheduled: 0, dead_letter: 0, oldest_pending_age_seconds: 0 }, workflow_runs: [], adapter_identities: [], adapter_receipts: [], alerts: [], deployment_checks: [] };

export default function Page() {
	const [session, setSession] = useState<Session | null>(null);
	const [authLoading, setAuthLoading] = useState(true);
	const [authBusy, setAuthBusy] = useState(false);
	const [authError, setAuthError] = useState<string | null>(null);
  const [audit, setAudit] = useState<AuditRecord[]>([]);
	const [finance, setFinance] = useState<FinanceOverview>(emptyFinance);
	const [growth, setGrowth] = useState<GrowthOverview>(emptyGrowth);
	const [analytics, setAnalytics] = useState<AnalyticsOverview>(emptyAnalytics);
	const [operations, setOperations] = useState<OperationsOverview>(emptyOperations);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const load = useCallback(async () => {
		const [auditResult, financeResult, growthResult, analyticsResult, operationsResult] = await Promise.all([api.get<Collection<AuditRecord>>("/v1/audit"), api.get<FinanceOverview>("/v1/finance"), api.get<GrowthOverview>("/v1/growth"), api.get<AnalyticsOverview>("/v1/analytics/outcomes"), api.get<OperationsOverview>("/v1/operations")]);
		setAudit(auditResult.items);
		setFinance(financeResult);
		setGrowth(growthResult);
		setAnalytics(analyticsResult);
		setOperations(operationsResult);
  }, []);

  useEffect(() => {
		api.session().then(setSession).catch(() => setSession(null)).finally(() => setAuthLoading(false));
	}, []);

	useEffect(() => {
		if (!session) return;
		setLoading(true);
		load().catch(error => setError(error instanceof Error ? error.message : "无法读取审计日志")).finally(() => setLoading(false));
	}, [load, session]);

	async function login(email: string, password: string) {
		setAuthBusy(true);
		setAuthError(null);
		try {
			setSession(await api.login(email, password));
		} catch (error) {
			setAuthError(error instanceof Error ? error.message : "登录失败");
		} finally {
			setAuthBusy(false);
		}
	}

	async function logout() {
		await api.logout().catch(() => undefined);
		setSession(null);
		setAudit([]);
		setFinance(emptyFinance);
		setGrowth(emptyGrowth);
		setAnalytics(emptyAnalytics);
		setOperations(emptyOperations);
	}

  const actors = useMemo(() => new Set(audit.map(item => item.actor_id)).size, [audit]);
  const objectsTouched = useMemo(() => new Set(audit.map(item => `${item.object_type}:${item.object_id}`)).size, [audit]);
	if (authLoading) return <div className="auth-screen"><div className="loading">正在恢复会话...</div></div>;
	if (!session) return <LoginScreen title="平台管理" error={authError} busy={authBusy} onLogin={login} />;

	return <PortalShell title="平台管理" role="Admin Web" navigation={nav} tenantLabel={session.tenant_name} userLabel={`${session.display_name} · ${session.role}`} onLogout={logout}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics">
      <Metric label="审计事件" value={String(audit.length)} detail="当前租户最近 200 条" />
      <Metric label="活跃操作者" value={String(actors)} detail="来自持久化 Actor ID" />
      <Metric label="涉及对象" value={String(objectsTouched)} detail="按类型与对象 ID 去重" />
			<Metric label="账本交易" value={String(finance.transactions.length)} detail={`${finance.wallets.length} 个钱包 · ${finance.reconciliation_runs.filter(item => item.status === "discrepancy").length} 次差异`} />
			<Metric label="增长治理" value={String(growth.campaigns.filter(item => item.status === "pending_approval").length)} detail={`${growth.suppressions.length} 条抑制 · ${growth.outreach.filter(item => item.status === "blocked").length} 次阻止`} />
			<Metric label="运行告警" value={String(operations.alerts.filter(item => item.status !== "resolved").length)} detail={`${operations.outbox.dead_letter} 条死信 · ${operations.outbox.retry_scheduled} 条重试`} />
    </div>

		<section className="panel" id="operations">
			<div className="panel-title"><h2>生产运行健康</h2><StatusBadge tone={operations.deployment_checks.every(item => item.status === "passed") ? "green" : "amber"}>{operations.deployment_checks.filter(item => item.status === "passed").length}/{operations.deployment_checks.length} 检查通过</StatusBadge></div>
			<div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>检查</th><th>结果</th><th>说明</th></tr></thead><tbody>{operations.deployment_checks.map(check => <tr key={check.name}><td><strong>{check.name}</strong></td><td><StatusBadge tone={check.status === "passed" ? "green" : "amber"}>{check.status === "passed" ? "通过" : "阻断"}</StatusBadge></td><td>{check.message}</td></tr>)}</tbody></table></div>
			<div className="metrics" style={{ marginTop: 16 }}><Metric label="Outbox 待投递" value={String(operations.outbox.pending)} detail={`最老等待 ${operations.outbox.oldest_pending_age_seconds} 秒`} /><Metric label="工作流运行" value={String(operations.workflow_runs.length)} detail={`${operations.workflow_runs.filter(item => item.status === "retry_wait").length} 个等待重试`} /><Metric label="可信回执" value={String(operations.adapter_receipts.length)} detail={`${operations.adapter_identities.length} 个 Adapter 身份`} /></div>
		</section>

		<section className="panel">
			<div className="panel-title"><h2>机会结果反馈</h2><StatusBadge>{analytics.feedback.length} 条已验证反馈</StatusBadge></div>
			{analytics.projections.length === 0 ? <div className="empty-feature">完成订单结算与对账后生成机会结果投影</div> : <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>机会</th><th>反馈数</th><th>客户收费</th><th>毛利</th><th>更新时间</th></tr></thead><tbody>{analytics.projections.map(item => <tr key={item.opportunity_id}><td><strong>{item.opportunity_id.slice(0, 12)}</strong></td><td>{item.feedback_count}</td><td>{String(item.latest_metrics.customer_charge_minor ?? "-")}</td><td>{String(item.latest_metrics.gross_margin_minor ?? "-")}</td><td>{item.updated_at ? new Date(item.updated_at).toLocaleString("zh-CN") : "-"}</td></tr>)}</tbody></table></div>}
		</section>

    <section className="panel">
      <div className="panel-title"><h2>核心对象注册表</h2><StatusBadge>PostgreSQL 运行时</StatusBadge></div>
      <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>对象</th><th>生命周期</th><th>持久化契约</th><th>状态</th></tr></thead><tbody>{objects.map(row => <tr key={row[0]}><td><strong>{row[0]}</strong></td><td>{row[1]}</td><td>{row[2]}</td><td><StatusBadge>已注册</StatusBadge></td></tr>)}</tbody></table></div>
    </section>

		<section className="panel">
			<div className="panel-title"><h2>增长安全边界</h2><StatusBadge tone={growth.outreach.some(item => item.status === "blocked") ? "amber" : "green"}>外部发送关闭</StatusBadge></div>
			<div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>治理对象</th><th>总量</th><th>待处理</th><th>控制</th></tr></thead><tbody>
				<tr><td><strong>Campaign</strong></td><td>{growth.campaigns.length}</td><td>{growth.campaigns.filter(item => item.status === "pending_approval").length}</td><td>版本化人工审批</td></tr>
				<tr><td><strong>Proof</strong></td><td>{growth.proof_requests.length}</td><td>{growth.proof_requests.filter(item => item.status === "review").length}</td><td>Schema、访问策略与到期</td></tr>
				<tr><td><strong>触达计划</strong></td><td>{growth.outreach.length}</td><td>{growth.outreach.filter(item => item.status === "blocked").length}</td><td>抑制优先于配额预留</td></tr>
				<tr><td><strong>Deal</strong></td><td>{growth.deals.length}</td><td>{growth.deals.filter(item => ["open", "proposal"].includes(item.status)).length}</td><td>规范 Quote 外键</td></tr>
			</tbody></table></div>
		</section>

    <section className="panel" id="audit">
      <div className="panel-title"><h2>租户审计日志</h2><button className="button" onClick={() => load().catch(error => setError(error instanceof Error ? error.message : "刷新失败"))}>刷新</button></div>
      {loading ? <div className="loading">正在读取审计记录...</div> : audit.length === 0 ? <div className="empty-feature">暂无审计记录，先在运营端创建机会</div> :
        <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>对象</th><th>请求 ID</th></tr></thead><tbody>{audit.map(record => <tr key={record.id}><td>{new Date(record.created_at).toLocaleString("zh-CN")}</td><td>{record.actor_id}</td><td><strong>{record.action}</strong></td><td>{record.object_type}<div className="muted">{record.object_id.slice(0, 12)}</div></td><td>{record.request_id.slice(0, 12)}</td></tr>)}</tbody></table></div>}
    </section>
  </PortalShell>;
}
