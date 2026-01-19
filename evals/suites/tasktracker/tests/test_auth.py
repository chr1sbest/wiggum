"""
Authentication tests for Task Tracker API.

These tests validate the authentication endpoints and JWT token handling.
Tests are implementation-agnostic and focus on the API contract.
"""

import pytest
import requests
import uuid


def test_user_registration(base_url):
    """Test that new users can register successfully."""
    test_id = str(uuid.uuid4())[:8]
    response = requests.post(
        f"{base_url}/api/auth/register",
        json={
            "username": f"newuser_{test_id}",
            "email": f"newuser_{test_id}@example.com",
            "password": "ValidPassword123!"
        }
    )

    assert response.status_code in [200, 201], \
        f"Registration failed: {response.status_code} {response.text}"

    # Response should contain user info (implementation may vary)
    data = response.json()
    assert data is not None


def test_login_returns_jwt(base_url):
    """Test that login with valid credentials returns a JWT token."""
    # Register a user first
    test_id = str(uuid.uuid4())[:8]
    username = f"loginuser_{test_id}"
    password = "TestPassword123!"

    requests.post(
        f"{base_url}/api/auth/register",
        json={
            "username": username,
            "email": f"{username}@example.com",
            "password": password
        }
    )

    # Now login
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

    # Check for token in common response formats
    token = data.get("token") or data.get("access_token") or data.get("jwt")
    assert token, f"No token in response: {data}"
    assert isinstance(token, str)
    assert len(token) > 0


def test_invalid_credentials_rejected(base_url):
    """Test that login with invalid credentials is rejected."""
    response = requests.post(
        f"{base_url}/api/auth/login",
        json={
            "username": "nonexistent_user",
            "password": "WrongPassword123!"
        }
    )

    # Should return 401 Unauthorized or 403 Forbidden
    assert response.status_code in [401, 403], \
        f"Expected 401/403 for invalid credentials, got {response.status_code}"


def test_protected_route_requires_auth(base_url):
    """Test that protected routes require authentication."""
    # Try to access a protected endpoint without auth
    response = requests.get(f"{base_url}/api/tasks")

    # Should return 401 Unauthorized (not authenticated)
    assert response.status_code == 401, \
        f"Expected 401 for unauthenticated request, got {response.status_code}"


def test_protected_route_with_valid_token(api_client):
    """Test that protected routes accept valid JWT tokens."""
    # api_client fixture has auth headers pre-configured
    response = api_client.get("/api/tasks")

    # Should succeed (200) or return empty list, but not auth error
    assert response.status_code != 401, \
        "Valid token was rejected"
    assert response.status_code in [200, 404], \
        f"Unexpected status code: {response.status_code}"


def test_invalid_token_rejected(base_url):
    """Test that requests with invalid tokens are rejected."""
    response = requests.get(
        f"{base_url}/api/tasks",
        headers={"Authorization": "Bearer invalid_token_12345"}
    )

    # Should return 401 Unauthorized
    assert response.status_code == 401, \
        f"Expected 401 for invalid token, got {response.status_code}"


def test_duplicate_username_rejected(base_url):
    """Test that registering with duplicate username is rejected."""
    test_id = str(uuid.uuid4())[:8]
    username = f"duplicate_{test_id}"
    user_data = {
        "username": username,
        "email": f"{username}@example.com",
        "password": "Password123!"
    }

    # Register once
    response1 = requests.post(
        f"{base_url}/api/auth/register",
        json=user_data
    )
    assert response1.status_code in [200, 201]

    # Try to register again with same username
    response2 = requests.post(
        f"{base_url}/api/auth/register",
        json=user_data
    )

    # Should return 409 Conflict or 400 Bad Request
    assert response2.status_code in [400, 409], \
        f"Expected 400/409 for duplicate username, got {response2.status_code}"
