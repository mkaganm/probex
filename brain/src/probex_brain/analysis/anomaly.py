"""AI-powered anomaly classifier for API metrics."""

from __future__ import annotations

import logging

from probex_brain.ai.prompts import ANOMALY_CLASSIFY_SYSTEM, ANOMALY_CLASSIFY_USER_TEMPLATE
from probex_brain.ai.router import AIRouter
from probex_brain.models.schemas import AnomalyClassifyRequest, AnomalyClassifyResponse

logger = logging.getLogger(__name__)


class AnomalyClassifier:
    """Classifies and explains API metric anomalies using AI."""

    def __init__(self, router: AIRouter) -> None:
        self.router = router

    async def classify(self, request: AnomalyClassifyRequest) -> AnomalyClassifyResponse:
        prompt = ANOMALY_CLASSIFY_USER_TEMPLATE.format(
            endpoint_id=request.endpoint_id,
            metric=request.metric,
            expected=request.expected,
            actual=request.actual,
            z_score=request.z_score,
            description=request.description or "No additional context provided.",
        )

        data, _model_used = await self.router.generate_json(
            prompt, system=ANOMALY_CLASSIFY_SYSTEM, max_tokens=2048
        )

        return AnomalyClassifyResponse(
            classification=data.get("classification", "normal"),
            severity=data.get("severity", "medium"),
            explanation=data.get("explanation", ""),
            recommended_action=data.get("recommended_action", ""),
        )
