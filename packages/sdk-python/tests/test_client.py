import pytest
from unittest.mock import patch, MagicMock
import json

from blackbox_agentdiff import (
    init,
    trace,
    Blackbox,
    BlackboxError,
)


@pytest.fixture(autouse=True)
def reset_env(monkeypatch):
    monkeypatch.setenv("BLACKBOX_API_KEY", "test-key")
    monkeypatch.setenv("BLACKBOX_PROJECT_ID", "test-project")
    monkeypatch.setenv("BLACKBOX_BASE_URL", "http://localhost:4000")


class TestConfig:
    def test_resolve_config(self):
        from blackbox_agentdiff.client import resolveConfig
        config = resolveConfig()
        assert config.api_key == "test-key"
        assert config.project_id == "test-project"
        assert config.base_url == "http://localhost:4000"

    def test_resolve_config_overrides(self):
        from blackbox_agentdiff.client import resolveConfig
        config = resolveConfig({
            apiKey: "override-key",
            projectId: "override-project",
        })
        assert config.api_key == "override-key"
        assert config.project_id == "override-project"


class TestTraceContext:
    def test_trace_creation(self):
        ctx = trace("test-trace")
        assert ctx is not None
        assert ctx.name == "test-trace"
        assert ctx.traceId is not None

    def test_trace_with_input(self):
        ctx = trace("test", input={"msg": "hello"})
        assert ctx.input is not None
        assert ctx.input["msg"] == "hello"


class TestSpanRecording:
    def test_generation_record(self):
        ctx = trace("test")
        gen = ctx.generation("gpt-4o")
        gen.record(
            input=[{"role": "user", "content": "hi"}],
            output="hello",
            input_tokens=5,
            output_tokens=8,
        )
        assert gen.output == "hello"
        assert gen.output_tokens == 8
        assert gen.duration_ms is not None
        assert gen.end_time is not None

    def test_tool_record(self):
        ctx = trace("test")
        tool = ctx.tool("search")
        tool.record(output={"results": []})
        assert tool.output == {"results": []}


class TestExport:
    @patch("urllib.request.urlopen")
    def test_successful_export(self, mock_urlopen):
        ctx = trace("test", input={"msg": "hi"})
        gen = ctx.generation("gpt4", model="gpt-4o")
        gen.record(output="hello", input_tokens=1, output_tokens=2)

        mock_resp = MagicMock()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        ctx.end()

        assert mock_urlopen.called
        req = mock_urlopen.call_args[0][0]
        assert req.get_header("Authorization") == "Bearer test-key"
        body = json.loads(req.data)
        assert "resourceSpans" in body

    @patch("urllib.request.urlopen")
    def test_export_failure(self, mock_urlopen):
        from urllib.error import HTTPError
        mock_urlopen.side_effect = HTTPError(
            "http://localhost:4000/otel/v1/traces",
            401,
            "Unauthorized",
            {},
            None,
        )
        ctx = trace("test")
        gen = ctx.generation("gpt4")
        gen.record()
        with pytest.raises(BlackboxError):
            ctx.end()