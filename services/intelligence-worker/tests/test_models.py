from datetime import UTC, datetime
import pytest
from opportunity_intelligence.adapter import assert_advisory_output
from opportunity_intelligence.models import OpportunityScoreSuggestion


def test_ai_score_is_advisory() -> None:
    suggestion = OpportunityScoreSuggestion(
        opportunity_id="test-opportunity",
        score=80,
        factors=[],
        model_adapter="test-adapter",
        generated_at=datetime.now(UTC),
    )
    assert_advisory_output(suggestion)
    with pytest.raises(ValueError):
        assert_advisory_output(suggestion.model_copy(update={"advisory_only": False}))

