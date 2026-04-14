from base64 import urlsafe_b64encode
from datetime import UTC, datetime, timedelta
import hashlib
import hmac
import secrets
import uuid

from fastapi import Response

from app.config import get_settings


def utc_now() -> datetime:
    return datetime.now(UTC)


def new_session_expiry(now: datetime | None = None) -> datetime:
    reference = now or utc_now()
    return reference + timedelta(seconds=get_settings().session_ttl_seconds)


def new_session_id() -> uuid.UUID:
    return uuid.uuid4()


def new_csrf_token(session_id: uuid.UUID) -> str:
    nonce = secrets.token_urlsafe(24)
    return new_signed_csrf_token(str(session_id), nonce)


def is_valid_csrf_token(session_id: uuid.UUID, csrf_token: str) -> bool:
    try:
        payload_session_id, nonce, encoded_signature = csrf_token.split(":", 2)
    except ValueError:
        return False

    if payload_session_id != str(session_id) or not nonce or not encoded_signature:
        return False

    expected_token = new_signed_csrf_token(payload_session_id, nonce)
    return hmac.compare_digest(expected_token, csrf_token)


def new_signed_csrf_token(session_id: str, nonce: str) -> str:
    settings = get_settings()
    payload = f"{session_id}:{nonce}"
    signature = hmac.new(
        settings.session_hmac_secret.encode("utf-8"),
        payload.encode("utf-8"),
        hashlib.sha256,
    ).digest()
    encoded_signature = urlsafe_b64encode(signature).decode("ascii")
    return f"{payload}:{encoded_signature}"


def set_auth_cookies(response: Response, session_id: uuid.UUID, csrf_token: str) -> None:
    settings = get_settings()

    response.set_cookie(
        key=settings.session_cookie_name,
        value=str(session_id),
        max_age=settings.session_ttl_seconds,
        httponly=True,
        secure=settings.session_cookie_secure,
        samesite="lax",
        path="/",
    )
    response.set_cookie(
        key=settings.csrf_cookie_name,
        value=csrf_token,
        max_age=settings.session_ttl_seconds,
        httponly=False,
        secure=settings.session_cookie_secure,
        samesite="lax",
        path="/",
    )


def clear_auth_cookies(response: Response) -> None:
    settings = get_settings()
    response.delete_cookie(key=settings.session_cookie_name, path="/")
    response.delete_cookie(key=settings.csrf_cookie_name, path="/")
