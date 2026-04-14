import pytest
from pydantic import ValidationError

from app.config import get_settings


def test_get_settings_uses_defaults(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("APP_NAME", raising=False)
    monkeypatch.delenv("DATABASE_URL", raising=False)
    monkeypatch.delenv("PASSWORD_HASH_TIME_COST", raising=False)
    monkeypatch.delenv("PASSWORD_HASH_MEMORY_COST", raising=False)
    monkeypatch.delenv("PASSWORD_HASH_PARALLELISM", raising=False)
    monkeypatch.delenv("SESSION_COOKIE_NAME", raising=False)
    monkeypatch.delenv("CSRF_COOKIE_NAME", raising=False)
    monkeypatch.delenv("SESSION_TTL_SECONDS", raising=False)
    monkeypatch.delenv("SESSION_COOKIE_SECURE", raising=False)
    monkeypatch.setenv("SESSION_HMAC_SECRET", "test-session-secret")
    monkeypatch.delenv("PORT", raising=False)

    settings = get_settings()

    assert settings.app_name == "lineup-lab-auth"
    assert settings.database_url is None
    assert settings.password_hash_time_cost == 3
    assert settings.password_hash_memory_cost == 65536
    assert settings.password_hash_parallelism == 4
    assert settings.session_cookie_name == "lineup_lab_session"
    assert settings.csrf_cookie_name == "lineup_lab_csrf"
    assert settings.session_ttl_seconds == 1_209_600
    assert settings.session_cookie_secure is False
    assert settings.session_hmac_secret == "test-session-secret"
    assert settings.port == 8000


def test_get_settings_reads_environment(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("APP_NAME", "test-auth")
    monkeypatch.setenv("DATABASE_URL", "postgres://user:pass@postgres/example")
    monkeypatch.setenv("PASSWORD_HASH_TIME_COST", "2")
    monkeypatch.setenv("PASSWORD_HASH_MEMORY_COST", "32768")
    monkeypatch.setenv("PASSWORD_HASH_PARALLELISM", "2")
    monkeypatch.setenv("SESSION_COOKIE_NAME", "session_cookie")
    monkeypatch.setenv("CSRF_COOKIE_NAME", "csrf_cookie")
    monkeypatch.setenv("SESSION_TTL_SECONDS", "3600")
    monkeypatch.setenv("SESSION_COOKIE_SECURE", "true")
    monkeypatch.setenv("SESSION_HMAC_SECRET", "super-secret")
    monkeypatch.setenv("PORT", "9000")

    settings = get_settings()

    assert settings.app_name == "test-auth"
    assert settings.database_url == "postgres://user:pass@postgres/example"
    assert settings.password_hash_time_cost == 2
    assert settings.password_hash_memory_cost == 32768
    assert settings.password_hash_parallelism == 2
    assert settings.session_cookie_name == "session_cookie"
    assert settings.csrf_cookie_name == "csrf_cookie"
    assert settings.session_ttl_seconds == 3600
    assert settings.session_cookie_secure is True
    assert settings.session_hmac_secret == "super-secret"
    assert settings.port == 9000


def test_get_settings_rejects_invalid_port(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PORT", "not-a-number")

    with pytest.raises(ValidationError, match="PORT must be a valid integer"):
        get_settings()


def test_get_settings_rejects_non_positive_port(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PORT", "0")

    with pytest.raises(ValidationError, match="PORT must be greater than 0"):
        get_settings()


def test_get_settings_rejects_invalid_password_hash_setting(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PASSWORD_HASH_MEMORY_COST", "invalid")

    with pytest.raises(ValidationError, match="password hash settings must be valid integers"):
        get_settings()


def test_get_settings_rejects_invalid_session_ttl(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("SESSION_TTL_SECONDS", "0")

    with pytest.raises(ValidationError, match="SESSION_TTL_SECONDS must be greater than 0"):
        get_settings()


def test_get_settings_rejects_invalid_session_cookie_secure(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("SESSION_COOKIE_SECURE", "sometimes")

    with pytest.raises(ValidationError, match="SESSION_COOKIE_SECURE must be a valid boolean"):
        get_settings()


def test_get_settings_requires_session_hmac_secret(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("SESSION_HMAC_SECRET", raising=False)

    with pytest.raises(ValidationError):
        get_settings()
