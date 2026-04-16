import pytest
from pydantic import ValidationError

from app.schemas import LoginRequest, RegistrationRequest


def test_registration_request_accepts_valid_credentials() -> None:
    request = RegistrationRequest(
        username=" Test_User ",
        email="Test@Example.com",
        password="correct horse battery",
    )

    assert request.username == "test_user"
    assert request.email == "test@example.com"
    assert request.password == "correct horse battery"


@pytest.mark.parametrize(
    ("username", "error_message"),
    [
        (" a ", "username must be at least 3 characters long"),
        ("test-user", "username may only contain letters, numbers, and underscores"),
    ],
)
def test_registration_request_rejects_invalid_usernames(username: str, error_message: str) -> None:
    with pytest.raises(ValidationError, match=error_message):
        RegistrationRequest(
            username=username,
            email="test@example.com",
            password="correct horse battery",
        )


def test_registration_request_rejects_passwords_shorter_than_nist_minimum() -> None:
    with pytest.raises(ValidationError, match="at least 15 characters"):
        RegistrationRequest(
            username="test_user",
            email="test@example.com",
            password="short pass",
        )


def test_login_request_trims_identifier() -> None:
    request = LoginRequest(
        username_or_email=" Test@Example.com ",
        password="correct horse battery",
    )

    assert request.username_or_email == "test@example.com"


def test_login_request_rejects_too_short_identifier_after_trimming() -> None:
    with pytest.raises(ValidationError, match="username_or_email must be at least 3 characters long"):
        LoginRequest(
            username_or_email="  a ",
            password="correct horse battery",
        )
