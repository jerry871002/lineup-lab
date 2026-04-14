from pydantic import BaseModel, EmailStr, Field, field_validator

from app.security import MIN_PASSWORD_LENGTH, validate_login_identifier, validate_password_strength, validate_username


class RegistrationRequest(BaseModel):
    username: str = Field(min_length=3, max_length=50)
    email: EmailStr
    password: str = Field(min_length=MIN_PASSWORD_LENGTH, max_length=128)

    @field_validator("username")
    @classmethod
    def validate_username_field(cls, value: str) -> str:
        return validate_username(value)

    @field_validator("password")
    @classmethod
    def validate_password_field(cls, value: str) -> str:
        return validate_password_strength(value)


class LoginRequest(BaseModel):
    username_or_email: str = Field(min_length=3, max_length=255)
    password: str = Field(min_length=MIN_PASSWORD_LENGTH, max_length=128)

    @field_validator("username_or_email")
    @classmethod
    def validate_username_or_email_field(cls, value: str) -> str:
        return validate_login_identifier(value)


class APIMessage(BaseModel):
    detail: str
