"""
User management Pydantic schemas
"""

from typing import Optional
from datetime import datetime
from pydantic import BaseModel, EmailStr, Field


class UserBase(BaseModel):
    """Base user schema"""
    email: EmailStr = Field(..., description="User email address")
    username: str = Field(..., min_length=3, max_length=128, description="Username")
    first_name: Optional[str] = Field(None, max_length=128, description="First name")
    last_name: Optional[str] = Field(None, max_length=128, description="Last name")


class UserCreate(UserBase):
    """Schema for creating a new user"""
    password: str = Field(..., min_length=8, description="User password (min 8 characters)")
    is_admin: bool = Field(default=False, description="Grant admin privileges")
    is_active: bool = Field(default=True, description="Account active status")


class UserUpdate(BaseModel):
    """Schema for updating a user"""
    email: Optional[EmailStr] = None
    username: Optional[str] = Field(None, min_length=3, max_length=128)
    first_name: Optional[str] = Field(None, max_length=128)
    last_name: Optional[str] = Field(None, max_length=128)
    is_admin: Optional[bool] = None
    is_active: Optional[bool] = None
    totp_enabled: Optional[bool] = None


class UserResponse(UserBase):
    """Schema for user response"""
    id: int
    is_active: bool
    is_admin: bool
    is_verified: bool
    totp_enabled: bool
    created_at: datetime
    updated_at: datetime
    last_login: Optional[datetime]

    class Config:
        from_attributes = True


class UserListResponse(BaseModel):
    """Schema for list of users"""
    total: int
    users: list[UserResponse]


class UserClusterAssignmentCreate(BaseModel):
    """Schema for assigning user to cluster"""
    user_id: int = Field(..., description="User ID")
    cluster_id: int = Field(..., description="Cluster ID")
    role: str = Field(default="service_owner", description="Role in cluster")


class UserServiceAssignmentCreate(BaseModel):
    """Schema for assigning user to service"""
    user_id: int = Field(..., description="User ID")
    service_id: int = Field(..., description="Service ID")
