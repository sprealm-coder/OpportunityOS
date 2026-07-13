import { z } from "zod";
export const MoneySchema=z.object({currency:z.string().length(3),minor:z.number().int()});
export const EventEnvelopeSchema=z.object({id:z.string(),tenant_id:z.string(),aggregate_type:z.string(),aggregate_id:z.string(),event_type:z.string(),version:z.number().int().positive(),trace_id:z.string(),occurred_at:z.string().datetime(),payload:z.record(z.string(),z.unknown())});
export const OpportunitySchema=z.object({id:z.string(),tenant_id:z.string(),name:z.string().min(1),description:z.string(),status:z.enum(["detected","enriched","scored","under_review","approved","incubating","rejected","archived"]),score:z.number().int().min(0).max(100),version:z.number().int().positive()});
export type Money=z.infer<typeof MoneySchema>;export type Opportunity=z.infer<typeof OpportunitySchema>;

