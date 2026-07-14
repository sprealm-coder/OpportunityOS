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
export const VersionBindingsSchema = z.object({
  product_version_id: z.string(), sku_version_id: z.string(), pricing_version_id: z.string(),
  workflow_version_id: z.string(), routing_version_id: z.string(), contract_version_id: z.string().optional()
});
export const QuoteItemSchema = z.object({
  id: z.string(), tenant_id: z.string(), quote_version_id: z.string(), quantity: z.number().int().positive(),
  unit_amount_minor: z.number().int().nonnegative(), amount_minor: z.number().int().nonnegative(),
  input: z.record(z.string(), z.unknown()), bindings: VersionBindingsSchema
});
export const QuoteVersionSchema = z.object({
  id: z.string(), tenant_id: z.string(), quote_id: z.string(), version: z.number().int().positive(),
  currency: z.string().length(3), amount_minor: z.number().int().nonnegative(), valid_until: z.string().datetime(),
  created_by: z.string(), created_at: z.string().datetime(), items: z.array(QuoteItemSchema).default([])
});
export const QuoteSchema = z.object({
  id: z.string(), tenant_id: z.string(), deal_id: z.string(), growth_deal_id: z.string().optional(), customer_id: z.string(), status: z.string(),
  version: z.number().int().positive(), created_by: z.string(), created_at: z.string().datetime(),
  updated_at: z.string().datetime(), versions: z.array(QuoteVersionSchema).default([])
});
export const OrderItemSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), quantity: z.number().int().positive(),
  unit_amount_minor: z.number().int().nonnegative(), amount_minor: z.number().int().nonnegative(),
  input: z.record(z.string(), z.unknown()), bindings: VersionBindingsSchema
});
export const SubscriptionSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(), customer_id: z.string(),
  sku_version_id: z.string(), status: z.string(), starts_at: z.string().datetime().nullable().optional(),
  ends_at: z.string().datetime().nullable().optional(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const EntitlementSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(),
  subscription_id: z.string().optional(), key: z.string(), value: z.unknown(), status: z.string(),
  starts_at: z.string().datetime().nullable().optional(), ends_at: z.string().datetime().nullable().optional(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ExecutionOrderSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(), status: z.string(),
  provider_endpoint_id: z.string().optional(), external_id: z.string().optional(), idempotency_key: z.string(),
  created_by: z.string(), attempt: z.number().int().nonnegative(), input: z.record(z.string(), z.unknown()),
  output: z.record(z.string(), z.unknown()), error: z.record(z.string(), z.unknown()), bindings: VersionBindingsSchema,
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const DeliveryProjectSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(),
  execution_order_id: z.string(), mode: z.string(), status: z.string(), assignee: z.string().optional(),
  artifacts: z.array(z.record(z.string(), z.unknown())).default([]), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const UsageRecordSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(),
  execution_order_id: z.string(), meter_id: z.string(), idempotency_key: z.string(), created_by: z.string(),
  quantity: z.number().int().nonnegative(), occurred_at: z.string().datetime(), created_at: z.string().datetime()
});
export const ProviderCostSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(), execution_order_id: z.string(),
  provider_endpoint_id: z.string(), currency: z.string().length(3), idempotency_key: z.string(), created_by: z.string(),
  amount_minor: z.number().int().nonnegative(), created_at: z.string().datetime()
});
export const CustomerChargeSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string(), order_item_id: z.string(), execution_order_id: z.string(),
  price_book_id: z.string(), currency: z.string().length(3), status: z.string(), idempotency_key: z.string(),
  created_by: z.string(), amount_minor: z.number().int().nonnegative(), created_at: z.string().datetime()
});
export const WalletSchema = z.object({
  id: z.string(), tenant_id: z.string(), owner_type: z.string(), owner_id: z.string(), currency: z.string().length(3),
  status: z.string(), available_minor: z.number().int(), held_minor: z.number().int(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const LedgerAccountSchema = z.object({
  id: z.string(), tenant_id: z.string(), wallet_id: z.string().optional(), code: z.string(), name: z.string(),
  account_type: z.string(), purpose: z.string(), currency: z.string().length(3), balance_minor: z.number().int()
});
export const LedgerEntrySchema = z.object({
  id: z.string(), tenant_id: z.string(), transaction_id: z.string(), account_id: z.string(),
  direction: z.enum(["debit", "credit"]), currency: z.string().length(3), amount_minor: z.number().int().positive(),
  created_at: z.string().datetime()
});
export const LedgerTransactionSchema = z.object({
  id: z.string(), tenant_id: z.string(), idempotency_key: z.string(), transaction_type: z.string(),
  reference_type: z.string(), reference_id: z.string(), description: z.string(), reverses_transaction_id: z.string().optional(),
  created_by: z.string(), metadata: z.record(z.string(), z.unknown()), entries: z.array(LedgerEntrySchema), created_at: z.string().datetime()
});
export const HoldSchema = z.object({
  id: z.string(), tenant_id: z.string(), wallet_id: z.string(), order_id: z.string(), currency: z.string().length(3),
  amount_minor: z.number().int().positive(), captured_minor: z.number().int().nonnegative(), released_minor: z.number().int().nonnegative(),
  remaining_minor: z.number().int().nonnegative(), status: z.string(), ledger_transaction_id: z.string(), idempotency_key: z.string(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const RefundSchema = z.object({
  id: z.string(), tenant_id: z.string(), customer_charge_id: z.string(), wallet_id: z.string(), currency: z.string().length(3),
  amount_minor: z.number().int().positive(), reason: z.string(), status: z.string(), ledger_transaction_id: z.string(),
  idempotency_key: z.string(), created_by: z.string(), created_at: z.string().datetime()
});
export const CommissionSchema = z.object({
  id: z.string(), tenant_id: z.string(), customer_charge_id: z.string(), beneficiary_type: z.string(), beneficiary_id: z.string(),
  currency: z.string().length(3), amount_minor: z.number().int().positive(), settled_minor: z.number().int().nonnegative(), status: z.string(),
  payable_account_id: z.string(), ledger_transaction_id: z.string(), idempotency_key: z.string(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ProviderPayableSchema = z.object({
  id: z.string(), tenant_id: z.string(), provider_cost_id: z.string(), provider_id: z.string(), currency: z.string().length(3),
  amount_minor: z.number().int().positive(), settled_minor: z.number().int().nonnegative(), status: z.string(), payable_account_id: z.string(),
  ledger_transaction_id: z.string(), idempotency_key: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const SettlementSchema = z.object({
  id: z.string(), tenant_id: z.string(), source_type: z.string(), source_id: z.string(), beneficiary_type: z.string(),
  beneficiary_id: z.string(), currency: z.string().length(3), amount_minor: z.number().int().positive(), status: z.string(),
  ledger_transaction_id: z.string(), idempotency_key: z.string(), created_by: z.string(), created_at: z.string().datetime()
});
export const ReconciliationItemSchema = z.object({
  id: z.string(), tenant_id: z.string(), run_id: z.string(), reference_type: z.string(), reference_id: z.string(),
  currency: z.string().length(3), expected_minor: z.number().int().nonnegative(), actual_minor: z.number().int().nonnegative(),
  status: z.string(), created_at: z.string().datetime()
});
export const ReconciliationRunSchema = z.object({
  id: z.string(), tenant_id: z.string(), order_id: z.string().optional(), status: z.string(), checked_count: z.number().int().nonnegative(),
  discrepancy_count: z.number().int().nonnegative(), created_by: z.string(), idempotency_key: z.string(),
  started_at: z.string().datetime(), completed_at: z.string().datetime(), items: z.array(ReconciliationItemSchema)
});
export const FinanceOverviewSchema = z.object({
  wallets: z.array(WalletSchema), accounts: z.array(LedgerAccountSchema), transactions: z.array(LedgerTransactionSchema),
  holds: z.array(HoldSchema), refunds: z.array(RefundSchema), commissions: z.array(CommissionSchema),
  provider_payables: z.array(ProviderPayableSchema), settlements: z.array(SettlementSchema), reconciliation_runs: z.array(ReconciliationRunSchema)
});
export const OrderSchema = z.object({
  id: z.string(), tenant_id: z.string(), customer_id: z.string(), quote_version_id: z.string(), status: z.string(),
  currency: z.string().length(3), idempotency_key: z.string(), amount_minor: z.number().int().nonnegative(),
  version: z.number().int().positive(), bindings: VersionBindingsSchema, created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime(), items: z.array(OrderItemSchema).default([]),
  subscriptions: z.array(SubscriptionSchema).default([]), entitlements: z.array(EntitlementSchema).default([]),
  executions: z.array(ExecutionOrderSchema).default([]), deliveries: z.array(DeliveryProjectSchema).default([]),
  usage: z.array(UsageRecordSchema).default([]), provider_costs: z.array(ProviderCostSchema).default([]),
  customer_charges: z.array(CustomerChargeSchema).default([])
});

export const MarketSegmentSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), status: z.string(),
  definition: z.record(z.string(), z.unknown()), version: z.number().int().positive(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ICPDefinitionSchema = z.object({
  id: z.string(), tenant_id: z.string(), market_segment_id: z.string(), name: z.string(), status: z.string(),
  definition: z.record(z.string(), z.unknown()), version: z.number().int().positive(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const LeadSchema = z.object({
  id: z.string(), tenant_id: z.string(), market_segment_id: z.string(), icp_definition_id: z.string().optional(), name: z.string(),
  status: z.enum(["discovered", "enriched", "qualified", "proof_requested", "proof_ready", "approved_for_outreach", "contacted", "replied", "meeting", "proposal", "won", "lost", "suppressed"]),
  score: z.number().int().min(0).max(100), version: z.number().int().positive(), evidence: z.array(z.string()).optional(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const LeadEvidenceSchema = z.object({
  id: z.string(), tenant_id: z.string(), lead_id: z.string(), kind: z.string(), summary: z.string(),
  confidence: z.number().int().min(0).max(100), source_ref: z.string(), created_by: z.string(), created_at: z.string().datetime()
});
export const ContactSchema = z.object({
  id: z.string(), tenant_id: z.string(), lead_id: z.string(), channel: z.string(), value: z.string(), normalized_value: z.string(),
  status: z.string(), consent_status: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ProofTemplateSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), proof_type: z.enum(["report", "sample", "comparison", "prototype", "analysis", "audit", "simulation", "document", "media", "custom"]),
  workflow_version_id: z.string(), input_schema: z.record(z.string(), z.unknown()), output_schema: z.record(z.string(), z.unknown()),
  access_policy: z.record(z.string(), z.unknown()), retention_days: z.number().int().positive(), status: z.string(), version: z.number().int().positive(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ProofRequestSchema = z.object({
  id: z.string(), tenant_id: z.string(), lead_id: z.string(), deal_id: z.string().optional(), template_id: z.string(), status: z.string(),
  input: z.record(z.string(), z.unknown()), requested_by: z.string(), expires_at: z.string().datetime(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ProofInstanceSchema = z.object({
  id: z.string(), tenant_id: z.string(), proof_request_id: z.string(), status: z.string(), result: z.record(z.string(), z.unknown()),
  artifact_ref: z.string(), review_rationale: z.string(), generated_by: z.string(), reviewed_by: z.string().optional(),
  created_at: z.string().datetime(), reviewed_at: z.string().datetime().optional(), expires_at: z.string().datetime()
});
export const CampaignStepSchema = z.object({
  id: z.string(), tenant_id: z.string(), campaign_id: z.string(), position: z.number().int().positive(), kind: z.string(),
  definition: z.record(z.string(), z.unknown()), created_at: z.string().datetime()
});
export const CampaignApprovalSchema = z.object({
  id: z.string(), tenant_id: z.string(), campaign_id: z.string(), campaign_version: z.number().int().positive(),
  decision: z.enum(["approved", "rejected"]), rationale: z.string(), reviewed_by: z.string(), created_at: z.string().datetime()
});
export const CampaignSchema = z.object({
  id: z.string(), tenant_id: z.string(), market_segment_id: z.string().optional(), name: z.string(), channel: z.string(), purpose: z.string(),
  status: z.string(), version: z.number().int().positive(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime(),
  steps: z.array(CampaignStepSchema).default([]), approval: CampaignApprovalSchema.optional()
});
export const SuppressionEntrySchema = z.object({
  id: z.string(), tenant_id: z.string(), subject_type: z.string(), subject_id: z.string(), channel: z.string(), reason: z.string(), status: z.string(),
  source_ref: z.string(), created_by: z.string(), created_at: z.string().datetime(), released_at: z.string().datetime().optional()
});
export const OutreachMessageSchema = z.object({
  id: z.string(), tenant_id: z.string(), campaign_id: z.string(), campaign_step_id: z.string(), lead_id: z.string(), contact_id: z.string().optional(),
  status: z.enum(["planned", "blocked", "sent", "delivered", "replied", "bounced", "complained", "cancelled"]),
  content: z.record(z.string(), z.unknown()), block_reason: z.string().optional(), external_message_id: z.string().optional(),
  idempotency_key: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ConversationMessageSchema = z.object({
  id: z.string(), tenant_id: z.string(), conversation_id: z.string(), direction: z.enum(["inbound", "outbound", "system"]),
  status: z.string(), content: z.record(z.string(), z.unknown()), idempotency_key: z.string(), created_by: z.string(), created_at: z.string().datetime()
});
export const ConversationSchema = z.object({
  id: z.string(), tenant_id: z.string(), lead_id: z.string(), deal_id: z.string().optional(), channel: z.string(), status: z.string(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime(), last_message_at: z.string().datetime().optional(),
  messages: z.array(ConversationMessageSchema).default([])
});
export const DealSchema = z.object({
  id: z.string(), tenant_id: z.string(), lead_id: z.string(), name: z.string(), customer_id: z.string(),
  status: z.enum(["open", "proposal", "won", "lost", "cancelled"]), value_minor: z.number().int().nonnegative(), currency: z.string().length(3),
  version: z.number().int().positive(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime(), closed_at: z.string().datetime().optional()
});
export const ExperimentSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), entity_type: z.enum(["market_segment", "lead", "proof", "campaign", "deal"]), entity_id: z.string(),
  hypothesis: z.string(), status: z.enum(["draft", "running", "completed", "cancelled"]), allocation_basis_points: z.number().int().min(0).max(10000),
  metrics_definition: z.record(z.string(), z.unknown()), result: z.record(z.string(), z.unknown()), version: z.number().int().positive(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const GrowthOverviewSchema = z.object({
  segments: z.array(MarketSegmentSchema), icps: z.array(ICPDefinitionSchema), leads: z.array(LeadSchema), evidence: z.array(LeadEvidenceSchema),
  contacts: z.array(ContactSchema), proof_templates: z.array(ProofTemplateSchema), proof_requests: z.array(ProofRequestSchema), proof_instances: z.array(ProofInstanceSchema),
  campaigns: z.array(CampaignSchema), suppressions: z.array(SuppressionEntrySchema), outreach: z.array(OutreachMessageSchema),
  conversations: z.array(ConversationSchema), deals: z.array(DealSchema), experiments: z.array(ExperimentSchema)
});

export const ResellerLevelSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), rank: z.number().int().positive(),
  default_commission_bps: z.number().int().min(0).max(10000), status: z.string(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ResellerSchema = z.object({
  id: z.string(), tenant_id: z.string(), level_id: z.string().optional(), name: z.string(), status: z.string(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const AttributionRuleSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), priority: z.number().int().positive(),
  definition: z.record(z.string(), z.unknown()), status: z.string(), version: z.number().int().positive(),
  created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const LeadOwnershipSchema = z.object({
  id: z.string(), tenant_id: z.string(), lead_id: z.string(), reseller_id: z.string(), attribution_rule_id: z.string(),
  status: z.string(), protection_expires_at: z.string().datetime(), version: z.number().int().positive(),
  created_by: z.string(), acquired_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const CustomerOwnershipSchema = z.object({
  id: z.string(), tenant_id: z.string(), customer_id: z.string(), reseller_id: z.string(), source_lead_ownership_id: z.string().optional(),
  status: z.string(), protection_expires_at: z.string().datetime(), version: z.number().int().positive(),
  created_by: z.string(), acquired_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const TransferRequestSchema = z.object({
  id: z.string(), tenant_id: z.string(), ownership_type: z.enum(["lead", "customer"]), ownership_id: z.string(),
  from_reseller_id: z.string(), to_reseller_id: z.string(), status: z.string(), rationale: z.string(), requested_by: z.string(),
  reviewed_by: z.string().optional(), version: z.number().int().positive(), created_at: z.string().datetime(), reviewed_at: z.string().datetime().optional()
});
export const ConflictRecordSchema = z.object({
  id: z.string(), tenant_id: z.string(), ownership_type: z.string(), ownership_id: z.string(), claimant_reseller_ids: z.array(z.string()),
  status: z.string(), resolution: z.record(z.string(), z.unknown()), created_by: z.string(), resolved_by: z.string().optional(),
  created_at: z.string().datetime(), resolved_at: z.string().datetime().optional()
});
export const CommissionRuleSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), reseller_id: z.string().optional(), reseller_level_id: z.string().optional(),
  trigger_type: z.string(), basis_points: z.number().int().min(0).max(10000), status: z.string(), version: z.number().int().positive(),
  effective_from: z.string().datetime(), effective_until: z.string().datetime().optional(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const CommissionLockSchema = z.object({
  id: z.string(), tenant_id: z.string(), customer_charge_id: z.string(), reseller_id: z.string(), commission_rule_id: z.string(),
  commission_id: z.string().optional(), currency: z.string().length(3), amount_minor: z.number().int().positive(), status: z.string(),
  idempotency_key: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const SettlementCycleSchema = z.object({
  id: z.string(), tenant_id: z.string(), reseller_id: z.string(), name: z.string(), period_start: z.string().datetime(),
  period_end: z.string().datetime(), status: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const SupplierSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), status: z.string(), created_by: z.string(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const SupplierCapabilitySchema = z.object({
  id: z.string(), tenant_id: z.string(), supplier_id: z.string(), capability_id: z.string(), status: z.string(), created_at: z.string().datetime()
});
export const ProviderSupplierBindingSchema = z.object({ provider_id: z.string(), provider_name: z.string(), supplier_id: z.string() });
export const SupplierContractSchema = z.object({
  id: z.string(), tenant_id: z.string(), supplier_id: z.string(), provider_id: z.string().optional(), name: z.string(), status: z.string(),
  currency: z.string().length(3), terms: z.record(z.string(), z.unknown()), version: z.number().int().positive(),
  starts_at: z.string().datetime().optional(), ends_at: z.string().datetime().optional(), created_by: z.string(), approved_by: z.string().optional(),
  created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const SupplierRateSchema = z.object({
  id: z.string(), tenant_id: z.string(), contract_id: z.string(), capability_id: z.string(), unit: z.string(), rate_minor: z.number().int().nonnegative(),
  version: z.number().int().positive(), status: z.string(), created_by: z.string(), created_at: z.string().datetime()
});
export const SupplierQualityRecordSchema = z.object({
  id: z.string(), tenant_id: z.string(), supplier_id: z.string(), provider_id: z.string().optional(), provider_endpoint_id: z.string().optional(),
  metric: z.string(), score_bps: z.number().int().min(0).max(10000), evidence: z.record(z.string(), z.unknown()),
  period_start: z.string().datetime(), period_end: z.string().datetime(), created_by: z.string(), created_at: z.string().datetime()
});
export const DeveloperSchema = z.object({
  id: z.string(), tenant_id: z.string(), name: z.string(), status: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const PublisherSchema = z.object({
  id: z.string(), tenant_id: z.string(), developer_id: z.string(), name: z.string(), status: z.string(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const ListingVersionSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), version: z.number().int().positive(), capability_manifest: z.record(z.string(), z.unknown()),
  permission_manifest: z.record(z.string(), z.unknown()), content_ref: z.string(), checksum: z.string(), created_by: z.string(), created_at: z.string().datetime()
});
export const ListingSchema = z.object({
  id: z.string(), tenant_id: z.string(), publisher_id: z.string(), name: z.string(), listing_type: z.string(), status: z.string(),
  version: z.number().int().positive(), created_by: z.string(), created_at: z.string().datetime(), updated_at: z.string().datetime(), versions: z.array(ListingVersionSchema)
});
export const ListingReviewSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), listing_version_id: z.string(), review_type: z.string(), decision: z.string(),
  rationale: z.string(), reviewed_by: z.string(), created_at: z.string().datetime()
});
export const SandboxRunSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_version_id: z.string(), status: z.string(), created_by: z.string(),
  policy: z.record(z.string(), z.unknown()), result: z.record(z.string(), z.unknown()), started_at: z.string().datetime(), completed_at: z.string().datetime().optional()
});
export const QualityScoreSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_version_id: z.string(), score_bps: z.number().int().min(0).max(10000),
  dimensions: z.record(z.string(), z.unknown()), created_by: z.string(), created_at: z.string().datetime()
});
export const IncidentRecordSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), severity: z.string(), summary: z.string(), status: z.string(), created_by: z.string(),
  created_at: z.string().datetime(), resolved_at: z.string().datetime().optional()
});
export const RevenueShareRuleSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), publisher_id: z.string(), currency: z.string().length(3), status: z.string(),
  created_by: z.string(), basis_points: z.number().int(), version: z.number().int(), created_at: z.string().datetime()
});
export const PayoutReserveSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), publisher_id: z.string(), currency: z.string().length(3), status: z.string(),
  reference_type: z.string(), reference_id: z.string(), created_by: z.string(), amount_minor: z.number().int(), created_at: z.string().datetime(), updated_at: z.string().datetime()
});
export const MarketplaceDisputeSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), claimant_type: z.string(), claimant_id: z.string(), reason: z.string(), status: z.string(),
  created_by: z.string(), resolved_by: z.string().optional(), resolution: z.record(z.string(), z.unknown()), created_at: z.string().datetime(), resolved_at: z.string().datetime().optional()
});
export const TakedownSchema = z.object({
  id: z.string(), tenant_id: z.string(), listing_id: z.string(), reason: z.string(), status: z.string(), requested_by: z.string(), reviewed_by: z.string().optional(),
  created_at: z.string().datetime(), reviewed_at: z.string().datetime().optional()
});
export const MarketplaceOverviewSchema = z.object({
  developers: z.array(DeveloperSchema), publishers: z.array(PublisherSchema), listings: z.array(ListingSchema), reviews: z.array(ListingReviewSchema),
  sandbox_runs: z.array(SandboxRunSchema), quality_scores: z.array(QualityScoreSchema), incidents: z.array(IncidentRecordSchema),
  revenue_share_rules: z.array(RevenueShareRuleSchema), payout_reserves: z.array(PayoutReserveSchema), disputes: z.array(MarketplaceDisputeSchema), takedowns: z.array(TakedownSchema)
});
export const ChannelOverviewSchema = z.object({
  reseller_levels: z.array(ResellerLevelSchema), resellers: z.array(ResellerSchema), attribution_rules: z.array(AttributionRuleSchema),
  lead_ownerships: z.array(LeadOwnershipSchema), customer_ownerships: z.array(CustomerOwnershipSchema), transfer_requests: z.array(TransferRequestSchema), conflicts: z.array(ConflictRecordSchema),
  commission_rules: z.array(CommissionRuleSchema), commission_locks: z.array(CommissionLockSchema), settlement_cycles: z.array(SettlementCycleSchema),
  suppliers: z.array(SupplierSchema), supplier_capabilities: z.array(SupplierCapabilitySchema), provider_bindings: z.array(ProviderSupplierBindingSchema),
  supplier_contracts: z.array(SupplierContractSchema), supplier_rates: z.array(SupplierRateSchema), supplier_quality: z.array(SupplierQualityRecordSchema),
  provider_payables: z.array(ProviderPayableSchema), reseller_settlements: z.array(SettlementSchema), supplier_settlements: z.array(SettlementSchema), marketplace: MarketplaceOverviewSchema
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
export type VersionBindings = z.infer<typeof VersionBindingsSchema>;
export type QuoteItem = z.infer<typeof QuoteItemSchema>;
export type QuoteVersion = z.infer<typeof QuoteVersionSchema>;
export type Quote = z.infer<typeof QuoteSchema>;
export type OrderItem = z.infer<typeof OrderItemSchema>;
export type Subscription = z.infer<typeof SubscriptionSchema>;
export type Entitlement = z.infer<typeof EntitlementSchema>;
export type ExecutionOrder = z.infer<typeof ExecutionOrderSchema>;
export type DeliveryProject = z.infer<typeof DeliveryProjectSchema>;
export type UsageRecord = z.infer<typeof UsageRecordSchema>;
export type ProviderCost = z.infer<typeof ProviderCostSchema>;
export type CustomerCharge = z.infer<typeof CustomerChargeSchema>;
export type Wallet = z.infer<typeof WalletSchema>;
export type LedgerAccount = z.infer<typeof LedgerAccountSchema>;
export type LedgerEntry = z.infer<typeof LedgerEntrySchema>;
export type LedgerTransaction = z.infer<typeof LedgerTransactionSchema>;
export type Hold = z.infer<typeof HoldSchema>;
export type Refund = z.infer<typeof RefundSchema>;
export type Commission = z.infer<typeof CommissionSchema>;
export type ProviderPayable = z.infer<typeof ProviderPayableSchema>;
export type Settlement = z.infer<typeof SettlementSchema>;
export type ReconciliationItem = z.infer<typeof ReconciliationItemSchema>;
export type ReconciliationRun = z.infer<typeof ReconciliationRunSchema>;
export type FinanceOverview = z.infer<typeof FinanceOverviewSchema>;
export type Order = z.infer<typeof OrderSchema>;
export type MarketSegment = z.infer<typeof MarketSegmentSchema>;
export type ICPDefinition = z.infer<typeof ICPDefinitionSchema>;
export type Lead = z.infer<typeof LeadSchema>;
export type LeadEvidence = z.infer<typeof LeadEvidenceSchema>;
export type Contact = z.infer<typeof ContactSchema>;
export type ProofTemplate = z.infer<typeof ProofTemplateSchema>;
export type ProofRequest = z.infer<typeof ProofRequestSchema>;
export type ProofInstance = z.infer<typeof ProofInstanceSchema>;
export type CampaignStep = z.infer<typeof CampaignStepSchema>;
export type CampaignApproval = z.infer<typeof CampaignApprovalSchema>;
export type Campaign = z.infer<typeof CampaignSchema>;
export type SuppressionEntry = z.infer<typeof SuppressionEntrySchema>;
export type OutreachMessage = z.infer<typeof OutreachMessageSchema>;
export type ConversationMessage = z.infer<typeof ConversationMessageSchema>;
export type Conversation = z.infer<typeof ConversationSchema>;
export type Deal = z.infer<typeof DealSchema>;
export type Experiment = z.infer<typeof ExperimentSchema>;
export type GrowthOverview = z.infer<typeof GrowthOverviewSchema>;
export type ResellerLevel = z.infer<typeof ResellerLevelSchema>;
export type Reseller = z.infer<typeof ResellerSchema>;
export type AttributionRule = z.infer<typeof AttributionRuleSchema>;
export type LeadOwnership = z.infer<typeof LeadOwnershipSchema>;
export type CustomerOwnership = z.infer<typeof CustomerOwnershipSchema>;
export type TransferRequest = z.infer<typeof TransferRequestSchema>;
export type CommissionRule = z.infer<typeof CommissionRuleSchema>;
export type CommissionLock = z.infer<typeof CommissionLockSchema>;
export type SettlementCycle = z.infer<typeof SettlementCycleSchema>;
export type Supplier = z.infer<typeof SupplierSchema>;
export type SupplierCapability = z.infer<typeof SupplierCapabilitySchema>;
export type SupplierContract = z.infer<typeof SupplierContractSchema>;
export type SupplierRate = z.infer<typeof SupplierRateSchema>;
export type SupplierQualityRecord = z.infer<typeof SupplierQualityRecordSchema>;
export type Developer = z.infer<typeof DeveloperSchema>;
export type Publisher = z.infer<typeof PublisherSchema>;
export type ListingVersion = z.infer<typeof ListingVersionSchema>;
export type Listing = z.infer<typeof ListingSchema>;
export type ListingReview = z.infer<typeof ListingReviewSchema>;
export type SandboxRun = z.infer<typeof SandboxRunSchema>;
export type QualityScore = z.infer<typeof QualityScoreSchema>;
export type MarketplaceDispute = z.infer<typeof MarketplaceDisputeSchema>;
export type Takedown = z.infer<typeof TakedownSchema>;
export type MarketplaceOverview = z.infer<typeof MarketplaceOverviewSchema>;
export type ChannelOverview = z.infer<typeof ChannelOverviewSchema>;

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
