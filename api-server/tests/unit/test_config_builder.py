"""Unit tests for app/services/config_builder.py"""
from unittest.mock import AsyncMock, MagicMock # noqa: F401

import pytest # noqa: F401, # noqa: F401


class TestParseServiceList:
    def setup_method(self):
        from app.services.config_builder import ConfigBuilder # noqa: F401
        self.builder = ConfigBuilder(MagicMock())

    def test_all_string_returns_list_with_all(self):
        result = self.builder._parse_service_list("all")
        assert result == ["all"]

    def test_all_uppercase_returns_list_with_all(self):
        result = self.builder._parse_service_list("ALL")
        assert result == ["all"]

    def test_single_id_returns_list(self):
        result = self.builder._parse_service_list("1")
        assert result == [1]

    def test_comma_separated_returns_list_of_ints(self):
        result = self.builder._parse_service_list("1,2,3")
        assert result == [1, 2, 3]

    def test_empty_string_returns_empty(self):
        result = self.builder._parse_service_list("")
        assert result == []

    def test_whitespace_stripped(self):
        result = self.builder._parse_service_list("1, 2, 3")
        assert result == [1, 2, 3]

    def test_invalid_string_returns_empty(self):
        result = self.builder._parse_service_list("invalid")
        assert result == []

    def test_single_zero_returns_zero(self):
        result = self.builder._parse_service_list("0")
        assert result == [0]


class TestParsePortConfig:
    def setup_method(self):
        from app.services.config_builder import ConfigBuilder # noqa: F401
        self.builder = ConfigBuilder(MagicMock())

    def test_single_port(self):
        result = self.builder._parse_port_config("80")
        assert 80 in result

    def test_single_port_returns_int(self):
        result = self.builder._parse_port_config("443")
        assert result == [443]

    def test_port_range_returns_dict(self):
        result = self.builder._parse_port_config("80-443")
        assert len(result) == 1
        assert isinstance(result[0], dict)
        assert result[0]["range"] == [80, 443]

    def test_multiple_ports(self):
        result = self.builder._parse_port_config("80,443,8080")
        assert result == [80, 443, 8080]

    def test_empty_string_returns_empty(self):
        result = self.builder._parse_port_config("")
        assert result == []

    def test_mixed_ports_and_ranges(self):
        result = self.builder._parse_port_config("80,443-8443,9000")
        assert len(result) == 3
        assert result[0] == 80
        assert result[1] == {"range": [443, 8443]}
        assert result[2] == 9000

    def test_port_with_whitespace(self):
        result = self.builder._parse_port_config("80, 443")
        assert 80 in result
        assert 443 in result

    def test_port_range_with_whitespace(self):
        result = self.builder._parse_port_config("80 - 443")
        assert len(result) == 1
        assert result[0]["range"] == [80, 443]


class TestGenerateConfigHash:
    def setup_method(self):
        from app.services.config_builder import ConfigBuilder # noqa: F401
        self.builder = ConfigBuilder(MagicMock())

    def test_returns_hex_string(self):
        result = self.builder._generate_config_hash({"key": "value"})
        assert isinstance(result, str)
        assert len(result) == 32  # MD5 hex digest length

    def test_deterministic(self):
        data = {"cluster": "test", "version": 1}
        r1 = self.builder._generate_config_hash(data)
        r2 = self.builder._generate_config_hash(data)
        assert r1 == r2

    def test_different_data_different_hash(self):
        r1 = self.builder._generate_config_hash({"a": 1})
        r2 = self.builder._generate_config_hash({"a": 2})
        assert r1 != r2

    def test_empty_dict_produces_hash(self):
        result = self.builder._generate_config_hash({})
        assert isinstance(result, str)
        assert len(result) == 32

    def test_nested_dict_produces_hash(self):
        data = {"outer": {"inner": [1, 2, 3]}, "key": "value"}
        result = self.builder._generate_config_hash(data)
        assert isinstance(result, str)
        assert len(result) == 32

    def test_key_order_independent(self):
        # json.dumps with sort_keys=True ensures stable ordering
        r1 = self.builder._generate_config_hash({"a": 1, "b": 2})
        r2 = self.builder._generate_config_hash({"b": 2, "a": 1})
        assert r1 == r2
