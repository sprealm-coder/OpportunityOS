import { z } from "zod";

export const MoneySchema = z.object({ currency: z.string().length(3), minor: z.number().int() });
export const EventEnvelopeSchema = z.object({
  id: z.string(), tenant_id: z.string(), aggregate_type: z.string(), aggregate_id: z.string(),
  event_type: z.string(), version: z.number().int().positive(), trace_id: z.string(),
  occurred_at: z.string().datetime(), payload: z.record(z.string(), z.unknown())
});
export const EvidenceSchema = z.object({
  id: z.string(), tenant_id: z.string(), opportunity_id: z.string(), kind: z.string(),
  summary: z.string(), confidence: z.number().int().min(0).max(100), created_at: z.string().datetime()
});
export const OpportunitySchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string().min(1), description: z.string(),
  status: z.enum(["detected", "enriched", "scored", "under_review", "approved", "incubating", "rejected", "archived"]),
  score: z.number().int().min(0).max(100), version: z.number().int().positive(),
  evidence: z.array(EvidenceSchema).default([]), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const IncubationSchema = z.object({
  id: z.string(), tenant_id: z.string(), opportunity_id: z.string(), name: z.string(),
  status: z.string(), version: z.number().int().positive(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const BlueprintSchema = z.object({
  id: z.string(), tenant_id: z.string(), source_opportunity_id: z.string(), name: z.string(),
  description: z.string(), status: z.string(), version: z.number().int().positive(),
  value_proposition: z.string(), required_capabilities: z.array(z.string()),
  created_by: z.string(), approved_by: z.string().optional(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const AuditRecordSchema = z.object({
  id: z.string(), tenant_id: z.string(), actor_id: z.string(), action: z.string(),
  object_type: z.string(), object_id: z.string(), request_id: z.string(), trace_id: z.string(),
  metadata: z.record(z.string(), z.unknown()).optional(), created_at: z.string().datetime()
});

export type Money = z.infer<typeof MoneySchema>;
export type Evidence = z.infer<typeof EvidenceSchema>;
export type Opportunity = z.infer<typeof OpportunitySchema>;
export type Incubation = z.infer<typeof IncubationSchema>;
export type Blueprint = z.infer<typeof BlueprintSchema>;
export type AuditRecord = z.infer<typeof AuditRecordSchema>;

export type Collection<T> = { items: T[] };

export class CoreApiError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(message);
  }
}

export class CoreApiClient {
  constructor(
    private readonly baseURL: string,
    private readonly tenantID: string,
    private readonly actorID: string
  ) {}

  async get<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: "GET" });
  }

  async command<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      headers: { "Content-Type": "application/json", "Idempotency-Key": crypto.randomUUID() }
    });
  }

  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseURL}${path}`, {
      ...init,
      cache: "no-store",
      headers: {
        "X-Tenant-ID": this.tenantID,
        "X-Actor-ID": this.actorID,
        ...(init.headers ?? {})
      }
    });
    const payload = await response.json().catch(() => ({})) as {
      error?: { code?: string; message?: string };
    };
    if (!response.ok) {
      throw new CoreApiError(payload.error?.code ?? "request_failed", payload.error?.message ?? "Request failed", response.status);
    }
    return payload as T;
  }
}
