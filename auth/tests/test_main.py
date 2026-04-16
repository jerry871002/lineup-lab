from types import SimpleNamespace
from uuid import uuid4

import pytest
from fastapi import HTTPException, status
from fastapi.testclient import TestClient

from app import main


client = TestClient(main.app)


@pytest.fixture(autouse=True)
def override_db_dependency():
    def fake_get_db():
        yield object()

    client.cookies.clear()
    main.app.dependency_overrides[main.get_db] = fake_get_db
    try:
        yield
    finally:
        client.cookies.clear()
        main.app.dependency_overrides.clear()


def test_healthz() -> None:
    response = client.get("/healthz")

    assert response.status_code == 200
    assert response.text == ""


def test_readyz_returns_503_without_database() -> None:
    response = client.get("/readyz")

    assert response.status_code == 503
    assert response.json() == {"detail": "database is not ready"}


def test_register_route_returns_created_user(monkeypatch) -> None:
    def fake_register_user(_db, _payload):
        return SimpleNamespace(id=1, username="testuser", email="test@example.com")

    monkeypatch.setattr(main, "register_user", fake_register_user)

    response = client.post(
        "/auth/register",
        json={
            "username": "testuser",
            "email": "test@example.com",
            "password": "correct horse battery",
        },
    )

    assert response.status_code == 201
    assert response.json() == {"id": 1, "username": "testuser", "email": "test@example.com"}


def test_register_route_normalizes_email_before_service_call(monkeypatch) -> None:
    def fake_register_user(_db, payload):
        assert payload.email == "test@example.com"
        return SimpleNamespace(id=1, username="testuser", email=payload.email)

    monkeypatch.setattr(main, "register_user", fake_register_user)

    response = client.post(
        "/auth/register",
        json={
            "username": "testuser",
            "email": "Test@Example.com",
            "password": "correct horse battery",
        },
    )

    assert response.status_code == 201


def test_register_route_rejects_short_passwords_before_handler() -> None:
    response = client.post(
        "/auth/register",
        json={
            "username": "testuser",
            "email": "test@example.com",
            "password": "short pass",
        },
    )

    assert response.status_code == 422


def test_login_route_sets_auth_and_csrf_cookies(monkeypatch) -> None:
    session_id = uuid4()

    def fake_login_user(_db, _payload, _user_agent, _client_ip):
        user = SimpleNamespace(id=1, username="testuser", email="test@example.com")
        session = SimpleNamespace(id=session_id)
        return user, session

    monkeypatch.setattr(main, "login_user", fake_login_user)

    response = client.post(
        "/auth/login",
        json={
            "username_or_email": "testuser",
            "password": "correct horse battery",
        },
    )

    assert response.status_code == 200
    assert response.json() == {"id": 1, "username": "testuser", "email": "test@example.com"}
    assert main.settings.session_cookie_name in response.cookies
    assert main.settings.csrf_cookie_name in response.cookies


def test_login_route_rejects_invalid_credentials(monkeypatch) -> None:
    def fake_login_user(_db, _payload, _user_agent, _client_ip):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="invalid username/email or password")

    monkeypatch.setattr(main, "login_user", fake_login_user)

    response = client.post(
        "/auth/login",
        json={
            "username_or_email": "testuser",
            "password": "correct horse battery",
        },
    )

    assert response.status_code == 401
    assert response.json() == {"detail": "invalid username/email or password"}


def test_logout_route_rejects_invalid_session_cookie() -> None:
    local_client = TestClient(main.app)
    local_client.cookies.set(main.settings.session_cookie_name, "not-a-uuid")
    local_client.cookies.set(main.settings.csrf_cookie_name, "cookie-token")
    response = local_client.post("/auth/logout", headers={"X-CSRF-Token": "cookie-token"})

    assert response.status_code == 401
    assert response.json() == {"detail": "authentication required"}


def test_logout_route_requires_matching_csrf(monkeypatch) -> None:
    session_id = str(uuid4())
    local_client = TestClient(main.app)
    local_client.cookies.set(main.settings.session_cookie_name, session_id)
    local_client.cookies.set(main.settings.csrf_cookie_name, "wrong-cookie-token")
    response = local_client.post("/auth/logout", headers={"X-CSRF-Token": "wrong-header-token"})

    assert response.status_code == 403
    assert response.json() == {"detail": "csrf token is invalid"}


def test_logout_route_revokes_session_and_clears_cookies(monkeypatch) -> None:
    session_id = uuid4()
    csrf_token = main.new_csrf_token(session_id)

    def fake_revoke_session(_db, actual_session_id):
        assert actual_session_id == session_id
        return SimpleNamespace(id=actual_session_id)

    monkeypatch.setattr(main, "revoke_session", fake_revoke_session)

    local_client = TestClient(main.app)
    local_client.cookies.set(main.settings.session_cookie_name, str(session_id))
    local_client.cookies.set(main.settings.csrf_cookie_name, csrf_token)
    response = local_client.post("/auth/logout", headers={"X-CSRF-Token": csrf_token})

    assert response.status_code == 200
    assert response.json() == {"detail": "logout succeeded"}
    set_cookie_headers = response.headers.get_list("set-cookie")
    assert any(f"{main.settings.session_cookie_name}=" in header for header in set_cookie_headers)
    assert any(f"{main.settings.csrf_cookie_name}=" in header for header in set_cookie_headers)


def test_me_route_returns_current_user(monkeypatch) -> None:
    session_id = uuid4()

    def fake_require_current_user(_db, actual_session_id):
        assert actual_session_id == session_id
        return SimpleNamespace(id=1, username="testuser", email="test@example.com")

    monkeypatch.setattr(main, "require_current_user", fake_require_current_user)

    local_client = TestClient(main.app)
    local_client.cookies.set(main.settings.session_cookie_name, str(session_id))
    response = local_client.get("/users/me")

    assert response.status_code == 200
    assert response.json() == {"id": 1, "username": "testuser", "email": "test@example.com"}


def test_me_route_requires_authentication() -> None:
    response = client.get("/users/me")

    assert response.status_code == 401
    assert response.json() == {"detail": "authentication required"}
