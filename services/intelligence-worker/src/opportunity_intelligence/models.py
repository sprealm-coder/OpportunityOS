from datetime import datetime
from typing import Any
from pydantic import BaseModel, ConfigDict, Field


class SignalInput(BaseModel):
    model_config = ConfigDict(extra="forbid")
    tenant_id: str = Field(min_length=1)
    source_id: str = Field(min_length=1)
    raw_text: str = Field(min_length=1, max_length=100_000)
    metadata: dict[str, Any] = Field(default_factory=dict)


class ScoreFactor(BaseModel):
    name: str
    score: int = Field(ge=0, le=100)
    evidence: list[str] = Field(default_factory=list)


class OpportunityScoreSuggestion(BaseModel):
    opportunity_id: str
    score: int = Field(ge=0, le=100)
    factors: list[ScoreFactor]
    model_adapter: str
    generated_at: datetime
    advisory_only: bool = True

