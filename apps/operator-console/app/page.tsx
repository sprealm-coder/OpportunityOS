"use client";

import {
  CoreApiClient,
  type Blueprint,
  type Collection,
  type Incubation,
  type Opportunity
} from "@opportunity-os/contracts";
import { Metric, PortalShell, StatusBadge } from "@opportunity-os/ui";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

const api = new CoreApiClient(
  process.env.NEXT_PUBLIC_CORE_API_URL ?? "http://127.0.0.1:8080",
  process.env.NEXT_PUBLIC_TENANT_ID ?? "00000000-0000-4000-8000-000000000001",
  process.env.NEXT_PUBLIC_ACTOR_ID ?? "operator-console"
);

const nav = ["运行概览", "机会审核", "孵化项目", "业务蓝图", "产品发布", "工作流运行", "对账队列"].map(label => ({ label, href: "#workspace" }));
const statusLabel: Record<string, string> = {
  detected: "待补充证据", enriched: "待评分", scored: "待送审", under_review: "审核中",
  approved: "已批准", incubating: "孵化中", rejected: "已拒绝", archived: "已归档",
  draft: "草稿", researching: "研究中", validating: "验证中", analyzing: "分析中"
};

function tone(status: string): "green" | "amber" | "gray" {
  if (["approved", "incubating", "ready", "launched"].includes(status)) return "green";
  if (["enriched", "scored", "under_review", "researching", "validating", "analyzing"].includes(status)) return "amber";
  return "gray";
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "操作未完成";
}

export default function Page() {
  const [opportunities, setOpportunities] = useState<Opportunity[]>([]);
  const [incubations, setIncubations] = useState<Incubation[]>([]);
  const [blueprints, setBlueprints] = useState<Blueprint[]>([]);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [view, setView] = useState<"opportunities" | "incubations" | "blueprints">("opportunities");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    const [opportunityResult, incubationResult, blueprintResult] = await Promise.all([
      api.get<Collection<Opportunity>>("/v1/opportunities"),
      api.get<Collection<Incubation>>("/v1/incubations"),
      api.get<Collection<Blueprint>>("/v1/blueprints")
    ]);
    setOpportunities(opportunityResult.items);
    setIncubations(incubationResult.items);
    setBlueprints(blueprintResult.items);
    setSelectedID(current => current ?? opportunityResult.items[0]?.id ?? null);
  }, []);

  useEffect(() => {
    reload().catch(error => setError(errorMessage(error))).finally(() => setLoading(false));
  }, [reload]);

  const selected = useMemo(() => opportunities.find(item => item.id === selectedID) ?? null, [opportunities, selectedID]);
  const pendingReviews = opportunities.filter(item => item.status === "under_review").length;

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

  return <PortalShell title="运营控制台" role="Operator Console" navigation={nav}>
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="metrics">
      <Metric label="机会总数" value={String(opportunities.length)} detail="PostgreSQL 实时数据" />
      <Metric label="待人工审核" value={String(pendingReviews)} detail="需要批准或拒绝" />
      <Metric label="孵化项目" value={String(incubations.length)} detail="版本化阶段门" />
      <Metric label="业务蓝图" value={String(blueprints.length)} detail="不可变发布基础" />
    </div>

    <section className="panel" id="workspace">
      <div className="tabs" role="tablist" aria-label="运营对象">
        <button className={`tab ${view === "opportunities" ? "active" : ""}`} onClick={() => setView("opportunities")}>机会</button>
        <button className={`tab ${view === "incubations" ? "active" : ""}`} onClick={() => setView("incubations")}>孵化项目</button>
        <button className={`tab ${view === "blueprints" ? "active" : ""}`} onClick={() => setView("blueprints")}>业务蓝图</button>
      </div>
      {loading ? <div className="loading">正在读取 Core API...</div> : view === "opportunities" ?
        <OpportunityWorkspace opportunities={opportunities} selected={selected} selectedID={selectedID} busy={busy}
          onSelect={setSelectedID} onCreate={createOpportunity} onEvidence={addEvidence} onScore={scoreOpportunity}
          onTransition={to => selected && execute(() => api.command(`/v1/opportunities/${selected.id}/transitions`, { to }))}
          onReview={reviewOpportunity} onIncubation={createIncubation} onBlueprint={createBlueprint} /> :
        view === "incubations" ? <ObjectRecords items={incubations.map(item => ({ id: item.id, title: item.name, status: item.status, detail: `Opportunity ${item.opportunity_id} · v${item.version}` }))} /> :
          <ObjectRecords items={blueprints.map(item => ({ id: item.id, title: item.name, status: item.status, detail: `${item.value_proposition || "待补充价值主张"} · v${item.version}` }))} />}
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
