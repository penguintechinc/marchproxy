"""Unit tests for ConfigSync service"""

import pytest
import json
import yaml
import os
import tempfile
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock, mock_open, call
from datetime import datetime

from sync import ConfigSync


class TestConfigSync:
    """Test suite for ConfigSync class"""

    @pytest.fixture
    def mock_env(self):
        """Fixture for mocking environment variables"""
        with patch.dict(os.environ, {
            'MANAGER_URL': 'http://test-manager:8000',
            'CLUSTER_API_KEY': 'test-api-key-123',
            'SYNC_INTERVAL': '60',
        }):
            yield

    @pytest.fixture
    def config_sync(self, mock_env):
        """Fixture for ConfigSync instance"""
        with patch.object(Path, 'mkdir'):
            return ConfigSync()

    @pytest.fixture
    def sample_config(self):
        """Fixture for sample manager configuration"""
        return {
            'monitoring': {
                'smtp': {
                    'host': 'smtp.example.com',
                    'port': 587,
                    'from': 'alerts@example.com',
                    'username': 'user@example.com',
                    'password': 'secret-password',
                },
                'alerts': {
                    'default_email': 'ops@example.com',
                    'critical_email': 'critical@example.com',
                    'license_email': 'license@example.com',
                    'performance_email': 'perf@example.com',
                    'security_email': 'security@example.com',
                    'slack_webhook': 'https://hooks.slack.com/services/xyz',
                    'pagerduty_url': 'https://events.pagerduty.com/v2/enqueue',
                },
                'external_datasources': [
                    {
                        'enabled': True,
                        'name': 'Datadog',
                        'type': 'datadog',
                        'url': 'https://api.datadoghq.com',
                        'username': 'user',
                        'password': 'pass',
                        'basic_auth': True,
                    }
                ],
            },
            'proxies': [
                {
                    'hostname': 'proxy-1',
                    'metrics_port': 8081,
                    'status': 'active',
                },
                {
                    'hostname': 'proxy-2',
                    'metrics_port': 8081,
                    'status': 'active',
                },
                {
                    'hostname': 'proxy-3',
                    'metrics_port': 8081,
                    'status': 'inactive',
                },
            ],
            'manager': {
                'hostname': 'manager',
            },
        }

    def test_init_default_values(self, mock_env):
        """Test ConfigSync initialization with default values"""
        with patch.object(Path, 'mkdir'):
            sync = ConfigSync()
            assert sync.manager_url == 'http://test-manager:8000'
            assert sync.cluster_api_key == 'test-api-key-123'
            assert sync.sync_interval == 60

    def test_init_fallback_env_values(self):
        """Test ConfigSync initialization with fallback environment values"""
        with patch.dict(os.environ, {}, clear=True):
            with patch.object(Path, 'mkdir'):
                sync = ConfigSync()
                assert sync.manager_url == 'http://manager:8000'
                assert sync.cluster_api_key == 'default-api-key'
                assert sync.sync_interval == 300

    def test_init_creates_config_directory(self):
        """Test that ConfigSync creates config directory on initialization"""
        with patch.dict(os.environ, {
            'MANAGER_URL': 'http://manager:8000',
            'CLUSTER_API_KEY': 'test-key',
            'SYNC_INTERVAL': '300',
        }):
            with patch.object(Path, 'mkdir') as mock_mkdir:
                ConfigSync()
                mock_mkdir.assert_called_once_with(parents=True, exist_ok=True)

    @patch('sync.requests.get')
    def test_get_manager_config_success(self, mock_get, config_sync, sample_config):
        """Test successful manager config retrieval"""
        mock_response = Mock()
        mock_response.json.return_value = sample_config
        mock_get.return_value = mock_response

        result = config_sync.get_manager_config()

        assert result == sample_config
        mock_get.assert_called_once()
        call_args = mock_get.call_args
        assert 'http://test-manager:8000/api/v1/monitoring/config' in str(call_args)
        assert call_args.kwargs['headers']['Authorization'] == 'Bearer test-api-key-123'

    @patch('sync.requests.get')
    def test_get_manager_config_network_error(self, mock_get, config_sync):
        """Test manager config retrieval with network error"""
        mock_get.side_effect = Exception('Connection refused')

        result = config_sync.get_manager_config()

        assert result == {}

    @patch('sync.requests.get')
    def test_get_manager_config_http_error(self, mock_get, config_sync):
        """Test manager config retrieval with HTTP error"""
        mock_response = Mock()
        mock_response.raise_for_status.side_effect = Exception('401 Unauthorized')
        mock_get.return_value = mock_response

        result = config_sync.get_manager_config()

        assert result == {}

    @patch('builtins.open', new_callable=mock_open)
    def test_update_alertmanager_config(self, mock_file, config_sync, sample_config):
        """Test AlertManager configuration update"""
        config_sync.update_alertmanager_config(sample_config)

        # Verify file was opened for writing
        mock_file.assert_called_once()
        handle = mock_file()

        # Verify environment variables were written
        written_content = ''.join(call.args[0] for call in handle.write.call_args_list)
        assert 'SMTP_HOST=smtp.example.com' in written_content
        assert 'SMTP_PORT=587' in written_content
        assert 'SMTP_FROM=alerts@example.com' in written_content
        assert 'ALERT_EMAIL_DEFAULT=ops@example.com' in written_content
        assert 'SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xyz' in written_content

    @patch('builtins.open', new_callable=mock_open)
    def test_update_alertmanager_config_missing_fields(self, mock_file, config_sync):
        """Test AlertManager config with missing fields uses defaults"""
        config = {'monitoring': {'smtp': {}, 'alerts': {}}}

        config_sync.update_alertmanager_config(config)

        mock_file.assert_called_once()
        handle = mock_file()
        written_content = ''.join(call.args[0] for call in handle.write.call_args_list)

        # Should use defaults
        assert 'SMTP_HOST=localhost' in written_content
        assert 'SMTP_PORT=587' in written_content
        assert 'ALERT_EMAIL_DEFAULT=ops-team@company.com' in written_content

    @patch('builtins.open', new_callable=mock_open)
    def test_update_alertmanager_config_error_handling(self, mock_file, config_sync):
        """Test AlertManager config error handling"""
        mock_file.side_effect = IOError('Permission denied')

        # Should not raise exception
        config_sync.update_alertmanager_config({'monitoring': {}})

    @patch('builtins.open', new_callable=mock_open)
    @patch('json.dump')
    def test_update_prometheus_targets(self, mock_json_dump, mock_file, config_sync, sample_config):
        """Test Prometheus targets update"""
        config_sync.update_prometheus_targets(sample_config)

        # Verify file was opened
        mock_file.assert_called_once()

        # Verify JSON was dumped
        mock_json_dump.assert_called_once()
        dumped_config = mock_json_dump.call_args[0][0]

        # Verify only active proxies are included
        assert len(dumped_config['proxy_targets']) == 2
        assert 'proxy-1:8081' in dumped_config['proxy_targets']
        assert 'proxy-2:8081' in dumped_config['proxy_targets']

    @patch('builtins.open', new_callable=mock_open)
    @patch('json.dump')
    def test_update_prometheus_targets_no_proxies(self, mock_json_dump, mock_file, config_sync):
        """Test Prometheus targets with no proxies"""
        config = {'proxies': [], 'manager': {}}

        config_sync.update_prometheus_targets(config)

        dumped_config = mock_json_dump.call_args[0][0]
        assert dumped_config['proxy_targets'] == []

    @patch('builtins.open', new_callable=mock_open)
    @patch('json.dump')
    def test_update_prometheus_targets_error_handling(self, mock_json_dump, mock_file, config_sync):
        """Test Prometheus targets error handling"""
        mock_json_dump.side_effect = Exception('JSON encoding error')

        # Should not raise exception
        config_sync.update_prometheus_targets({'proxies': [], 'manager': {}})

    @patch('builtins.open', new_callable=mock_open)
    @patch('yaml.dump')
    def test_update_grafana_datasources(self, mock_yaml_dump, mock_file, config_sync, sample_config):
        """Test Grafana datasources update"""
        config_sync.update_grafana_datasources(sample_config)

        # Verify file was opened
        mock_file.assert_called_once()

        # Verify YAML was dumped
        mock_yaml_dump.assert_called_once()
        dumped_config = mock_yaml_dump.call_args[0][0]

        # Verify default datasources
        assert dumped_config['apiVersion'] == 1
        datasources = dumped_config['datasources']
        names = [ds['name'] for ds in datasources]
        assert 'Prometheus' in names
        assert 'Loki' in names

        # Verify external datasources added
        assert len(datasources) >= 3  # At least Prometheus, Loki, + Datadog

    @patch('builtins.open', new_callable=mock_open)
    @patch('yaml.dump')
    def test_update_grafana_datasources_disabled_external(self, mock_yaml_dump, mock_file, config_sync):
        """Test Grafana datasources with disabled external datasource"""
        config = {
            'monitoring': {
                'external_datasources': [
                    {
                        'enabled': False,
                        'name': 'Disabled',
                        'type': 'datadog',
                        'url': 'https://example.com',
                    }
                ]
            }
        }

        config_sync.update_grafana_datasources(config)

        dumped_config = mock_yaml_dump.call_args[0][0]
        names = [ds['name'] for ds in dumped_config['datasources']]
        assert 'Disabled' not in names

    @patch('builtins.open', new_callable=mock_open)
    @patch('yaml.dump')
    def test_update_grafana_datasources_basic_auth(self, mock_yaml_dump, mock_file, config_sync):
        """Test Grafana datasources with basic auth"""
        config = {
            'monitoring': {
                'external_datasources': [
                    {
                        'enabled': True,
                        'name': 'Secure DS',
                        'type': 'custom',
                        'url': 'https://example.com',
                        'basic_auth': True,
                        'username': 'admin',
                        'password': 'secret',
                    }
                ]
            }
        }

        config_sync.update_grafana_datasources(config)

        dumped_config = mock_yaml_dump.call_args[0][0]
        secure_ds = next(ds for ds in dumped_config['datasources'] if ds['name'] == 'Secure DS')

        assert secure_ds['basicAuth'] is True
        assert secure_ds['basicAuthUser'] == 'admin'
        assert secure_ds['secureJsonData']['basicAuthPassword'] == 'secret'

    @patch('builtins.open', new_callable=mock_open)
    @patch('yaml.dump')
    def test_update_grafana_datasources_error_handling(self, mock_yaml_dump, mock_file, config_sync):
        """Test Grafana datasources error handling"""
        mock_yaml_dump.side_effect = yaml.YAMLError('YAML error')

        # Should not raise exception
        config_sync.update_grafana_datasources({'monitoring': {}})

    @patch('sync.ConfigSync.get_manager_config')
    @patch('sync.ConfigSync.update_alertmanager_config')
    @patch('sync.ConfigSync.update_prometheus_targets')
    @patch('sync.ConfigSync.update_grafana_datasources')
    @patch('builtins.open', new_callable=mock_open)
    @patch('time.sleep')
    def test_sync_configuration_success(
        self,
        mock_sleep,
        mock_file,
        mock_update_grafana,
        mock_update_prometheus,
        mock_update_alertmanager,
        mock_get_config,
        config_sync,
        sample_config
    ):
        """Test successful configuration sync cycle"""
        # Make sleep raise exception after first call to break the loop
        mock_sleep.side_effect = KeyboardInterrupt()
        mock_get_config.return_value = sample_config

        with pytest.raises(KeyboardInterrupt):
            config_sync.sync_configuration()

        # Verify all update methods were called
        mock_get_config.assert_called()
        mock_update_alertmanager.assert_called_once_with(sample_config)
        mock_update_prometheus.assert_called_once_with(sample_config)
        mock_update_grafana.assert_called_once_with(sample_config)

        # Verify timestamp was written
        mock_file.assert_called()

    @patch('sync.ConfigSync.get_manager_config')
    @patch('time.sleep')
    def test_sync_configuration_empty_config(
        self,
        mock_sleep,
        mock_get_config,
        config_sync
    ):
        """Test sync with empty configuration"""
        mock_sleep.side_effect = KeyboardInterrupt()
        mock_get_config.return_value = {}

        with pytest.raises(KeyboardInterrupt):
            config_sync.sync_configuration()

        # get_manager_config should be called but nothing else
        mock_get_config.assert_called()

    @patch('sync.ConfigSync.get_manager_config')
    @patch('time.sleep')
    def test_sync_configuration_exception_handling(
        self,
        mock_sleep,
        mock_get_config,
        config_sync
    ):
        """Test sync handles exceptions and continues"""
        mock_sleep.side_effect = [None, KeyboardInterrupt()]
        mock_get_config.side_effect = [Exception('API error'), {}]

        with pytest.raises(KeyboardInterrupt):
            config_sync.sync_configuration()

        # Should have called get_config twice (once failed, once on retry after sleep)
        assert mock_get_config.call_count >= 1


