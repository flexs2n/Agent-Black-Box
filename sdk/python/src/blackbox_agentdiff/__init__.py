"""
Blackbox-AgentDiff SDK
===============================

A unified SDK for interacting with the Blackbox-AgentDiff API,
providing clean abstractions over sessions, messages, and contexts.
"""

from .client import BlackboxClient
from .config import Config
from .exceptions import BlackboxError, AuthenticationError, NotFoundError


__all__ = ["BlackboxClient", "Config", "BlackboxError", "AuthenticationError", "NotFoundError"]