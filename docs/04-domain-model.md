# Domain Model

The aggregate chain is Source -> Signal -> Opportunity -> IncubationProject -> BusinessBlueprint -> ProductVersion -> SKUVersion -> Order -> Execution -> Usage/Cost/Charge -> Ledger -> Settlement -> OutcomeFeedback.

Definitions and runtime facts are separate. Every commercial binding points to immutable versions. Monetary amounts use signed 64-bit minor units and ISO currency codes.