class TestConfigSyncIntegration:
    """Integration tests for ConfigSync"""

    def test_full_sync_workflow(self):
        """Test complete sync workflow with mocked HTTP"""
        with patch.dict(os.environ, {
            'MANAGER_URL': 'http://localhost:8000',
            'CLUSTER_API_KEY': 'test-key',
            'SYNC_INTERVAL': '10',
        }):
            with patch.object(Path, 'mkdir'):
                with tempfile.TemporaryDirectory() as tmpdir:
                    config_sync = ConfigSync()
                    config_sync.config_dir = Path(tmpdir)

                    sample_config = {
                        'monitoring': {
                            'smtp': {'host': 'smtp.local', 'port': 25},
                            'alerts': {'default_email': 'admin@local'},
                            'external_datasources': [],
                        },
                        'proxies': [
                            {'hostname': 'proxy-1', 'metrics_port': 8081, 'status': 'active'}
                        ],
                        'manager': {'hostname': 'manager'},
                    }

                    with patch.object(
                        config_sync, 'get_manager_config', return_value=sample_config
                    ):
                        config_sync.update_alertmanager_config(sample_config)
                        config_sync.update_prometheus_targets(sample_config)
                        config_sync.update_grafana_datasources(sample_config)

                        # Verify files were created
                        assert (Path(tmpdir) / 'alertmanager.env').exists()
                        assert (Path(tmpdir) / 'prometheus_targets.json').exists()
                        assert (Path(tmpdir) / 'grafana_datasources.yaml').exists()

                        # Verify content
                        with open(Path(tmpdir) / 'alertmanager.env') as f:
                            env_content = f.read()
                            assert 'SMTP_HOST=smtp.local' in env_content

                        with open(Path(tmpdir) / 'prometheus_targets.json') as f:
                            targets = json.load(f)
                            assert 'proxy-1:8081' in targets['proxy_targets']

                        with open(Path(tmpdir) / 'grafana_datasources.yaml') as f:
                            datasources = yaml.safe_load(f)
                            assert datasources['apiVersion'] == 1
