from typing import Protocol
from .models import OpportunityScoreSuggestion, SignalInput


class IntelligenceAdapter(Protocol):
    async def normalize_signal(self, signal: SignalInput) -> dict: ...
    async def suggest_score(self, opportunity_id: str, evidence: list[dict]) -> OpportunityScoreSuggestion: ...


def assert_advisory_output(suggestion: OpportunityScoreSuggestion) -> None:
    if not suggestion.advisory_only:
        raise ValueError("intelligence output must remain advisory")

