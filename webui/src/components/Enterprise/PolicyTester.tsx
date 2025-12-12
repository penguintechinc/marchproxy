import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Alert,
  Stack,
  Typography,
  Chip,
  Grid,
} from '@mui/material';
import { PlayArrow, CheckCircle, Cancel } from '@mui/icons-material';
import Editor from '@monaco-editor/react';
import { apiClient } from '../../services/api';

interface TestResult {
  allowed: boolean;
  deny: boolean;
  reason?: string;
  annotations?: Record<string, any>;
  rate_limit?: {
    limit: number;
    remaining: number;
    window: string;
  };
}

const PolicyTester: React.FC = () => {
  const [policies, setPolicies] = useState<string[]>([]);
  const [selectedPolicy, setSelectedPolicy] = useState<string>('');
  const [inputJSON, setInputJSON] = useState<string>(`{
  "service": "api-gateway",
  "user": "john.doe",
  "action": "read",
  "resource": "/api/users",
  "source_ip": "192.168.1.100",
  "timestamp": "${new Date().toISOString()}",
  "metadata": {
    "auth_method": "mtls",
    "tls_enabled": true
  }
}`);
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchPolicies();
  }, []);

  const fetchPolicies = async () => {
    try {
      const response = await apiClient.get('/api/v1/zerotrust/policies');
      const policyNames = response.data.policies?.map((p: any) => p.name) || [];
      setPolicies(policyNames);
      if (policyNames.length > 0) {
        setSelectedPolicy(policyNames[0]);
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch policies');
    }
  };

  const handleTest = async () => {
    try {
      setLoading(true);
      setError(null);
      setTestResult(null);

      // Parse input JSON
      let input;
      try {
        input = JSON.parse(inputJSON);
      } catch (e) {
        throw new Error('Invalid JSON input');
      }

      const response = await apiClient.post('/api/v1/zerotrust/policies/test', {
        policy: selectedPolicy,
        input: input,
      });

      setTestResult(response.data.result);
    } catch (err: any) {
      setError(err.response?.data?.message || err.message || 'Failed to test policy');
    } finally {
      setLoading(false);
    }
  };

  const formatJSON = () => {
    try {
      const parsed = JSON.parse(inputJSON);
      setInputJSON(JSON.stringify(parsed, null, 2));
    } catch (e) {
      setError('Invalid JSON');
    }
  };

  const loadSample = (sample: string) => {
    const samples: Record<string, any> = {
      'authenticated-access': {
        service: 'api-gateway',
        user: 'john.doe',
        action: 'read',
        resource: '/api/users',
        source_ip: '192.168.1.100',
        timestamp: new Date().toISOString(),
        metadata: {
          auth_method: 'mtls',
          tls_enabled: true,
        },
      },
      'unauthenticated-access': {
        service: '',
        user: '',
        action: 'read',
        resource: '/api/users',
        source_ip: '10.0.0.50',
        timestamp: new Date().toISOString(),
        metadata: {
          auth_method: 'none',
        },
      },
      'admin-access': {
        service: 'admin-portal',
        user: 'admin',
        action: 'write',
        resource: '/api/config',
        source_ip: '192.168.1.1',
        timestamp: new Date().toISOString(),
        metadata: {
          auth_method: 'jwt',
          jwt_validated: true,
        },
      },
    };

    if (samples[sample]) {
      setInputJSON(JSON.stringify(samples[sample], null, 2));
    }
  };

  return (
    <Box>
      <Grid container spacing={3}>
        {/* Input Section */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Test Input
              </Typography>

              <Stack spacing={2}>
                <FormControl fullWidth>
                  <InputLabel>Policy</InputLabel>
                  <Select
                    value={selectedPolicy}
                    onChange={(e) => setSelectedPolicy(e.target.value)}
                    label="Policy"
                  >
                    {policies.map((policy) => (
                      <MenuItem key={policy} value={policy}>
                        {policy}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Sample Inputs:
                  </Typography>
                  <Stack direction="row" spacing={1}>
                    <Button size="small" onClick={() => loadSample('authenticated-access')}>
                      Authenticated
                    </Button>
                    <Button size="small" onClick={() => loadSample('unauthenticated-access')}>
                      Unauthenticated
                    </Button>
                    <Button size="small" onClick={() => loadSample('admin-access')}>
                      Admin
                    </Button>
                  </Stack>
                </Box>

                <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1 }}>
                  <Editor
                    height="400px"
                    language="json"
                    theme="vs-dark"
                    value={inputJSON}
                    onChange={(value) => setInputJSON(value || '')}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 14,
                      lineNumbers: 'on',
                    }}
                  />
                </Box>

                <Stack direction="row" spacing={2}>
                  <Button
                    variant="contained"
                    startIcon={<PlayArrow />}
                    onClick={handleTest}
                    disabled={loading || !selectedPolicy}
                  >
                    Test Policy
                  </Button>
                  <Button onClick={formatJSON}>Format JSON</Button>
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        {/* Result Section */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Test Result
              </Typography>

              {error && (
                <Alert severity="error" onClose={() => setError(null)}>
                  {error}
                </Alert>
              )}

              {testResult && (
                <Stack spacing={2}>
                  <Box>
                    {testResult.allowed ? (
                      <Stack direction="row" spacing={1} alignItems="center">
                        <CheckCircle sx={{ color: 'success.main', fontSize: 40 }} />
                        <div>
                          <Typography variant="h5">Access Allowed</Typography>
                          <Chip label="ALLOW" color="success" size="small" />
                        </div>
                      </Stack>
                    ) : (
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Cancel sx={{ color: 'error.main', fontSize: 40 }} />
                        <div>
                          <Typography variant="h5">Access Denied</Typography>
                          <Chip label="DENY" color="error" size="small" />
                        </div>
                      </Stack>
                    )}
                  </Box>

                  {testResult.reason && (
                    <Box>
                      <Typography variant="subtitle2" color="text.secondary">
                        Reason:
                      </Typography>
                      <Typography>{testResult.reason}</Typography>
                    </Box>
                  )}

                  {testResult.annotations && Object.keys(testResult.annotations).length > 0 && (
                    <Box>
                      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                        Annotations:
                      </Typography>
                      <Box
                        sx={{
                          p: 2,
                          bgcolor: 'background.default',
                          borderRadius: 1,
                          fontFamily: 'monospace',
                          fontSize: '0.875rem',
                        }}
                      >
                        <pre>{JSON.stringify(testResult.annotations, null, 2)}</pre>
                      </Box>
                    </Box>
                  )}

                  {testResult.rate_limit && (
                    <Box>
                      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                        Rate Limit Info:
                      </Typography>
                      <Stack spacing={1}>
                        <Typography variant="body2">
                          Limit: {testResult.rate_limit.limit} requests per {testResult.rate_limit.window}
                        </Typography>
                        <Typography variant="body2">
                          Remaining: {testResult.rate_limit.remaining}
                        </Typography>
                      </Stack>
                    </Box>
                  )}

                  <Box>
                    <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                      Full Response:
                    </Typography>
                    <Box
                      sx={{
                        p: 2,
                        bgcolor: 'background.default',
                        borderRadius: 1,
                        fontFamily: 'monospace',
                        fontSize: '0.875rem',
                        maxHeight: 300,
                        overflow: 'auto',
                      }}
                    >
                      <pre>{JSON.stringify(testResult, null, 2)}</pre>
                    </Box>
                  </Box>
                </Stack>
              )}

              {!testResult && !error && (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <Typography color="text.secondary">
                    Run a test to see results
                  </Typography>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default PolicyTester;
