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
export const SessionSchema = z.object({
  id: z.string(), user_id: z.string(), email: z.string().email(), display_name: z.string(),
  tenant_id: z.string(), tenant_name: z.string(), role: z.string(), expires_at: z.string().datetime()
});
export const CapabilitySchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), description: z.string(),
  definition: z.record(z.string(), z.unknown())
});
export const ProviderEndpointSchema = z.object({
  id: z.string(), tenant_id: z.string(), provider_id: z.string(), capability_id: z.string(),
  adapter_type: z.string(), adapter_version: z.string(), status: z.string()
});
export const ProviderSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), status: z.string(),
  endpoints: z.array(ProviderEndpointSchema).default([])
});
export const ProductSchema = z.object({
  id: z.string(), tenant_id: z.string(), blueprint_id: z.string(), name: z.string(),
  status: z.string(), version: z.number().int().positive(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ProductVersionSchema = z.object({
  id: z.string(), tenant_id: z.string(), product_id: z.string(), version: z.number().int().positive(),
  input_schema: z.record(z.string(), z.unknown()), output_schema: z.record(z.string(), z.unknown()),
  form_schema: z.record(z.string(), z.unknown()), capability_ids: z.array(z.string()),
  workflow: z.record(z.string(), z.unknown()), metering_id: z.string(), price_book_id: z.string(),
  route_policy_id: z.string(), delivery_mode: z.string(), compliance_profile_id: z.string(),
  compliance_profile: z.record(z.string(), z.unknown()), growth_playbook: z.record(z.string(), z.unknown()),
  created_by: z.string(), created_at: z.string().datetime()
});
export const SKUVersionSchema = z.object({
  id: z.string(), tenant_id: z.string(), sku_id: z.string(), product_version_id: z.string(),
  workflow_version_id: z.string(), metering_version_id: z.string(), pricing_version_id: z.string(),
  routing_version_id: z.string(), version: z.number().int().positive(),
  entitlements: z.record(z.string(), z.unknown()), created_by: z.string(), created_at: z.string().datetime()
});
export const SKUSchema = z.object({
  id: z.string(), tenant_id: z.string(), product_id: z.string(), code: z.string(), name: z.string(),
  status: z.string(), created_by: z.string(), created_at: z.string().datetime(),
  versions: z.array(SKUVersionSchema).default([])
});
export const PublicationSchema = z.object({
  id: z.string(), tenant_id: z.string(), product_id: z.string(), product_version_id: z.string(),
  status: z.string(), published_by: z.string(), published_at: z.string().datetime()
});
export const ProductDetailSchema = ProductSchema.extend({
  versions: z.array(ProductVersionSchema).default([]), skus: z.array(SKUSchema).default([]),
  publications: z.array(PublicationSchema).default([])
});

export type Money = z.infer<typeof MoneySchema>;
export type Evidence = z.infer<typeof EvidenceSchema>;
export type Opportunity = z.infer<typeof OpportunitySchema>;
export type Incubation = z.infer<typeof IncubationSchema>;
export type Blueprint = z.infer<typeof BlueprintSchema>;
export type AuditRecord = z.infer<typeof AuditRecordSchema>;
export type Session = z.infer<typeof SessionSchema>;
export type Capability = z.infer<typeof CapabilitySchema>;
export type ProviderEndpoint = z.infer<typeof ProviderEndpointSchema>;
export type Provider = z.infer<typeof ProviderSchema>;
export type Product = z.infer<typeof ProductSchema>;
export type ProductVersion = z.infer<typeof ProductVersionSchema>;
export type SKUVersion = z.infer<typeof SKUVersionSchema>;
export type SKU = z.infer<typeof SKUSchema>;
export type Publication = z.infer<typeof PublicationSchema>;
export type ProductDetail = z.infer<typeof ProductDetailSchema>;

export type Collection<T> = { items: T[] };

export class CoreApiError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(message);
  }
}

export class CoreApiClient {
  constructor(private readonly baseURL: string) {}

  async login(email: string, password: string): Promise<Session> {
    return this.request<Session>("/v1/auth/sessions", {
      method: "POST", body: JSON.stringify({ email, password }), headers: { "Content-Type": "application/json" }
    });
  }

  async session(): Promise<Session> {
    return this.get<Session>("/v1/auth/session");
  }

  async logout(): Promise<void> {
    await this.request("/v1/auth/session", { method: "DELETE" });
  }

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
      credentials: "include",
      headers: { ...(init.headers ?? {}) }
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
