"""
Validation module - PyDAL-style input validators.

Provides validators compatible with PyDAL's IS_* pattern:
- String validators: IsNotEmpty, IsLength, IsMatch, IsAlphanumeric, IsSlug, IsIn
- Numeric validators: IsInt, IsFloat, IsIntInRange, IsFloatInRange
- Network validators: IsEmail, IsURL, IsIPAddress
- DateTime validators: IsDate, IsDateTime, IsTime
- Password validators: IsStrongPassword

Usage:
    from py_libs.validation import IsEmail, IsLength, chain # noqa: F401

  : # Single validator
    validator = IsEmail()
    result = validator("user@example.com")

  : # Chained validators
    validators = chain(IsNotEmpty(), IsLength(3, 255), IsEmail())
    result = validators("user@example.com")
"""

from py_libs.validation.base import ValidationError, ValidationResult, Validator, chain # noqa: F401
from py_libs.validation.datetime import IsDate, IsDateInRange, IsDateTime, IsTime # noqa: F401
from py_libs.validation.network import IsEmail, IsHostname, IsIPAddress, IsURL # noqa: F401
from py_libs.validation.numeric import ( # noqa: F401
    IsFloat,
    IsFloatInRange,
    IsInt,
    IsIntInRange,
    IsNegative,
    IsPositive,
)
from py_libs.validation.password import IsStrongPassword, PasswordOptions # noqa: F401
from py_libs.validation.string import ( # noqa: F401
    IsAlphanumeric,
    IsIn,
    IsLength,
    IsMatch,
    IsNotEmpty,
    IsSlug,
    IsTrimmed,
)

__all__ = [
  : # Base
    "ValidationError",
    "ValidationResult",
    "Validator",
    "chain",
  : # String
    "IsNotEmpty",
    "IsLength",
    "IsMatch",
    "IsAlphanumeric",
    "IsSlug",
    "IsIn",
    "IsTrimmed",
  : # Numeric
    "IsInt",
    "IsFloat",
    "IsIntInRange",
    "IsFloatInRange",
    "IsPositive",
    "IsNegative",
  : # Network
    "IsEmail",
    "IsURL",
    "IsIPAddress",
    "IsHostname",
  : # DateTime
    "IsDate",
    "IsDateTime",
    "IsTime",
    "IsDateInRange",
  : # Password
    "IsStrongPassword",
    "PasswordOptions",
]
