from app.security import hash_password, verify_password


def test_hash_password_uses_argon2_hash() -> None:
    password_hash = hash_password("Supersecret1")

    assert password_hash.startswith("$argon2id$")
    assert password_hash != "Supersecret1"


def test_verify_password_accepts_matching_password() -> None:
    password_hash = hash_password("Supersecret1")

    assert verify_password("Supersecret1", password_hash) is True


def test_hash_password_allows_whitespace_characters() -> None:
    password_hash = hash_password(" Super secret 1A ")

    assert verify_password(" Super secret 1A ", password_hash) is True


def test_verify_password_rejects_wrong_password_or_invalid_hash() -> None:
    password_hash = hash_password("Supersecret1")

    assert verify_password("WrongPassword1", password_hash) is False
    assert verify_password("Supersecret1", "not-a-real-hash") is False
