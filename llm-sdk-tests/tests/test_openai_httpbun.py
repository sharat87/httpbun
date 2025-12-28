#!/usr/bin/env python3
"""
Strict pytest tests for OpenAI SDK with httpbun.com/llm endpoint.
Any deviation in the response structure or values will cause test failures.
"""

import re
from openai import OpenAI, AsyncOpenAI
from openai.types.chat import ChatCompletion


def test_openai_sync_response_structure(base_url: str):
    """Test the synchronous OpenAI client with httpbun endpoint - strict response structure validation."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    response = client.chat.completions.create(
        model="gpt-5-nano", messages=[{"role": "user", "content": "Hello"}]
    )

    # Assert response is the correct type
    assert isinstance(response, ChatCompletion)

    # Assert all top-level fields exist and have exact values/types
    assert hasattr(response, "id")
    assert isinstance(response.id, str)
    assert response.id.startswith("chatcmpl-")
    assert len(response.id) == 33  # 'chatcmpl-' + 24 chars
    assert re.match(r"^chatcmpl-[a-f0-9]{24}$", response.id)

    assert response.object == "chat.completion"
    assert response.model == "gpt-5-nano"
    assert isinstance(response.created, int)
    assert response.created > 0

    assert response.service_tier is None
    assert response.system_fingerprint is None

    # Assert choices structure
    assert hasattr(response, "choices")
    assert isinstance(response.choices, list)
    assert len(response.choices) == 1

    choice = response.choices[0]
    assert choice.index == 0
    assert choice.finish_reason == "stop"
    assert choice.logprobs is None

    # Assert message structure
    message = choice.message
    assert message.role == "assistant"
    assert (
        message.content
        == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
    assert message.refusal is None
    assert message.annotations is None
    assert message.audio is None
    assert message.function_call is None
    assert message.tool_calls is None

    # Assert usage structure
    assert hasattr(response, "usage")
    usage = response.usage
    assert usage is not None
    assert usage.prompt_tokens == 3
    assert usage.completion_tokens == 29
    assert usage.total_tokens == 32
    assert usage.completion_tokens_details is None
    assert usage.prompt_tokens_details is None


def test_openai_sync_exact_field_count(base_url: str):
    """Test that the response has exactly the expected fields, no more, no less."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    response = client.chat.completions.create(
        model="gpt-5-nano", messages=[{"role": "user", "content": "Hello"}]
    )

    # Check exact fields in response model dump
    response_dict = response.model_dump()
    expected_keys = {
        "id",
        "choices",
        "created",
        "model",
        "object",
        "service_tier",
        "system_fingerprint",
        "usage",
    }
    assert set(response_dict.keys()) == expected_keys

    # Check exact fields in choice
    choice_dict = response_dict["choices"][0]
    expected_choice_keys = {"finish_reason", "index", "logprobs", "message"}
    assert set(choice_dict.keys()) == expected_choice_keys

    # Check exact fields in message
    message_dict = choice_dict["message"]
    expected_message_keys = {
        "content",
        "refusal",
        "role",
        "annotations",
        "audio",
        "function_call",
        "tool_calls",
    }
    assert set(message_dict.keys()) == expected_message_keys

    # Check exact fields in usage
    usage_dict = response_dict["usage"]
    expected_usage_keys = {
        "completion_tokens",
        "prompt_tokens",
        "total_tokens",
        "completion_tokens_details",
        "prompt_tokens_details",
    }
    assert set(usage_dict.keys()) == expected_usage_keys


async def test_openai_async_response(base_url: str):
    """Test the async OpenAI client with httpbun endpoint."""
    client = AsyncOpenAI(base_url=base_url, api_key="dummy-key")

    try:
        response = await client.chat.completions.create(
            model="gpt-5-nano", messages=[{"role": "user", "content": "Hello"}]
        )

        # Assert response type and content
        assert isinstance(response, ChatCompletion)
        assert response.model == "gpt-5-nano"
        assert response.object == "chat.completion"

        # Assert the exact message content
        assert len(response.choices) == 1
        assert (
            response.choices[0].message.content
            == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
        )
        assert response.choices[0].message.role == "assistant"
        assert response.choices[0].finish_reason == "stop"

        # Assert token usage
        assert response.usage is not None
        assert response.usage is not None
        assert response.usage.prompt_tokens == 3
        assert response.usage.completion_tokens == 29
        assert response.usage.total_tokens == 32

    finally:
        await client.close()


def test_openai_multiple_requests_consistent(base_url: str):
    """Test that multiple requests return consistent structure."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    responses = []
    for _ in range(3):
        response = client.chat.completions.create(
            model="gpt-5-nano", messages=[{"role": "user", "content": "Hello"}]
        )
        responses.append(response)

    # All responses should have the same structure and content
    for response in responses:
        assert response.model == "gpt-5-nano"
        assert response.object == "chat.completion"
        assert (
            response.choices[0].message.content
            == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
        )
        assert response.usage is not None
        assert response.usage.prompt_tokens == 3
        assert response.usage.completion_tokens == 29
        assert response.usage.total_tokens == 32

    # IDs should be different
    ids = [r.id for r in responses]
    assert len(set(ids)) == 3  # All IDs should be unique


def test_openai_error_handling(base_url: str):
    """Test error handling with an invalid model name."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    # httpbun might accept any model name, but let's test the response
    response = client.chat.completions.create(
        model="invalid-model-name", messages=[{"role": "user", "content": "Hello"}]
    )

    # Even with invalid model, httpbun returns the same structure
    assert response.model == "invalid-model-name"  # httpbun echoes the model name
    assert (
        response.choices[0].message.content
        == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
    )


def test_openai_different_message(base_url: str):
    """Test with a different message to ensure httpbun returns the same mock response."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    response = client.chat.completions.create(
        model="gpt-5-nano",
        messages=[{"role": "user", "content": "This is a different message"}],
    )

    # httpbun returns the same mock response regardless of input
    assert (
        response.choices[0].message.content
        == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
    # But token counts might be different
    assert response.usage is not None
    assert response.usage.prompt_tokens > 3  # Should be more than "Hello"


def test_openai_conversation_history(base_url: str):
    """Test with conversation history."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    response = client.chat.completions.create(
        model="gpt-5-nano",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Hello"},
            {"role": "assistant", "content": "Hi there!"},
            {"role": "user", "content": "How are you?"},
        ],
    )

    # httpbun still returns the same mock response
    assert (
        response.choices[0].message.content
        == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
    # Token count should reflect all messages
    assert response.usage is not None
    assert response.usage.prompt_tokens > 3


def test_openai_response_serialization(base_url: str):
    """Test that the response can be properly serialized and deserialized."""
    client = OpenAI(base_url=base_url, api_key="dummy-key")

    response = client.chat.completions.create(
        model="gpt-5-nano", messages=[{"role": "user", "content": "Hello"}]
    )

    # Test model_dump()
    response_dict = response.model_dump()
    assert isinstance(response_dict, dict)
    assert response_dict["model"] == "gpt-5-nano"
    assert (
        response_dict["choices"][0]["message"]["content"]
        == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
    )

    # Test model_dump_json()
    import json

    response_json = response.model_dump_json()
    parsed = json.loads(response_json)
    assert parsed["model"] == "gpt-5-nano"
    assert (
        parsed["choices"][0]["message"]["content"]
        == "This is a mock chat response from httpbun. I received your messages and I'm responding with this placeholder text."
    )
