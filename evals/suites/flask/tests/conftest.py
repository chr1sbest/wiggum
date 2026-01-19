"""
Pytest fixtures for Flask Day Server testing.

These fixtures provide common setup for testing the Flask web server
including base URL configuration and HTTP client setup.
"""

import os
import pytest
import requests


@pytest.fixture(scope="session")
def base_url() -> str:
    """
    Base URL for the Flask Day Server.

    Can be overridden with FLASK_BASE_URL environment variable.
    Default: http://localhost:8000
    """
    return os.environ.get("FLASK_BASE_URL", "http://localhost:8000")


@pytest.fixture
def client(base_url: str) -> requests.Session:
    """
    Configured requests session for HTTP calls.

    Pre-configured with:
    - Base URL
    - Connection pooling

    Usage:
        response = client.get("/")
    """
    session = requests.Session()
    session.base_url = base_url

    # Override request method to prepend base_url
    original_request = session.request

    def request_with_base_url(method, url, *args, **kwargs):
        if not url.startswith("http"):
            url = f"{base_url}{url}"
        return original_request(method, url, *args, **kwargs)

    session.request = request_with_base_url

    return session
