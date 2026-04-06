import pytest

from app.db import normalize_database_url


def test_normalize_database_url_supports_repo_postgres_urls() -> None:
    assert normalize_database_url("postgres://user:pass@db/mydatabase?sslmode=disable") == (
        "postgresql+psycopg://user:pass@db/mydatabase?sslmode=disable"
    )
    assert normalize_database_url("postgresql+psycopg://user:pass@db/mydatabase") == (
        "postgresql+psycopg://user:pass@db/mydatabase"
    )


@pytest.mark.parametrize("database_url", ["mysql://user:pass@db/example", "sqlite:///tmp/test.db"])
def test_normalize_database_url_keeps_non_repo_urls(database_url: str) -> None:
    assert normalize_database_url(database_url) == database_url
