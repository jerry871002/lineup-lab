from __future__ import annotations

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(extra="ignore", populate_by_name=True)

    app_name: str = Field(default="lineup-lab-auth", alias="APP_NAME")
    database_url: str | None = Field(default=None, alias="DATABASE_URL")
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


def get_settings() -> Settings:
    return Settings()
