import re
from functools import lru_cache

from argon2 import PasswordHasher
from argon2.exceptions import InvalidHashError, VerificationError, VerifyMismatchError

from app.config import get_settings


_USERNAME_PATTERN = re.compile(r"^[A-Za-z0-9_]+$")
_MIN_LOGIN_IDENTIFIER_LENGTH = 3
MIN_PASSWORD_LENGTH = 15


@lru_cache
def get_password_hasher() -> PasswordHasher:
    # Build the configured Argon2 hasher once per process so password checks
    # reuse the same validated settings instead of recreating the object.
    settings = get_settings()
    return PasswordHasher(
        time_cost=settings.password_hash_time_cost,
        memory_cost=settings.password_hash_memory_cost,
        parallelism=settings.password_hash_parallelism,
    )


def validate_username(username: str) -> str:
    normalized = username.strip().lower()

    if len(normalized) < _MIN_LOGIN_IDENTIFIER_LENGTH:
        raise ValueError(f"username must be at least {_MIN_LOGIN_IDENTIFIER_LENGTH} characters long")

    if not _USERNAME_PATTERN.fullmatch(normalized):
        raise ValueError("username may only contain letters, numbers, and underscores")

    return normalized


def validate_password_strength(password: str) -> str:
    # Intentionally avoid composition rules here. NIST SP 800-63B and OWASP
    # recommend length-based password validation plus blocklist checks instead
    # of requirements like "must include uppercase/lowercase/number". The
    # minimum length is enforced at the schema layer; this hook remains as the
    # place for future password checks such as compromised-password blocklists.
    return password


def validate_login_identifier(identifier: str) -> str:
    normalized = identifier.strip().lower()
    if len(normalized) < _MIN_LOGIN_IDENTIFIER_LENGTH:
        raise ValueError(f"username_or_email must be at least {_MIN_LOGIN_IDENTIFIER_LENGTH} characters long")

    return normalized


def hash_password(password: str) -> str:
    return get_password_hasher().hash(password)


def verify_password(password: str, password_hash: str) -> bool:
    try:
        return get_password_hasher().verify(password_hash, password)
    except (InvalidHashError, VerificationError, VerifyMismatchError):
        return False
