from fastapi.testclient import TestClient

from app import main


client = TestClient(main.app)


def test_healthz() -> None:
    response = client.get("/healthz")

    assert response.status_code == 200
    assert response.text == ""


def test_readyz_returns_503_without_database() -> None:
    response = client.get("/readyz")

    assert response.status_code == 503
    assert response.json() == {"detail": "database is not ready"}


def test_auth_routes_are_reserved() -> None:
    response = client.post(
        "/auth/register",
        json={
            "username": "testuser",
            "email": "test@example.com",
            "password": "correct horse battery",
        },
    )

    assert response.status_code == 501
    assert response.json() == {"detail": "registration is not implemented yet"}


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


def test_login_route_is_reserved() -> None:
    response = client.post(
        "/auth/login",
        json={
            "username_or_email": "testuser",
            "password": "correct horse battery",
        },
    )

    assert response.status_code == 501
    assert response.json() == {"detail": "login is not implemented yet"}


def test_logout_and_me_routes_are_reserved() -> None:
    logout_response = client.post("/auth/logout")
    me_response = client.get("/users/me")

    assert logout_response.status_code == 501
    assert logout_response.json() == {"detail": "logout is not implemented yet"}
    assert me_response.status_code == 501
    assert me_response.json() == {"detail": "current user lookup is not implemented yet"}
