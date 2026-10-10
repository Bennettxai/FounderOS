"""Exception hierarchy for the Optimal Engine SDK.

All errors raised by the client inherit from :class:`OptimalEngineError`,
making it safe to wrap any SDK call in a single ``except`` clause.
"""

from __future__ import annotations

from typing import Any


class OptimalEngineError(Exception):
    """Base exception for every error surfaced by the SDK."""

    def __init__(
        self,
        message: str,
        *,
        status_code: int | None = None,
        body: Any | None = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.body = body

    def __str__(self) -> str:
        if self.status_code is not None:
            return f"[{self.status_code}] {self.message}"
        return self.message


class APIConnectionError(OptimalEngineError):
    """Raised when the SDK cannot reach the engine (network / DNS / TLS)."""


class APITimeoutError(APIConnectionError):
    """Raised when a request exceeds the configured timeout."""


class APIStatusError(OptimalEngineError):
    """Raised for any non-2xx HTTP response."""


class BadRequestError(APIStatusError):
    """HTTP 400 — request validation failed engine-side."""


class AuthenticationError(APIStatusError):
    """HTTP 401 — credentials missing or invalid."""


class PermissionDeniedError(APIStatusError):
    """HTTP 403 — caller is authenticated but not authorized."""


class NotFoundError(APIStatusError):
    """HTTP 404 — the requested resource does not exist."""


class ConflictError(APIStatusError):
    """HTTP 409 — request conflicts with current resource state."""


class ValidationError(APIStatusError):
    """HTTP 422 — semantic validation failed on the engine."""


class RateLimitError(APIStatusError):
    """HTTP 429 — caller is being rate-limited."""


class InternalServerError(APIStatusError):
    """HTTP 5xx — engine-side failure."""


_STATUS_TO_EXC: dict[int, type[APIStatusError]] = {
    400: BadRequestError,
    401: AuthenticationError,
    403: PermissionDeniedError,
    404: NotFoundError,
    409: ConflictError,
    422: ValidationError,
    429: RateLimitError,
}


def status_to_exception(status_code: int) -> type[APIStatusError]:
    """Map an HTTP status code to the appropriate exception class."""
    if status_code in _STATUS_TO_EXC:
        return _STATUS_TO_EXC[status_code]
    if status_code >= 500:
        return InternalServerError
    return APIStatusError


__all__ = [
    "OptimalEngineError",
    "APIConnectionError",
    "APITimeoutError",
    "APIStatusError",
    "BadRequestError",
    "AuthenticationError",
    "PermissionDeniedError",
    "NotFoundError",
    "ConflictError",
    "ValidationError",
    "RateLimitError",
    "InternalServerError",
    "status_to_exception",
]
