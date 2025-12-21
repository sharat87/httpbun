#!/usr/bin/env python3
"""
Strict pytest tests for OpenAI Responses API with httpbun.com/llm endpoint.
Any deviation in the response structure or values will cause test failures.
"""

import json
from typing import cast

import pytest
from openai import AsyncOpenAI, OpenAI
from openai.types.responses import Response
from openai.types.responses.response_input_param import ResponseInputParam
from openai.types.responses.response_output_message import ResponseOutputMessage
from openai.types.responses.response_output_text import ResponseOutputText

API_KEY = "dummy-key"
MODEL_NAME = "gpt-5-nano"
EXPECTED_OUTPUT_TEXT = (
    "This is a mock responses API response from httpbun. "
    "I received your input and I'm responding with this placeholder text."
)


def test_openai_responses_sync_response_structure(base_url: str):
    """Test the synchronous OpenAI client with responses endpoint - strict response structure validation."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    response = client.responses.create(model=MODEL_NAME, input="Hello")

    assert isinstance(response, Response)
    assert isinstance(response.id, str)
    assert response.id.startswith("resp")
    assert len(response.id) > 8

    assert response.object == "response"
    assert response.model == MODEL_NAME
    assert isinstance(response.created_at, (int, float))
    assert response.created_at > 0

    assert response.status == "completed"
    assert response.error is None
    assert response.service_tier is None

    assert isinstance(response.output, list)
    assert len(response.output) == 1

    output_message = cast(ResponseOutputMessage, response.output[0])
    assert output_message.type == "message"
    assert output_message.role == "assistant"
    assert output_message.status == "completed"
    assert isinstance(output_message.content, list)
    assert len(output_message.content) == 1

    content_block = cast(ResponseOutputText, output_message.content[0])
    assert content_block.type == "output_text"
    assert content_block.text == EXPECTED_OUTPUT_TEXT
    assert isinstance(content_block.annotations, list)
    assert response.output_text == EXPECTED_OUTPUT_TEXT

    assert response.usage is not None
    usage = response.usage
    assert usage.input_tokens == 3
    assert usage.output_tokens == 29
    assert usage.total_tokens == 32
    assert usage.input_tokens_details.cached_tokens == 0
    assert usage.output_tokens_details.reasoning_tokens == 0


def test_openai_responses_sync_exact_field_count(base_url: str):
    """Test that the response has exactly the expected fields, no more, no less."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    response = client.responses.create(model=MODEL_NAME, input="Hello")

    response_dict = response.model_dump(exclude={"output_text"}) # output_text is an SDK-only shortcut thingy
    expected_keys = set(Response.__annotations__.keys())
    assert set(response_dict.keys()) == expected_keys

    output_dict = response_dict["output"][0]
    expected_output_keys = {"id", "content", "role", "status", "type"}
    assert set(output_dict.keys()) == expected_output_keys

    content_dict = output_dict["content"][0]
    expected_content_keys = {"type", "text", "annotations", "logprobs"}
    assert set(content_dict.keys()) == expected_content_keys

    usage_dict = response_dict["usage"]
    expected_usage_keys = {
        "input_tokens",
        "output_tokens",
        "total_tokens",
        "input_tokens_details",
        "output_tokens_details",
    }
    assert set(usage_dict.keys()) == expected_usage_keys

    input_details = usage_dict["input_tokens_details"]
    assert set(input_details.keys()) == {"cached_tokens"}

    output_details = usage_dict["output_tokens_details"]
    assert set(output_details.keys()) == {"reasoning_tokens"}


async def test_openai_responses_async_response(base_url: str):
    """Test the async OpenAI client with responses endpoint."""
    client = AsyncOpenAI(base_url=base_url, api_key=API_KEY)

    try:
        response = await client.responses.create(model=MODEL_NAME, input="Hello")

        assert isinstance(response, Response)
        assert response.model == MODEL_NAME
        assert response.object == "response"
        assert response.status == "completed"

        output_message = cast(ResponseOutputMessage, response.output[0])
        content_block = cast(ResponseOutputText, output_message.content[0])
        assert content_block.text == EXPECTED_OUTPUT_TEXT
        assert content_block.type == "output_text"
        assert response.output_text == EXPECTED_OUTPUT_TEXT

        assert response.usage is not None
        assert response.usage.input_tokens == 3
        assert response.usage.output_tokens == 29
        assert response.usage.total_tokens == 32
    finally:
        await client.close()


def test_openai_responses_multiple_requests_consistent(base_url: str):
    """Test that multiple requests return consistent structure."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    responses = []
    for _ in range(3):
        responses.append(client.responses.create(model=MODEL_NAME, input="Hello"))

    for response in responses:
        assert response.model == MODEL_NAME
        assert response.object == "response"
        assert response.status == "completed"
        assert response.output_text == EXPECTED_OUTPUT_TEXT
        assert response.usage is not None
        assert response.usage.input_tokens == 3
        assert response.usage.output_tokens == 29
        assert response.usage.total_tokens == 32

    ids = [resp.id for resp in responses]
    assert len(set(ids)) == len(ids)


def test_openai_responses_error_handling(base_url: str):
    """Test error handling with an invalid model name."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    response = client.responses.create(model="invalid-model-name", input="Hello")

    assert response.model == "invalid-model-name"
    assert response.output_text == EXPECTED_OUTPUT_TEXT


def test_openai_responses_different_input(base_url: str):
    """Test with a different input to ensure httpbun returns the same mock response."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    message = "This is a different input"
    response = client.responses.create(model=MODEL_NAME, input=message)

    assert response.output_text == EXPECTED_OUTPUT_TEXT
    assert response.usage is not None
    assert response.usage.input_tokens > 3


def test_openai_responses_conversation_style_input(base_url: str):
    """Test with conversation style input to ensure consistent mock response."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    conversation_input: ResponseInputParam = [
        {
            "type": "message",
            "role": "system",
            "content": [
                {"type": "input_text", "text": "You are a helpful assistant."}
            ],
        },
        {
            "type": "message",
            "role": "user",
            "content": [
                {"type": "input_text", "text": "Hello"}
            ],
        },
        {
            "type": "message",
            "role": "user",
            "content": [
                {"type": "input_text", "text": "How are you?"}
            ],
        },
    ]

    response = client.responses.create(model=MODEL_NAME, input=conversation_input)

    assert response.output_text == EXPECTED_OUTPUT_TEXT
    assert response.usage is not None
    assert response.usage.input_tokens > 3


def test_openai_responses_response_serialization(base_url: str):
    """Test that the response can be properly serialized and deserialized."""
    client = OpenAI(base_url=base_url, api_key=API_KEY)

    response = client.responses.create(model=MODEL_NAME, input="Hello")

    response_dict = response.model_dump()
    assert isinstance(response_dict, dict)
    assert response_dict["model"] == MODEL_NAME
    assert response_dict["output"][0]["content"][0]["text"] == EXPECTED_OUTPUT_TEXT

    response_json = response.model_dump_json()
    parsed = json.loads(response_json)
    assert parsed["model"] == MODEL_NAME
    assert parsed["output"][0]["content"][0]["text"] == EXPECTED_OUTPUT_TEXT
