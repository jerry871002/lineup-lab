import logging
from uuid import UUID

from fastapi import Cookie, Depends, FastAPI, Header, HTTPException, Request, Response, status
from fastapi.responses import JSONResponse
from sqlalchemy.orm import Session

from app.auth_service import login_user, register_user, require_current_user, revoke_session
from app.config import get_settings
from app.db import get_db, is_database_ready
from app.schemas import APIMessage, LoginRequest, RegistrationRequest, UserResponse
from app.session import clear_auth_cookies, is_valid_csrf_token, new_csrf_token, set_auth_cookies


settings = get_settings()
app = FastAPI(title=settings.app_name)
logger = logging.getLogger(__name__)


def require_session_id(session_cookie: str | None) -> UUID:
    if session_cookie is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="authentication required")

    try:
        return UUID(session_cookie)
    except ValueError as err:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="authentication required") from err


@app.get("/healthz", status_code=status.HTTP_200_OK)
def healthz() -> Response:
    return Response(status_code=status.HTTP_200_OK)


@app.get("/readyz", status_code=status.HTTP_200_OK, response_model=None, responses={503: {"model": APIMessage}})
def readyz() -> Response:
    if is_database_ready():
        return Response(status_code=status.HTTP_200_OK)

    return JSONResponse(status_code=status.HTTP_503_SERVICE_UNAVAILABLE, content={"detail": "database is not ready"})


@app.post("/auth/register", status_code=status.HTTP_201_CREATED, response_model=UserResponse)
def register(payload: RegistrationRequest, db: Session = Depends(get_db)) -> UserResponse:
    user = register_user(db, payload)
    return UserResponse(id=user.id, username=user.username, email=user.email)


@app.post("/auth/login", status_code=status.HTTP_200_OK, response_model=UserResponse)
def login(payload: LoginRequest, request: Request, response: Response, db: Session = Depends(get_db)) -> UserResponse:
    client_ip = request.client.host if request.client is not None else None
    user, session = login_user(db, payload, request.headers.get("user-agent"), client_ip)
    csrf_token = new_csrf_token(session.id)
    set_auth_cookies(response, session.id, csrf_token)
    return UserResponse(id=user.id, username=user.username, email=user.email)


@app.post("/auth/logout", status_code=status.HTTP_200_OK, response_model=APIMessage)
def logout(
    response: Response,
    db: Session = Depends(get_db),
    session_cookie: str | None = Cookie(default=None, alias=settings.session_cookie_name),
    csrf_cookie: str | None = Cookie(default=None, alias=settings.csrf_cookie_name),
    csrf_header: str | None = Header(default=None, alias="X-CSRF-Token"),
) -> APIMessage:
    if csrf_cookie is None or csrf_header is None or csrf_cookie != csrf_header:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="csrf token is invalid")

    session_id = require_session_id(session_cookie)
    if not is_valid_csrf_token(session_id, csrf_cookie):
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="csrf token is invalid")

    revoke_session(db, session_id)
    clear_auth_cookies(response)
    return APIMessage(detail="logout succeeded")


@app.get("/users/me", status_code=status.HTTP_200_OK, response_model=UserResponse)
def me(
    db: Session = Depends(get_db),
    session_cookie: str | None = Cookie(default=None, alias=settings.session_cookie_name),
) -> UserResponse:
    session_id = require_session_id(session_cookie)
    user = require_current_user(db, session_id)
    return UserResponse(id=user.id, username=user.username, email=user.email)
