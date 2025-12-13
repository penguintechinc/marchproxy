import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Grid,
  Typography,
  Button,
  Tabs,
  Tab,
  Alert,
  Chip,
  Stack,
} from '@mui/material';
import {
  Shield,
  Policy,
  Assessment,
  Description,
  Security,
} from '@mui/icons-material';
import PolicyEditor from '../../components/Enterprise/PolicyEditor';
import PolicyTester from '../../components/Enterprise/PolicyTester';
import AuditLogViewer from '../../components/Enterprise/AuditLogViewer';
import ComplianceReports from '../../components/Enterprise/ComplianceReports';
import { apiClient } from '../../services/api';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`zerotrust-tabpanel-${index}`}
      aria-labelledby={`zerotrust-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const ZeroTrust: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [licenseStatus, setLicenseStatus] = useState<{
    valid: boolean;
    tier: string;
  } | null>(null);
  const [zeroTrustStatus, setZeroTrustStatus] = useState<{
    enabled: boolean;
    opaConnected: boolean;
    auditChainValid: boolean;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchStatus();
  }, []);

  const fetchStatus = async () => {
    try {
      setLoading(true);
      setError(null);

      // Check license status
      const licenseResp = await apiClient.get('/api/v1/license/status');
      setLicenseStatus(licenseResp.data);

      // Check zero-trust status
      const ztResp = await apiClient.get('/api/v1/zerotrust/status');
      setZeroTrustStatus(ztResp.data);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch status');
    } finally {
      setLoading(false);
    }
  };

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const handleToggleZeroTrust = async () => {
    try {
      const newStatus = !zeroTrustStatus?.enabled;
      await apiClient.post('/api/v1/zerotrust/toggle', {
        enabled: newStatus,
      });
      await fetchStatus();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to toggle zero-trust');
    }
  };

  // Check if Enterprise license is valid
  const isEnterprise = licenseStatus?.tier === 'Enterprise';

  if (!isEnterprise && !loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="warning">
          <Typography variant="h6" gutterBottom>
            Enterprise Feature
          </Typography>
          <Typography>
            Zero-Trust Security requires an Enterprise license. Please upgrade
            your license to access this feature.
          </Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ flexGrow: 1 }}>
      {/* Header */}
      <Box sx={{ mb: 3, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <Shield sx={{ fontSize: 40, mr: 2, color: 'primary.main' }} />
          <div>
            <Typography variant="h4" component="h1">
              Zero-Trust Security
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Policy-based access control with OPA integration and immutable audit logging
            </Typography>
          </div>
        </Box>
        <Button
          variant="contained"
          color={zeroTrustStatus?.enabled ? 'error' : 'success'}
          onClick={handleToggleZeroTrust}
          disabled={loading}
        >
          {zeroTrustStatus?.enabled ? 'Disable' : 'Enable'} Zero-Trust
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Status Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Stack direction="row" spacing={1} alignItems="center">
                <Policy color="primary" />
                <Typography variant="h6">Policy Enforcement</Typography>
              </Stack>
              <Typography variant="h3" sx={{ mt: 2 }}>
                {zeroTrustStatus?.enabled ? (
                  <Chip label="Enabled" color="success" />
                ) : (
                  <Chip label="Disabled" color="default" />
                )}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Stack direction="row" spacing={1} alignItems="center">
                <Security color="primary" />
                <Typography variant="h6">OPA Connection</Typography>
              </Stack>
              <Typography variant="h3" sx={{ mt: 2 }}>
                {zeroTrustStatus?.opaConnected ? (
                  <Chip label="Connected" color="success" />
                ) : (
                  <Chip label="Disconnected" color="error" />
                )}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Stack direction="row" spacing={1} alignItems="center">
                <Assessment color="primary" />
                <Typography variant="h6">Audit Chain</Typography>
              </Stack>
              <Typography variant="h3" sx={{ mt: 2 }}>
                {zeroTrustStatus?.auditChainValid ? (
                  <Chip label="Valid" color="success" />
                ) : (
                  <Chip label="Invalid" color="error" />
                )}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Card>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={tabValue} onChange={handleTabChange}>
            <Tab icon={<Policy />} label="Policy Editor" />
            <Tab icon={<Description />} label="Policy Tester" />
            <Tab icon={<Assessment />} label="Audit Logs" />
            <Tab icon={<Description />} label="Compliance Reports" />
          </Tabs>
        </Box>

        <TabPanel value={tabValue} index={0}>
          <PolicyEditor />
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <PolicyTester />
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <AuditLogViewer />
        </TabPanel>

        <TabPanel value={tabValue} index={3}>
          <ComplianceReports />
        </TabPanel>
      </Card>
    </Box>
  );
};

export default ZeroTrust;
