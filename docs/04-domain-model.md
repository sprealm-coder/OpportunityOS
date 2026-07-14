# Domain Model

The aggregate chain is Source -> Signal -> Opportunity -> IncubationProject -> BusinessBlueprint -> ProductVersion -> SKUVersion -> MarketSegment -> ICPDefinition -> Lead -> ProofRequest -> Campaign -> Conversation -> Deal -> Quote -> Order -> Execution -> Usage/Cost/Charge -> Ledger -> Settlement -> OutcomeFeedback.

Definitions and runtime facts are separate. Every commercial binding points to immutable versions. Monetary amounts use signed 64-bit minor units and ISO currency codes.

Critical distinctions are enforced in storage and commands: Lead is not Customer, Deal is not Order, Campaign is not delivery, Proof is not a product-specific demo, Contact consent is not permission to bypass suppression, and Experiment results do not mutate pricing or ledger facts. Quote remains owned by the transaction domain; Growth stores only its canonical Deal reference.
