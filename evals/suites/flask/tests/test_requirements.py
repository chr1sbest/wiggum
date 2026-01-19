"""
Tests verifying Flask Day Server meets specific requirements.

These tests validate the explicit requirements from flask_requirements.md:
- Index at / displays "Today is [day of week]"
- Simple HTML with basic styling
- Runs on port 8000
"""

import pytest
import requests
from datetime import datetime


def test_requirement_index_route(client: requests.Session):
    """Requirement: Index page at / displays today's day of week."""
    response = client.get("/")
    assert response.status_code == 200, "Index route (/) must be accessible"

    # Get current day name
    day_name = datetime.now().strftime("%A")

    # Check for "Today is [day]" format
    text = response.text
    assert "Today" in text or "today" in text, "Page must display 'Today'"
    assert day_name in text, f"Page must display current day name: {day_name}"


def test_requirement_runs_on_port_8000(base_url: str):
    """Requirement: Server runs on port 8000."""
    assert "8000" in base_url, f"Server should run on port 8000 (base_url: {base_url})"

    # Verify the port is actually responding
    response = requests.get(base_url)
    assert response.status_code == 200, "Server on port 8000 should be responding"


def test_requirement_html_response(client: requests.Session):
    """Requirement: Simple HTML page."""
    response = client.get("/")
    assert response.status_code == 200

    # Verify it's HTML
    content_type = response.headers.get("Content-Type", "")
    assert "html" in content_type.lower(), "Response must be HTML"

    # Verify HTML structure
    html = response.text.lower()
    assert "<html" in html or "<!doctype" in html, "Must have HTML structure"


def test_requirement_has_styling(client: requests.Session):
    """Requirement: Basic styling included."""
    response = client.get("/")
    assert response.status_code == 200

    html = response.text.lower()

    # Check for any form of styling
    has_css = (
        "<style" in html or          # Embedded style tag
        "style=" in html or           # Inline styles
        ".css" in html or             # External stylesheet
        'rel="stylesheet"' in html    # Link to stylesheet
    )

    assert has_css, "Page must include basic styling (CSS)"
