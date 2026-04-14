from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(extra="ignore", populate_by_name=True)

    app_name: str = Field(default="lineup-lab-auth", alias="APP_NAME")
    database_url: str | None = Field(default=None, alias="DATABASE_URL")
    password_hash_time_cost: int = Field(default=3, alias="PASSWORD_HASH_TIME_COST", validate_default=True)
    password_hash_memory_cost: int = Field(default=65536, alias="PASSWORD_HASH_MEMORY_COST", validate_default=True)
    password_hash_parallelism: int = Field(default=4, alias="PASSWORD_HASH_PARALLELISM", validate_default=True)
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


def get_settings() -> Settings:
    return Settings()
