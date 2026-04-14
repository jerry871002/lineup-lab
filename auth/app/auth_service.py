from datetime import UTC, datetime
from ipaddress import ip_address
from uuid import UUID

from fastapi import HTTPException, status
from sqlalchemy import or_, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import SessionRecord, User
from app.schemas import LoginRequest, RegistrationRequest
from app.security import hash_password, verify_password
from app.session import new_session_expiry, new_session_id


def register_user(db: Session, payload: RegistrationRequest) -> User:
    existing_user = db.scalar(
        select(User).where(or_(User.username == payload.username, User.email == payload.email))
    )
    if existing_user is not None:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="username or email is already registered",
        )

    user = User(
        username=payload.username,
        email=payload.email,
        password_hash=hash_password(payload.password),
    )
    db.add(user)
    try:
        db.commit()
    except IntegrityError as err:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="username or email is already registered",
        ) from err
    db.refresh(user)
    return user


def login_user(db: Session, payload: LoginRequest, user_agent: str | None, client_ip: str | None) -> tuple[User, SessionRecord]:
    user = db.scalar(
        select(User).where(or_(User.username == payload.username_or_email, User.email == payload.username_or_email))
    )
    if user is None or not verify_password(payload.password, user.password_hash):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="invalid username/email or password",
        )

    session = SessionRecord(
        id=new_session_id(),
        user_id=user.id,
        expires_at=new_session_expiry(),
        user_agent=user_agent,
        ip_address=normalize_ip_address(client_ip),
    )
    user.last_login_at = datetime.now(UTC)
    db.add(session)
    db.commit()
    db.refresh(user)
    db.refresh(session)
    return user, session


def revoke_session(db: Session, session_id: UUID) -> SessionRecord:
    now = datetime.now(UTC)
    session = require_active_session(db, session_id, persist_last_seen=False)
    session.last_seen_at = now
    session.revoked_at = now
    db.commit()
    db.refresh(session)
    return session


def require_current_user(db: Session, session_id: UUID) -> User:
    session = require_active_session(db, session_id)
    user = db.get(User, session.user_id)
    if user is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="authentication required")

    return user


def require_active_session(db: Session, session_id: UUID, *, persist_last_seen: bool = True) -> SessionRecord:
    session = db.get(SessionRecord, session_id)
    if session is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="authentication required")

    now = datetime.now(UTC)
    if session.revoked_at is not None or session.expires_at <= now:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="authentication required")

    if persist_last_seen:
        session.last_seen_at = now
        db.commit()
        db.refresh(session)

    return session


def normalize_ip_address(value: str | None) -> str | None:
    if value is None or not value.strip():
        return None

    try:
        return str(ip_address(value.strip()))
    except ValueError:
        return None
