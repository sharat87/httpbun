import os

import pytest


@pytest.fixture
def base_url() -> str:
    return os.getenv("BASE_URL")
