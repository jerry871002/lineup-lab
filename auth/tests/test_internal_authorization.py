from datetime import UTC, datetime, timedelta
from types import SimpleNamespace
from uuid import uuid4

import pytest
from fastapi.testclient import TestClient

from app import main
from app.models import SessionRecord, User


@pytest.fixture
def records():
    session = SessionRecord(id=uuid4(), user_id=42, expires_at=datetime.now(UTC) + timedelta(hours=1))
    user = User(id=42, username="owner", email="owner@example.com")
    return {SessionRecord: session, User: user}


@pytest.fixture
def client(records):
    def get_record(model, key):
        record = records.get(model)
        return record if record is not None and record.id == key else None

    db = SimpleNamespace(get=get_record, commit=lambda: None, refresh=lambda _record: None)

    def get_db():
        yield db

    main.app.dependency_overrides[main.get_db] = get_db
    with TestClient(main.app) as test_client:
        test_client.cookies.set(main.settings.session_cookie_name, str(records[SessionRecord].id))
        yield test_client
    main.app.dependency_overrides.clear()


@pytest.mark.parametrize("method", ["GET", "HEAD", "OPTIONS"])
def test_safe_original_methods_return_session_identity_without_csrf(client, method):
    response = client.get("/internal/auth/verify", headers={"X-Original-Method": method})

    assert response.status_code == 204
    assert response.content == b""
    assert response.headers["X-Authenticated-User-ID"] == "42"


@pytest.mark.parametrize("method", ["POST", "PUT", "PATCH", "DELETE"])
def test_mutating_original_methods_accept_matching_signed_csrf(client, records, method):
    token = main.new_csrf_token(records[SessionRecord].id)
    client.cookies.set(main.settings.csrf_cookie_name, token)

    response = client.get("/internal/auth/verify", headers={"X-Original-Method": method, "X-CSRF-Token": token})

    assert response.status_code == 204
    assert response.headers["X-Authenticated-User-ID"] == "42"


@pytest.mark.parametrize("method", [None, "", "get", "TRACE", "CONNECT", "UNKNOWN"])
def test_missing_or_unsupported_original_method_fails_closed(client, method):
    headers = {} if method is None else {"X-Original-Method": method}

    response = client.get("/internal/auth/verify", headers=headers)

    assert response.status_code == 403
    assert "X-Authenticated-User-ID" not in response.headers


@pytest.mark.parametrize("session_cookie", [None, "not-a-uuid", str(uuid4())])
def test_missing_invalid_or_unknown_session_is_unauthorized(client, session_cookie):
    client.cookies.clear()
    if session_cookie is not None:
        client.cookies.set(main.settings.session_cookie_name, session_cookie)

    response = client.get("/internal/auth/verify", headers={"X-Original-Method": "GET"})

    assert response.status_code == 401
    assert "X-Authenticated-User-ID" not in response.headers


@pytest.mark.parametrize("state", ["expired", "revoked", "missing-user"])
@pytest.mark.parametrize("method", ["GET", "POST"])
def test_inactive_sessions_and_missing_users_are_unauthorized(client, records, state, method):
    if state == "expired":
        records[SessionRecord].expires_at = datetime.now(UTC) - timedelta(seconds=1)
    elif state == "revoked":
        records[SessionRecord].revoked_at = datetime.now(UTC)
    else:
        records.pop(User)

    response = client.get("/internal/auth/verify", headers={"X-Original-Method": method})

    assert response.status_code == 401
    assert "X-Authenticated-User-ID" not in response.headers


@pytest.mark.parametrize("failure", ["both-missing", "cookie-missing", "header-missing", "mismatch", "tampered", "other-session"])
def test_mutating_request_rejects_invalid_csrf(client, records, failure):
    token = main.new_csrf_token(records[SessionRecord].id)
    cookie = token
    header = token
    if failure == "both-missing":
        cookie = header = None
    elif failure == "cookie-missing":
        cookie = None
    elif failure == "header-missing":
        header = None
    elif failure == "mismatch":
        header = main.new_csrf_token(records[SessionRecord].id)
    elif failure == "tampered":
        cookie = header = token + "tampered"
    else:
        cookie = header = main.new_csrf_token(uuid4())
    if cookie is not None:
        client.cookies.set(main.settings.csrf_cookie_name, cookie)
    headers = {"X-Original-Method": "POST"}
    if header is not None:
        headers["X-CSRF-Token"] = header

    response = client.get("/internal/auth/verify", headers=headers)

    assert response.status_code == 403
    assert "X-Authenticated-User-ID" not in response.headers


def test_spoofed_identity_is_replaced_by_session_user(client):
    response = client.get("/internal/auth/verify", headers={
        "X-Original-Method": "GET",
        "X-Authenticated-User-ID": "999",
        "X-User-ID": "999",
    })

    assert response.status_code == 204
    assert response.headers["X-Authenticated-User-ID"] == "42"


def test_browser_api_alias_does_not_expose_internal_verifier(client):
    response = client.get("/api/internal/auth/verify", headers={"X-Original-Method": "GET"})

    assert response.status_code == 404
