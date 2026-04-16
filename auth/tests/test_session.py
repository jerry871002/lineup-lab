from uuid import uuid4

from app.session import is_valid_csrf_token, new_csrf_token


def test_new_csrf_token_is_valid_for_matching_session() -> None:
    session_id = uuid4()
    csrf_token = new_csrf_token(session_id)

    assert is_valid_csrf_token(session_id, csrf_token) is True


def test_new_csrf_token_is_invalid_for_different_session() -> None:
    csrf_token = new_csrf_token(uuid4())

    assert is_valid_csrf_token(uuid4(), csrf_token) is False
