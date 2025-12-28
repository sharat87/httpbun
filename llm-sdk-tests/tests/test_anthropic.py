#!/usr/bin/env python3
"""
Strict pytest tests for Anthropic SDK with httpbun.com/llm endpoint.
Any deviation in the response structure or values will cause test failures.
"""

import json
import re

from anthropic import Anthropic, AsyncAnthropic
from anthropic.types import Message


def test_anthropic_sync_response_structure(base_url: str):
    """Test the synchronous Anthropic client with httpbun endpoint - strict response structure validation."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    response = client.messages.create(
        model="claude-3-5-sonnet-20241022",
        max_tokens=1024,
        messages=[{"role": "user", "content": "Hello"}],
    )

    # Assert response is the correct type
    assert isinstance(response, Message)

    # Assert all top-level fields exist and have exact values/types
    assert hasattr(response, "id")
    assert isinstance(response.id, str)
    assert response.id.startswith("msg-")
    assert len(response.id) == 28  # 'msg-' + 24 chars
    assert re.match(r"^msg-[a-f0-9]{24}$", response.id)

    assert response.type == "message"
    assert response.model == "claude-3-5-sonnet-20241022"
    assert response.role == "assistant"
    assert response.stop_reason == "end_turn"
    assert response.stop_sequence is None

    # Assert content structure
    assert hasattr(response, "content")
    assert isinstance(response.content, list)
    assert len(response.content) == 1

    content_block = response.content[0]
    assert content_block.type == "text"
    assert (
        content_block.text
        == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
    )

    # Assert usage structure
    assert hasattr(response, "usage")
    usage = response.usage
    assert usage is not None
    assert usage.input_tokens == 3
    assert usage.output_tokens == 33


def test_anthropic_sync_exact_field_count(base_url: str):
    """Test that the response has exactly the expected fields, no more, no less."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    response = client.messages.create(
        model="claude-3-5-sonnet-20241022",
        max_tokens=1024,
        messages=[{"role": "user", "content": "Hello"}],
    )

    # Check exact fields in response model dump
    response_dict = response.model_dump()
    expected_keys = {
        "id",
        "type",
        "role",
        "content",
        "model",
        "stop_reason",
        "stop_sequence",
        "usage",
    }
    assert set(response_dict.keys()) == expected_keys

    # Check exact fields in content block
    content_dict = response_dict["content"][0]
    expected_content_keys = {"type", "text", "citations"}
    assert set(content_dict.keys()) == expected_content_keys

    # Check exact fields in usage
    usage_dict = response_dict["usage"]
    expected_usage_keys = {
        "input_tokens",
        "output_tokens",
        "cache_creation",
        "cache_creation_input_tokens",
        "cache_read_input_tokens",
        "server_tool_use",
        "service_tier",
    }
    assert set(usage_dict.keys()) == expected_usage_keys


async def test_anthropic_async_response(base_url: str):
    """Test the async Anthropic client with httpbun endpoint."""
    client = AsyncAnthropic(base_url=base_url, api_key="dummy-key")

    try:
        response = await client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=1024,
            messages=[{"role": "user", "content": "Hello"}],
        )

        # Assert response type and content
        assert isinstance(response, Message)
        assert response.model == "claude-3-5-sonnet-20241022"
        assert response.type == "message"
        assert response.role == "assistant"
        assert response.stop_reason == "end_turn"

        # Assert the exact message content
        assert len(response.content) == 1
        assert (
            response.content[0].text
            == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
        )

        # Assert token usage
        assert response.usage is not None
        assert response.usage.input_tokens == 3
        assert response.usage.output_tokens == 33

    finally:
        await client.close()


def test_anthropic_multiple_requests_consistent(base_url: str):
    """Test that multiple requests return consistent structure."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    responses = []
    for _ in range(3):
        response = client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=1024,
            messages=[{"role": "user", "content": "Hello"}],
        )
        responses.append(response)

    # All responses should have the same structure and content
    for response in responses:
        assert response.model == "claude-3-5-sonnet-20241022"
        assert response.type == "message"
        assert (
            response.content[0].text
            == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
        )
        assert response.usage is not None
        assert response.usage.input_tokens == 3
        assert response.usage.output_tokens == 33

    # IDs should be different
    ids = [r.id for r in responses]
    assert len(set(ids)) == 3  # All IDs should be unique


def test_anthropic_error_handling(base_url: str):
    """Test error handling with an invalid model name."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    # httpbun might accept any model name, but let's test the response
    response = client.messages.create(
        model="invalid-model-name",
        max_tokens=1024,
        messages=[{"role": "user", "content": "Hello"}],
    )

    # Even with invalid model, httpbun returns the same structure
    assert response.model == "invalid-model-name"  # httpbun echoes the model name
    assert (
        response.content[0].text
        == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
    )


def test_anthropic_different_message(base_url: str):
    """Test with a different message to ensure httpbun returns the same mock response."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    response = client.messages.create(
        model="claude-3-5-sonnet-20241022",
        max_tokens=1024,
        messages=[{"role": "user", "content": "This is a different message"}],
    )

    # httpbun returns the same mock response regardless of input
    assert (
        response.content[0].text
        == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
    # But token counts might be different
    assert response.usage is not None
    assert response.usage.input_tokens > 3  # Should be more than "Hello"


def test_anthropic_conversation_history(base_url: str):
    """Test with conversation history."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    response = client.messages.create(
        model="claude-3-5-sonnet-20241022",
        max_tokens=1024,
        messages=[
            {"role": "user", "content": "Hello"},
            {"role": "assistant", "content": "Hi there!"},
            {"role": "user", "content": "How are you?"},
        ],
    )

    # httpbun still returns the same mock response
    assert (
        response.content[0].text
        == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
    # Token count should reflect all messages
    assert response.usage is not None
    assert response.usage.input_tokens > 3


def test_anthropic_response_serialization(base_url: str):
    """Test that the response can be properly serialized and deserialized."""
    client = Anthropic(base_url=base_url, api_key="dummy-key")

    response = client.messages.create(
        model="claude-3-5-sonnet-20241022",
        max_tokens=1024,
        messages=[{"role": "user", "content": "Hello"}],
    )

    # Test model_dump()
    response_dict = response.model_dump()
    assert isinstance(response_dict, dict)
    assert response_dict["model"] == "claude-3-5-sonnet-20241022"
    assert (
        response_dict["content"][0]["text"]
        == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
    )

    # Test model_dump_json()
    response_json = response.model_dump_json()
    parsed = json.loads(response_json)
    assert parsed["model"] == "claude-3-5-sonnet-20241022"
    assert (
        parsed["content"][0]["text"]
        == "This is a mock Anthropic messages API response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
