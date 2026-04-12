from __future__ import annotations

from dataclasses import dataclass
import os


@dataclass(frozen=True)
class Settings:
    app_name: str
    database_url: str | None
    port: int


def parse_port(value: str) -> int:
    try:
        port = int(value)
    except ValueError as err:
        raise ValueError("PORT must be a valid integer") from err

    if port <= 0:
        raise ValueError("PORT must be greater than 0")

    return port


def get_settings() -> Settings:
    port = parse_port(os.getenv("PORT", "8000"))
    database_url = os.getenv("DATABASE_URL")
    app_name = os.getenv("APP_NAME", "lineup-lab-auth")
    return Settings(app_name=app_name, database_url=database_url, port=port)
