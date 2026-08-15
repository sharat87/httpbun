import os

import pytest


def _get_base_url(suffix: str = "") -> str:
    """Get base URL from environment, with optional suffix for SDK-specific paths."""
    base = os.getenv("BASE_URL", "http://localhost:3090")
    if suffix and not base.endswith(suffix):
        return base + suffix
    return base


@pytest.fixture
def base_url() -> str:
    """Default base_url for backward compatibility."""
    return _get_base_url()


@pytest.fixture
def openai_base_url() -> str:
    """Base URL for OpenAI SDK - needs /v1 suffix."""
    return _get_base_url("/v1")


@pytest.fixture
def anthropic_base_url() -> str:
    """Base URL for Anthropic SDK - SDK automatically adds /v1."""
    return _get_base_url()
