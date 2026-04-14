from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(extra="ignore", populate_by_name=True)

    app_name: str = Field(default="lineup-lab-auth", alias="APP_NAME")
    database_url: str | None = Field(default=None, alias="DATABASE_URL")
    password_hash_time_cost: int = Field(default=3, alias="PASSWORD_HASH_TIME_COST", validate_default=True)
    password_hash_memory_cost: int = Field(default=65536, alias="PASSWORD_HASH_MEMORY_COST", validate_default=True)
    password_hash_parallelism: int = Field(default=4, alias="PASSWORD_HASH_PARALLELISM", validate_default=True)
    session_cookie_name: str = Field(default="lineup_lab_session", alias="SESSION_COOKIE_NAME")
    csrf_cookie_name: str = Field(default="lineup_lab_csrf", alias="CSRF_COOKIE_NAME")
    session_ttl_seconds: int = Field(default=1_209_600, alias="SESSION_TTL_SECONDS", validate_default=True)
    session_cookie_secure: bool = Field(default=False, alias="SESSION_COOKIE_SECURE", validate_default=True)
    session_hmac_secret: str = Field(alias="SESSION_HMAC_SECRET")
    port: int = Field(default=8000, alias="PORT", validate_default=True)

    @field_validator("port", mode="before")
    @classmethod
    def validate_port(cls, value: object) -> int:
        try:
            port = int(value)
        except (TypeError, ValueError) as err:
            raise ValueError("PORT must be a valid integer") from err

        if port <= 0:
            raise ValueError("PORT must be greater than 0")

        return port

    @field_validator("password_hash_time_cost", "password_hash_memory_cost", "password_hash_parallelism", mode="before")
    @classmethod
    def validate_password_hash_setting(cls, value: object) -> int:
        try:
            parsed_value = int(value)
        except (TypeError, ValueError) as err:
            raise ValueError("password hash settings must be valid integers") from err

        if parsed_value <= 0:
            raise ValueError("password hash settings must be greater than 0")

        return parsed_value

    @field_validator("session_ttl_seconds", mode="before")
    @classmethod
    def validate_session_ttl_seconds(cls, value: object) -> int:
        try:
            parsed_value = int(value)
        except (TypeError, ValueError) as err:
            raise ValueError("SESSION_TTL_SECONDS must be a valid integer") from err

        if parsed_value <= 0:
            raise ValueError("SESSION_TTL_SECONDS must be greater than 0")

        return parsed_value

    @field_validator("session_cookie_secure", mode="before")
    @classmethod
    def validate_session_cookie_secure(cls, value: object) -> bool:
        if isinstance(value, bool):
            return value

        if isinstance(value, str):
            normalized = value.strip().lower()
            if normalized in {"1", "true", "yes", "on"}:
                return True
            if normalized in {"0", "false", "no", "off"}:
                return False

        raise ValueError("SESSION_COOKIE_SECURE must be a valid boolean")

    @field_validator("session_cookie_name", "csrf_cookie_name")
    @classmethod
    def validate_cookie_name(cls, value: str) -> str:
        normalized = value.strip()
        if not normalized:
            raise ValueError("cookie names must not be empty")

        return normalized

    @field_validator("session_hmac_secret")
    @classmethod
    def validate_session_hmac_secret(cls, value: str) -> str:
        normalized = value.strip()
        if not normalized:
            raise ValueError("SESSION_HMAC_SECRET must not be empty")

        return normalized


def get_settings() -> Settings:
    return Settings()
