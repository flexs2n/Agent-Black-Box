"""
Blackbox-AgentDiff Python SDK

A clean abstraction for tracing agent executions with the Blackbox API.
"""

import json
import os
import time
import uuid
from dataclasses import dataclass, field
from typing import Any, Dict, Optional, Generator
from urllib.request import Request, urlopen
from urllib.error import HTTPError


__version__ = "0.1.0"


class BlackboxError(Exception):
    pass


class AuthenticationError(BlackboxError):
    pass


class NotFoundError(BlackboxError):
    pass


@dataclass
class Config:
    api_key: str
    project_id: str
    base_url: str = "http://localhost:4000"
    endpoint: str = "/otel/v1/traces"


@dataclass
class TraceContext:
    trace_id: str
    project_id: str
    name: str
    start_time: float
    input: Optional[Dict[str, Any]] = None
    output: Optional[Dict[str, Any]] = None
    _spans: list = field(default_factory=list)

    def generation(self, name: str, model: str = "", **kwargs):
        span = SpanContext(
            trace_id=self.trace_id,
            project_id=self.project_id,
            name=name,
            span_kind="generation",
            start_time=time.time(),
            model=model,
            **kwargs,
        )
        self._spans.append(span)
        return span

    def tool(self, name: str, **kwargs):
        span = SpanContext(
            trace_id=self.trace_id,
            project_id=self.project_id,
            name=name,
            span_kind="tool",
            start_time=time.time(),
            **kwargs,
        )
        self._spans.append(span)
        return span

    def retrieval(self, name: str, **kwargs):
        span = SpanContext(
            trace_id=self.trace_id,
            project_id=self.project_id,
            name=name,
            span_kind="retrieval",
            start_time=time.time(),
            **kwargs,
        )
        self._spans.append(span)
        return span

    def set_output(self, output: Dict[str, Any]):
        self.output = output

    def end(self):
        self.end_time = time.time()
        self._export()

    def _export(self):
        spans = []
        for span in self._spans:
            attrs = {
                "blackbox.span_kind": span.span_kind,
                **span.attributes,
            }
            if span.model:
                attrs["gen_ai.request.model"] = span.model
            if span.input is not None:
                attrs["gen_ai.prompt"] = json.dumps(span.input) if not isinstance(span.input, str) else span.input
            if span.output is not None:
                attrs["gen_ai.completion"] = json.dumps(span.output) if not isinstance(span.output, str) else span.output
            if span.input_tokens is not None:
                attrs["gen_ai.usage.input_tokens"] = span.input_tokens
            if span.output_tokens is not None:
                attrs["gen_ai.usage.output_tokens"] = span.output_tokens
            if span.duration_ms is not None:
                attrs["duration_ms"] = span.duration_ms

            event = {
                "name": span.name,
                "time_unix_nano": int(span.start_time * 1e9),
                "attributes": [{"key": k, "value": {"string_value": str(v)}} for k, v in attrs.items()],
            }

            spans.append({
                "trace_id": self.trace_id,
                "span_id": span.span_id,
                "parent_span_id": self.trace_id,
                "name": span.name,
                "kind": 1,
                "start_time_unix_nano": int(span.start_time * 1e9),
                "end_time_unix_nano": int(span.end_time * 1e9),
                "attributes": event["attributes"],
                "status": {"code": 1},
            })

        payload = {
            "resource_spans": [{
                "resource": {"attributes": [{"key": "service.name", "value": {"string_value": self.project_id}}]},
                "scope_spans": [{"spans": spans}],
            }]
        }

        config = _get_config()
        req = Request(
            config.base_url + config.endpoint,
            data=json.dumps(payload).encode(),
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {config.api_key}",
                "X-Blackbox-Project-ID": config.project_id,
            },
            method="POST",
        )
        try:
            with urlopen(req) as resp:
                pass
        except HTTPError as e:
            raise BlackboxError(f"Export failed: {e.code} {e.reason}")


@dataclass
class SpanContext:
    trace_id: str
    project_id: str
    name: str
    span_kind: str
    start_time: float
    end_time: Optional[float] = None
    span_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    model: str = ""
    input: Any = None
    output: Any = None
    input_tokens: Optional[int] = None
    output_tokens: Optional[int] = None
    duration_ms: Optional[int] = None
    attributes: Dict[str, Any] = field(default_factory=dict)
    error: Optional[str] = None

    def record(self, input=None, output=None, input_tokens=None, output_tokens=None, **kwargs):
        if input is not None:
            self.input = input
        if output is not None:
            self.output = output
        if input_tokens is not None:
            self.input_tokens = input_tokens
        if output_tokens is not None:
            self.output_tokens = output_tokens
        self.end_time = time.time()
        self.duration_ms = int((self.end_time - self.start_time) * 1000)

    def set_error(self, message: str):
        self.error = message


_config: Optional[Config] = None


def _get_config() -> Config:
    if _config is None:
        api_key = os.environ.get("BLACKBOX_API_KEY", "")
        project_id = os.environ.get("BLACKBOX_PROJECT_ID", "")
        base_url = os.environ.get("BLACKBOX_BASE_URL", "http://localhost:4000")
        if not api_key or not project_id:
            raise BlackboxError("BLACKBOX_API_KEY and BLACKBOX_PROJECT_ID must be set")
        return Config(api_key=api_key, project_id=project_id, base_url=base_url)
    return _config


def init(
    api_key: Optional[str] = None,
    project_id: Optional[str] = None,
    base_url: Optional[str] = None,
):
    global _config
    api_key = api_key or os.environ.get("BLACKBOX_API_KEY", "")
    project_id = project_id or os.environ.get("BLACKBOX_PROJECT_ID", "")
    base_url = base_url or os.environ.get("BLACKBOX_BASE_URL", "http://localhost:4000")
    _config = Config(api_key=api_key, project_id=project_id, base_url=base_url)


def trace(name: str, input: Optional[Dict[str, Any]] = None, **kwargs) -> TraceContext:
    config = _get_config()
    ctx = TraceContext(
        trace_id=str(uuid.uuid4()),
        project_id=config.project_id,
        name=name,
        start_time=time.time(),
        input=input,
    )
    return ctx


class Blackbox:
    def __init__(
        self,
        api_key: Optional[str] = None,
        project_id: Optional[str] = None,
        base_url: Optional[str] = None,
    ):
        init(api_key=api_key, project_id=project_id, base_url=base_url)

    def trace(self, name: str, input: Optional[Dict[str, Any]] = None, **kwargs) -> TraceContext:
        return trace(name, input=input, **kwargs)

    def generation(self, name: str, model: str = "", **kwargs):
        return lambda trace: trace.generation(name, model=model, **kwargs)

    def tool(self, name: str, **kwargs):
        return lambda trace: trace.tool(name, **kwargs)

    def retrieval(self, name: str, **kwargs):
        return lambda trace: trace.retrieval(name, **kwargs)