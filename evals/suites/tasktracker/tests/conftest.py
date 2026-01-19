"""
Pytest fixtures for Task Tracker API testing.

These fixtures provide common setup for API integration tests including
authentication, HTTP client, and base URL configuration.
"""

import os
import pytest
import requests
from typing import Dict


@pytest.fixture(scope="session")
def base_url() -> str:
    """
    Base URL for the Task Tracker API.

    Can be overridden with TASKTRACKER_BASE_URL environment variable.
    Default: http://localhost:8000
    """
    return os.environ.get("TASKTRACKER_BASE_URL", "http://localhost:8000")


@pytest.fixture(scope="session")
def auth_headers(base_url: str) -> Dict[str, str]:
    """
    Authentication headers with JWT token.

    Registers a test user and obtains a JWT token for authenticated requests.
    The token is obtained once per test session and reused across tests.

    Returns:
        Dict with Authorization header containing the JWT token
    """
    # Generate unique test user credentials
    import uuid
    test_id = str(uuid.uuid4())[:8]
    username = f"testuser_{test_id}"
    email = f"testuser_{test_id}@example.com"
    password = "TestPassword123!"

    # Register user
    response = requests.post(
        f"{base_url}/api/auth/register",
        json={
            "username": username,
            "email": email,
            "password": password
        }
    )

    # If registration fails with 409, user might already exist, try login
    if response.status_code == 409:
        pass  # Continue to login
    else:
        assert response.status_code in [200, 201], \
            f"Registration failed: {response.status_code} {response.text}"

    # Login to get JWT token
    response = requests.post(
        f"{base_url}/api/auth/login",
        json={
            "username": username,
            "password": password
        }
    )

    assert response.status_code == 200, \
        f"Login failed: {response.status_code} {response.text}"

    data = response.json()

    # Try common JWT response formats
    token = data.get("token") or data.get("access_token") or data.get("jwt")

    assert token, f"No token in response: {data}"

    return {"Authorization": f"Bearer {token}"}


@pytest.fixture
def api_client(base_url: str, auth_headers: Dict[str, str]) -> requests.Session:
    """
    Configured requests session for API calls.

    Pre-configured with:
    - Base URL
    - Authentication headers
    - JSON content type

    Usage:
        response = api_client.get("/api/tasks")
        response = api_client.post("/api/tasks", json={"title": "New Task"})
    """
    session = requests.Session()
    session.headers.update(auth_headers)
    session.headers.update({"Content-Type": "application/json"})

    # Store base_url as an attribute for convenience
    session.base_url = base_url

    # Override request method to prepend base_url
    original_request = session.request

    def request_with_base_url(method, url, *args, **kwargs):
        if not url.startswith("http"):
            url = f"{base_url}{url}"
        return original_request(method, url, *args, **kwargs)

    session.request = request_with_base_url

    return session
