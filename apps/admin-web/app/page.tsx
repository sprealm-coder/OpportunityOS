"use client";

import { CoreApiClient, type AuditRecord, type Collection, type Session } from "@opportunity-os/contracts";
import { LoginScreen, Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080");

const nav = ["系统首页", "租户与品牌", "角色与权限", "功能开关", "审计日志", "风险事件", "对象 Schema", "发布策略"].map(label => ({ label, href: "#audit" }));
const objects = [
  ["Opportunity", "8 状态", "PostgreSQL + Outbox"],
  ["IncubationProject", "8 状态", "事务阶段门"],
  ["BusinessBlueprint", "9 状态", "JSON 定义"],
  ["ProductVersion", "不可变", "11 版本绑定"],
  ["LedgerTransaction", "只追加", "借贷平衡"]
];

export default function Page() {
	const [session, setSession] = useState<Session | null>(null);
	const [authLoading, setAuthLoading] = useState(true);
	const [authBusy, setAuthBusy] = useState(false);
	const [authError, setAuthError] = useState<string | null>(null);
  const [audit, setAudit] = useState<AuditRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const load = useCallback(async () => {
    const result = await api.get<Collection<AuditRecord>>("/v1/audit");
    setAudit(result.items);
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
      <Metric label="事务模式" value="ON" detail="Aggregate + Audit + Outbox" />
    </div>

    <section className="panel">
      <div className="panel-title"><h2>核心对象注册表</h2><StatusBadge>PostgreSQL 运行时</StatusBadge></div>
      <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>对象</th><th>生命周期</th><th>持久化契约</th><th>状态</th></tr></thead><tbody>{objects.map(row => <tr key={row[0]}><td><strong>{row[0]}</strong></td><td>{row[1]}</td><td>{row[2]}</td><td><StatusBadge>已注册</StatusBadge></td></tr>)}</tbody></table></div>
    </section>

    <section className="panel" id="audit">
      <div className="panel-title"><h2>租户审计日志</h2><button className="button" onClick={() => load().catch(error => setError(error instanceof Error ? error.message : "刷新失败"))}>刷新</button></div>
      {loading ? <div className="loading">正在读取审计记录...</div> : audit.length === 0 ? <div className="empty-feature">暂无审计记录，先在运营端创建机会</div> :
        <div style={{ overflowX: "auto" }}><table className="data-grid"><thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>对象</th><th>请求 ID</th></tr></thead><tbody>{audit.map(record => <tr key={record.id}><td>{new Date(record.created_at).toLocaleString("zh-CN")}</td><td>{record.actor_id}</td><td><strong>{record.action}</strong></td><td>{record.object_type}<div className="muted">{record.object_id.slice(0, 12)}</div></td><td>{record.request_id.slice(0, 12)}</td></tr>)}</tbody></table></div>}
    </section>
  </PortalShell>;
}
