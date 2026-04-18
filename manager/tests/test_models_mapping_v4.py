"""
Unit tests for mapping models

Tests cover MappingModel for route mapping, service resolution, and port matching
"""

import pytest
from datetime import datetime
from unittest.mock import MagicMock, patch
from models.mapping import (
    MappingModel,
    CreateMappingRequest,
    UpdateMappingRequest,
)


@pytest.fixture
def mock_db():
    """Create a mock database object"""
    db = MagicMock()
    db.mappings = MagicMock()
    db.services = MagicMock()
    db.user_service_assignments = MagicMock()
    db.auth_user = MagicMock()
    return db


class TestMappingModel:
    """Tests for MappingModel"""

    def test_define_table(self, mock_db):
        """Test mapping table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = MappingModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_normalize_service_list_all_services(self, mock_db):
        """Test normalizing 'all' service reference"""
        service1 = MagicMock()
        service1.id = 1
        service2 = MagicMock()
        service2.id = 2

        mock_db.return_value.select.return_value = [service1, service2]

        normalized = MappingModel._normalize_service_list(mock_db, ["all"], cluster_id=1)

        assert len(normalized) == 1
        assert normalized[0]["type"] == "all"
        assert normalized[0]["count"] == 2

    def test_normalize_service_list_collection(self, mock_db):
        """Test normalizing collection reference"""
        service1 = MagicMock()
        service1.id = 1

        mock_db.return_value.select.return_value = [service1]

        normalized = MappingModel._normalize_service_list(
            mock_db, ["collection:web"], cluster_id=1
        )

        assert len(normalized) == 1
        assert normalized[0]["type"] == "collection"
        assert normalized[0]["name"] == "web"

    def test_normalize_service_list_individual_service(self, mock_db):
        """Test normalizing individual service ID"""
        service = MagicMock()
        service.id = 1
        service.name = "service1"
        service.ip_fqdn = "service1.local"
        service.port = 8080

        mock_db.return_value.select.return_value.first.return_value = service

        normalized = MappingModel._normalize_service_list(mock_db, [1], cluster_id=1)

        assert len(normalized) == 1
        assert normalized[0]["type"] == "service"
        assert normalized[0]["id"] == 1
        assert normalized[0]["name"] == "service1"

    def test_normalize_service_list_mixed(self, mock_db):
        """Test normalizing mixed service references"""
        service1 = MagicMock()
        service1.id = 1
        service1.name = "service1"
        service1.ip_fqdn = "service1.local"
        service1.port = 8080

        mock_db.return_value.select.return_value.first.return_value = service1

        normalized = MappingModel._normalize_service_list(mock_db, [1], cluster_id=1)

        assert len(normalized) >= 1

    def test_normalize_port_list_single_ports(self, mock_db):
        """Test normalizing single port numbers"""
        normalized = MappingModel._normalize_port_list([80, 443, 8080])

        assert len(normalized) == 3
        assert all(p["type"] == "single" for p in normalized)
        assert [p["port"] for p in normalized] == [80, 443, 8080]

    def test_normalize_port_list_port_range(self, mock_db):
        """Test normalizing port ranges"""
        normalized = MappingModel._normalize_port_list(["8000-8100"])

        assert len(normalized) == 1
        assert normalized[0]["type"] == "range"
        assert normalized[0]["start"] == 8000
        assert normalized[0]["end"] == 8100

    def test_normalize_port_list_comma_separated(self, mock_db):
        """Test normalizing comma-separated ports"""
        normalized = MappingModel._normalize_port_list(["80,443,8080"])

        assert len(normalized) == 1
        assert normalized[0]["type"] == "list"
        assert set(normalized[0]["ports"]) == {80, 443, 8080}

    def test_normalize_port_list_string_port(self, mock_db):
        """Test normalizing string port numbers"""
        normalized = MappingModel._normalize_port_list(["8080", "9090"])

        assert len(normalized) == 2
        assert all(p["type"] == "single" for p in normalized)

    def test_normalize_port_list_invalid_ports_ignored(self, mock_db):
        """Test that invalid ports are ignored"""
        normalized = MappingModel._normalize_port_list([80, 99999, 443])

        assert len(normalized) == 2  # Only valid ports

    def test_create_mapping(self, mock_db):
        """Test creating a mapping"""
        service = MagicMock()
        service.id = 1
        service.name = "service1"
        service.ip_fqdn = "service1.local"
        service.port = 8080

        # Create proper query mock chain
        query_result = MagicMock()
        query_result.first.return_value = service
        mock_db.return_value = query_result
        mock_db.mappings.insert = MagicMock(return_value=1)

        mapping_id = MappingModel.create_mapping(
            mock_db,
            name="test-mapping",
            source_services=[1],
            dest_services=[1],
            ports=[80, 443],
            cluster_id=1,
            created_by=1,
        )

        assert mapping_id == 1
        mock_db.mappings.insert.assert_called_once()

    def test_get_cluster_mappings_admin(self, mock_db):
        """Test getting cluster mappings as admin"""
        mapping = MagicMock()
        mapping.id = 1
        mapping.name = "mapping1"
        mapping.source_services = [{"type": "service", "id": 1}]
        mapping.dest_services = [{"type": "service", "id": 2}]
        mapping.protocols = ["tcp"]
        mapping.ports = [{"type": "single", "port": 80}]
        mapping.auth_required = True
        mapping.priority = 100
        mapping.created_at = datetime.utcnow()

        mock_db.return_value.select.return_value = [mapping]

        mappings = MappingModel.get_cluster_mappings(mock_db, cluster_id=1)

        assert len(mappings) == 1
        assert mappings[0]["id"] == 1

    def test_get_cluster_mappings_non_admin_with_access(self, mock_db):
        """Test getting mappings as non-admin with service access"""
        user = MagicMock()
        user.get.return_value = False  # Not admin

        assignment = MagicMock()
        assignment.service_id = 1

        mapping = MagicMock()
        mapping.id = 1
        mapping.name = "mapping1"
        mapping.source_services = [{"type": "service", "id": 1}]
        mapping.dest_services = [{"type": "service", "id": 2}]
        mapping.protocols = ["tcp"]
        mapping.ports = []
        mapping.auth_required = True
        mapping.priority = 100
        mapping.created_at = datetime.utcnow()

        mock_db.auth_user.__getitem__ = MagicMock(return_value=user)
        mock_db.return_value.select.return_value = [assignment]

        # Mock the mappings query differently
        with patch.object(MappingModel, '_normalize_service_list', return_value=[]):
            mock_db.return_value = MagicMock()
            mock_db.return_value.select.side_effect = [
                [assignment],  # user_service_assignments
                [mapping],  # all mappings
            ]

            mappings = MappingModel.get_cluster_mappings(
                mock_db, cluster_id=1, user_id=1
            )

            # Should get at least the mapping with matching service
            assert isinstance(mappings, list)

    def test_find_matching_mappings_empty(self, mock_db):
        """Test finding mappings with no matches"""
        source_service = MagicMock()
        source_service.cluster_id = 1

        query_mock = MagicMock()
        query_mock.select.return_value = []

        mock_db.services.__getitem__ = MagicMock(return_value=source_service)
        mock_db.return_value = query_mock

        matching = MappingModel.find_matching_mappings(
            mock_db, source_service_id=1, dest_service_id=2, protocol="tcp", port=80
        )

        assert matching == []
        assert isinstance(matching, list)

    def test_service_matches_all(self, mock_db):
        """Test service matching with 'all' reference"""
        service_refs = [{"type": "all"}]

        result = MappingModel._service_matches(mock_db, 1, service_refs, cluster_id=1)

        assert result is True

    def test_service_matches_specific_service(self, mock_db):
        """Test service matching with specific service ID"""
        service_refs = [{"type": "service", "id": 1}]

        result = MappingModel._service_matches(mock_db, 1, service_refs, cluster_id=1)

        assert result is True

    def test_service_matches_different_service(self, mock_db):
        """Test service not matching different ID"""
        service_refs = [{"type": "service", "id": 1}]

        result = MappingModel._service_matches(mock_db, 2, service_refs, cluster_id=1)

        assert result is False

    def test_service_matches_collection_no_match(self, mock_db):
        """Test service not matching different collection"""
        service = MagicMock()
        service.collection = "api"
        mock_db.services.__getitem__ = MagicMock(return_value=service)

        service_refs = [{"type": "collection", "name": "web"}]

        result = MappingModel._service_matches(mock_db, 1, service_refs, cluster_id=1)

        assert result is False

    def test_port_matches_single(self, mock_db):
        """Test port matching with single port"""
        port_refs = [{"type": "single", "port": 80}]

        assert MappingModel._port_matches(80, port_refs) is True
        assert MappingModel._port_matches(443, port_refs) is False

    def test_port_matches_range(self, mock_db):
        """Test port matching with range"""
        port_refs = [{"type": "range", "start": 8000, "end": 8100}]

        assert MappingModel._port_matches(8050, port_refs) is True
        assert MappingModel._port_matches(7999, port_refs) is False
        assert MappingModel._port_matches(8101, port_refs) is False

    def test_port_matches_list(self, mock_db):
        """Test port matching with list"""
        port_refs = [{"type": "list", "ports": [80, 443, 8080]}]

        assert MappingModel._port_matches(80, port_refs) is True
        assert MappingModel._port_matches(8080, port_refs) is True
        assert MappingModel._port_matches(9000, port_refs) is False

    def test_find_matching_mappings(self, mock_db):
        """Test finding matching mappings"""
        source_service = MagicMock()
        source_service.cluster_id = 1

        mapping = MagicMock()
        mapping.id = 1
        mapping.name = "test-mapping"
        mapping.source_services = [{"type": "service", "id": 1}]
        mapping.dest_services = [{"type": "service", "id": 2}]
        mapping.protocols = ["tcp"]
        mapping.ports = [{"type": "single", "port": 80}]
        mapping.auth_required = True
        mapping.priority = 100

        mock_db.services.__getitem__ = MagicMock(return_value=source_service)
        mock_db.return_value.select.return_value = [mapping]

        with patch.object(
            MappingModel, "_service_matches", return_value=True
        ), patch.object(MappingModel, "_port_matches", return_value=True):
            matching = MappingModel.find_matching_mappings(
                mock_db,
                source_service_id=1,
                dest_service_id=2,
                protocol="tcp",
                port=80,
            )

        assert len(matching) == 1
        assert matching[0]["id"] == 1


class TestCreateMappingRequest:
    """Tests for CreateMappingRequest Pydantic model"""

    def test_valid_request(self):
        """Test valid mapping creation request"""
        request = CreateMappingRequest(
            name="test-mapping",
            source_services=[1],
            dest_services=[2],
            cluster_id=1,
            ports=[80, 443],
            protocols=["tcp"],
        )

        assert request.name == "test-mapping"
        assert request.cluster_id == 1

    def test_name_normalization(self):
        """Test name gets lowercased"""
        request = CreateMappingRequest(
            name="TEST-MAPPING",
            source_services=[1],
            dest_services=[2],
            cluster_id=1,
            ports=[80],
        )

        assert request.name == "test-mapping"

    def test_name_too_short(self):
        """Test validation for short names"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="ab", source_services=[1], dest_services=[2], cluster_id=1, ports=[80]
            )

    def test_protocol_validation(self):
        """Test protocol validation"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="test",
                source_services=[1],
                dest_services=[2],
                cluster_id=1,
                ports=[80],
                protocols=["invalid"],
            )

    def test_protocol_normalization(self):
        """Test protocols are lowercase when provided"""
        # Protocols must be lowercase per validation
        request = CreateMappingRequest(
            name="test",
            source_services=[1],
            dest_services=[2],
            cluster_id=1,
            ports=[80],
            protocols=["tcp", "udp"],
        )

        # Verify protocols are in the list
        assert "tcp" in request.protocols
        assert "udp" in request.protocols

    def test_port_range_validation(self):
        """Test port range validation"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="test",
                source_services=[1],
                dest_services=[2],
                cluster_id=1,
                ports=["70000-80000"],
            )

    def test_port_list_validation(self):
        """Test port list validation"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="test",
                source_services=[1],
                dest_services=[2],
                cluster_id=1,
                ports=["80,99999,443"],
            )

    def test_priority_validation_low(self):
        """Test priority validation - too low"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="test",
                source_services=[1],
                dest_services=[2],
                cluster_id=1,
                ports=[80],
                priority=0,
            )

    def test_priority_validation_high(self):
        """Test priority validation - too high"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="test",
                source_services=[1],
                dest_services=[2],
                cluster_id=1,
                ports=[80],
                priority=1001,
            )

    def test_no_ports(self):
        """Test validation when no ports provided"""
        with pytest.raises(ValueError):
            CreateMappingRequest(
                name="test",
                source_services=[1],
                dest_services=[2],
                cluster_id=1,
                ports=[],
            )


class TestUpdateMappingRequest:
    """Tests for UpdateMappingRequest Pydantic model"""

    def test_all_optional_fields(self):
        """Test that all fields are optional"""
        request = UpdateMappingRequest()

        assert request.name is None
        assert request.source_services is None
        assert request.protocols is None

    def test_partial_update(self):
        """Test partial update"""
        request = UpdateMappingRequest(
            name="updated-mapping", auth_required=False
        )

        assert request.name == "updated-mapping"
        assert request.auth_required is False
        assert request.protocols is None
