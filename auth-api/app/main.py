from __future__ import annotations

import logging

from fastapi import FastAPI, Response, status
from fastapi.responses import JSONResponse

from app.config import get_settings
from app.db import is_database_ready
from app.schemas import APIMessage, LoginRequest, RegistrationRequest


settings = get_settings()
app = FastAPI(title=settings.app_name)
logger = logging.getLogger(__name__)


@app.get("/healthz", status_code=status.HTTP_200_OK)
def healthz() -> Response:
    return Response(status_code=status.HTTP_200_OK)


@app.get("/readyz", status_code=status.HTTP_200_OK, response_model=None, responses={503: {"model": APIMessage}})
def readyz() -> Response:
    if is_database_ready():
        return Response(status_code=status.HTTP_200_OK)

    return JSONResponse(status_code=status.HTTP_503_SERVICE_UNAVAILABLE, content={"detail": "database is not ready"})


@app.post("/auth/register", status_code=status.HTTP_501_NOT_IMPLEMENTED, response_model=APIMessage)
def register(_: RegistrationRequest) -> APIMessage:
    return APIMessage(detail="registration is not implemented yet")


@app.post("/auth/login", status_code=status.HTTP_501_NOT_IMPLEMENTED, response_model=APIMessage)
def login(_: LoginRequest) -> APIMessage:
    return APIMessage(detail="login is not implemented yet")


@app.post("/auth/logout", status_code=status.HTTP_501_NOT_IMPLEMENTED, response_model=APIMessage)
def logout() -> APIMessage:
    return APIMessage(detail="logout is not implemented yet")


@app.get("/users/me", status_code=status.HTTP_501_NOT_IMPLEMENTED, response_model=APIMessage)
def me() -> APIMessage:
    return APIMessage(detail="current user lookup is not implemented yet")
