"""
Tests for Flask Day Server basic functionality.

These tests verify the Flask server responds correctly and displays
the current day information.
"""

import pytest
import requests
from datetime import datetime


def test_server_is_running(client: requests.Session):
    """Test that the Flask server is accessible and responding."""
    response = client.get("/")
    assert response.status_code == 200, "Server should respond with 200 OK"


def test_index_page_content_type(client: requests.Session):
    """Test that the index page returns HTML content."""
    response = client.get("/")
    assert response.status_code == 200
    content_type = response.headers.get("Content-Type", "")
    assert "text/html" in content_type, f"Expected HTML content type, got {content_type}"


def test_index_page_contains_day(client: requests.Session):
    """Test that the index page displays day of week information."""
    response = client.get("/")
    assert response.status_code == 200

    text = response.text.lower()

    # Check for the word "today"
    assert "today" in text, "Page should mention 'today'"

    # Check for current day of week
    days = ["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"]
    current_day = datetime.now().strftime("%A").lower()

    assert current_day in text, f"Page should display current day '{current_day}'"


def test_index_page_has_html_structure(client: requests.Session):
    """Test that the page has basic HTML structure."""
    response = client.get("/")
    assert response.status_code == 200

    html = response.text.lower()

    # Check for essential HTML tags
    assert "<html" in html or "<!doctype html>" in html, "Should have HTML tag or doctype"
    assert "<body" in html, "Should have body tag"


def test_index_page_has_styling(client: requests.Session):
    """Test that the page includes basic styling (CSS)."""
    response = client.get("/")
    assert response.status_code == 200

    html = response.text.lower()

    # Check for style indicators (inline styles, style tag, or external stylesheet)
    has_style = (
        "<style" in html or
        "style=" in html or
        'rel="stylesheet"' in html or
        "rel='stylesheet'" in html
    )

    assert has_style, "Page should include some form of styling"


def test_404_for_nonexistent_route(client: requests.Session):
    """Test that non-existent routes return 404."""
    response = client.get("/nonexistent-route-12345")
    assert response.status_code == 404, "Should return 404 for non-existent routes"


def test_index_responds_to_head_request(client: requests.Session):
    """Test that the index responds to HEAD requests."""
    response = client.head("/")
    assert response.status_code == 200, "Should respond to HEAD requests"


def test_response_time_reasonable(client: requests.Session):
    """Test that the server responds in reasonable time (< 2 seconds)."""
    response = client.get("/", timeout=2)
    assert response.status_code == 200
    assert response.elapsed.total_seconds() < 2, "Response should be fast"
