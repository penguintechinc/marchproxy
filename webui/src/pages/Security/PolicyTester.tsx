/**
 * Policy Testing Interface
 *
 * Simulates policy evaluation with input JSON, displays allow/deny
 * decisions, evaluation traces, and performance metrics.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Grid,
  Card,
  CardContent,
  Alert,
  Snackbar,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  Divider
} from '@mui/material';
import {
  PlayArrow as RunIcon,
  Save as SaveIcon,
  Upload as UploadIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
  Speed as PerformanceIcon
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';
import {
  OPAPolicy,
  PolicyTestRequest,
  PolicyTestResponse,
  getPolicies,
  testPolicy
} from '../../services/securityApi';
import LicenseGate from '../../components/Common/LicenseGate';

interface TestCase {
  id: string;
  name: string;
  description: string;
  input_json: object;
  expected_decision?: 'allow' | 'deny';
}

const PolicyTester: React.FC = () => {
  const hasEnterpriseAccess = true; // TODO: Get from license check
  const [policies, setPolicies] = useState<OPAPolicy[]>([]);
  const [selectedPolicy, setSelectedPolicy] = useState<number | null>(null);
  const [customRegoCode, setCustomRegoCode] = useState<string>('');
  const [useCustomCode, setUseCustomCode] = useState(false);
  const [inputJson, setInputJson] = useState<string>('{\n  "user": "alice",\n  "action": "read",\n  "resource": "document"\n}');
  const [testResult, setTestResult] = useState<PolicyTestResponse | null>(null);
  const [testCases, setTestCases] = useState<TestCase[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [csvDialogOpen, setCsvDialogOpen] = useState(false);
  const [bulkResults, setBulkResults] = useState<Array<{
    test_case: string;
    decision: string;
    time_ms: number;
  }>>([]);

  useEffect(() => {
    loadPolicies();
    loadTestCases();
  }, []);

  const loadPolicies = async () => {
    try {
      const data = await getPolicies();
      setPolicies(data.filter((p) => p.is_active));
    } catch (err: any) {
      console.error('Failed to load policies:', err);
    }
  };

  const loadTestCases = () => {
    // Load saved test cases from localStorage
    const saved = localStorage.getItem('policy_test_cases');
    if (saved) {
      try {
        setTestCases(JSON.parse(saved));
      } catch (err) {
        console.error('Failed to parse saved test cases');
      }
    }
  };

  const saveTestCases = (cases: TestCase[]) => {
    localStorage.setItem('policy_test_cases', JSON.stringify(cases));
    setTestCases(cases);
  };

  const handleRunTest = async () => {
    try {
      setLoading(true);
      setError(null);

      // Validate JSON input
      let parsedInput: object;
      try {
        parsedInput = JSON.parse(inputJson);
      } catch (err) {
        setError('Invalid JSON input');
        return;
      }

      const request: PolicyTestRequest = {
        policy_id: useCustomCode ? undefined : selectedPolicy || undefined,
        rego_code: useCustomCode ? customRegoCode : '',
        input_json: parsedInput
      };

      const result = await testPolicy(request);
      setTestResult(result);

      if (result.errors && result.errors.length > 0) {
        setError(`Policy evaluation errors: ${result.errors.join(', ')}`);
      } else {
        setSuccess('Policy test executed successfully');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to run test');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveTestCase = () => {
    const testCase: TestCase = {
      id: Date.now().toString(),
      name: `Test Case ${testCases.length + 1}`,
      description: 'Saved test case',
      input_json: JSON.parse(inputJson)
    };
    saveTestCases([...testCases, testCase]);
    setSuccess('Test case saved');
  };

  const handleLoadTestCase = (testCase: TestCase) => {
    setInputJson(JSON.stringify(testCase.input_json, null, 2));
  };

  const handleDeleteTestCase = (id: string) => {
    saveTestCases(testCases.filter((tc) => tc.id !== id));
    setSuccess('Test case deleted');
  };

  const handleBulkTest = async () => {
    if (testCases.length === 0) {
      setError('No test cases to run');
      return;
    }

    setLoading(true);
    const results: Array<{
      test_case: string;
      decision: string;
      time_ms: number;
    }> = [];

    for (const testCase of testCases) {
      try {
        const request: PolicyTestRequest = {
          policy_id: useCustomCode ? undefined : selectedPolicy || undefined,
          rego_code: useCustomCode ? customRegoCode : '',
          input_json: testCase.input_json
        };

        const result = await testPolicy(request);
        results.push({
          test_case: testCase.name,
          decision: result.decision,
          time_ms: result.evaluation_time_ms
        });
      } catch (err: any) {
        results.push({
          test_case: testCase.name,
          decision: 'error',
          time_ms: 0
        });
      }
    }

    setBulkResults(results);
    setLoading(false);
    setSuccess(`Completed ${results.length} tests`);
  };

  const handleImportCSV = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      try {
        const lines = text.split('\n').filter((line) => line.trim());
        const headers = lines[0].split(',');

        const imported: TestCase[] = lines.slice(1).map((line, idx) => {
          const values = line.split(',');
          const inputObj: any = {};
          headers.forEach((header, i) => {
            if (values[i]) {
              inputObj[header.trim()] = values[i].trim();
            }
          });

          return {
            id: Date.now().toString() + idx,
            name: `Imported ${idx + 1}`,
            description: 'CSV import',
            input_json: inputObj
          };
        });

        saveTestCases([...testCases, ...imported]);
        setSuccess(`Imported ${imported.length} test cases`);
        setCsvDialogOpen(false);
      } catch (err) {
        setError('Failed to parse CSV file');
      }
    };
    reader.readAsText(file);
  };

  const copyTraceToClipboard = () => {
    if (testResult?.evaluation_trace) {
      navigator.clipboard.writeText(testResult.evaluation_trace.join('\n'));
      setSuccess('Trace copied to clipboard');
    }
  };

  return (
    <LicenseGate
      featureName="Zero-Trust Security"
      hasAccess={hasEnterpriseAccess}
      isLoading={false}
    >
      <Box sx={{ p: 3 }}>
        <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h4">Policy Testing Interface</Typography>
          <Box>
            <Button
              variant="outlined"
              startIcon={<UploadIcon />}
              onClick={() => setCsvDialogOpen(true)}
              sx={{ mr: 1 }}
            >
              Import CSV
            </Button>
            <Button
              variant="contained"
              startIcon={<RunIcon />}
              onClick={handleBulkTest}
              disabled={loading || testCases.length === 0}
            >
              Bulk Test
            </Button>
          </Box>
        </Box>

        <Grid container spacing={3}>
          {/* Test Configuration */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Test Configuration
              </Typography>
              <Divider sx={{ mb: 2 }} />

              <FormControl fullWidth sx={{ mb: 2 }}>
                <InputLabel>Select Policy</InputLabel>
                <Select
                  value={selectedPolicy || ''}
                  onChange={(e) => setSelectedPolicy(Number(e.target.value))}
                  disabled={useCustomCode}
                >
                  {policies.map((policy) => (
                    <MenuItem key={policy.id} value={policy.id}>
                      {policy.name} (v{policy.version})
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              <Typography variant="subtitle2" gutterBottom>
                Input JSON
              </Typography>
              <Box sx={{ mb: 2, border: '1px solid #ddd', borderRadius: 1 }}>
                <Editor
                  height="250px"
                  language="json"
                  value={inputJson}
                  onChange={(value) => setInputJson(value || '')}
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    wordWrap: 'on',
                    automaticLayout: true,
                    scrollBeyondLastLine: false
                  }}
                  theme="vs-light"
                />
              </Box>

              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button
                  variant="contained"
                  startIcon={<RunIcon />}
                  onClick={handleRunTest}
                  disabled={loading || (!selectedPolicy && !useCustomCode)}
                  fullWidth
                >
                  Run Test
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<SaveIcon />}
                  onClick={handleSaveTestCase}
                >
                  Save
                </Button>
              </Box>
            </Paper>

            {/* Saved Test Cases */}
            <Paper sx={{ p: 3, mt: 3 }}>
              <Typography variant="h6" gutterBottom>
                Saved Test Cases ({testCases.length})
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <List sx={{ maxHeight: 300, overflow: 'auto' }}>
                {testCases.map((testCase) => (
                  <ListItem
                    key={testCase.id}
                    secondaryAction={
                      <IconButton
                        edge="end"
                        onClick={() => handleDeleteTestCase(testCase.id)}
                      >
                        <DeleteIcon />
                      </IconButton>
                    }
                  >
                    <ListItemText
                      primary={testCase.name}
                      secondary={testCase.description}
                      onClick={() => handleLoadTestCase(testCase)}
                      sx={{ cursor: 'pointer' }}
                    />
                  </ListItem>
                ))}
              </List>
            </Paper>
          </Grid>

          {/* Test Results */}
          <Grid item xs={12} md={6}>
            {testResult && (
              <Paper sx={{ p: 3 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="h6">Test Result</Typography>
                  <Chip
                    label={testResult.decision.toUpperCase()}
                    color={testResult.decision === 'allow' ? 'success' : 'error'}
                  />
                </Box>
                <Divider sx={{ mb: 2 }} />

                <Card sx={{ mb: 2, bgcolor: 'grey.50' }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                      <PerformanceIcon sx={{ mr: 1 }} />
                      <Typography variant="subtitle2">
                        Performance Metrics
                      </Typography>
                    </Box>
                    <Typography variant="body2">
                      Evaluation Time: <strong>{testResult.evaluation_time_ms.toFixed(2)} ms</strong>
                    </Typography>
                  </CardContent>
                </Card>

                {testResult.result && (
                  <Card sx={{ mb: 2 }}>
                    <CardContent>
                      <Typography variant="subtitle2" gutterBottom>
                        Result Object
                      </Typography>
                      <Box
                        sx={{
                          p: 1,
                          bgcolor: 'grey.100',
                          borderRadius: 1,
                          fontFamily: 'monospace',
                          fontSize: '0.875rem',
                          maxHeight: 200,
                          overflow: 'auto'
                        }}
                      >
                        <pre>{JSON.stringify(testResult.result, null, 2)}</pre>
                      </Box>
                    </CardContent>
                  </Card>
                )}

                {testResult.evaluation_trace && testResult.evaluation_trace.length > 0 && (
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                        <Typography variant="subtitle2">
                          Evaluation Trace ({testResult.evaluation_trace.length} steps)
                        </Typography>
                        <Tooltip title="Copy to clipboard">
                          <IconButton size="small" onClick={copyTraceToClipboard}>
                            <CopyIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                      <Box
                        sx={{
                          p: 1,
                          bgcolor: 'grey.100',
                          borderRadius: 1,
                          fontFamily: 'monospace',
                          fontSize: '0.75rem',
                          maxHeight: 300,
                          overflow: 'auto'
                        }}
                      >
                        {testResult.evaluation_trace.map((trace, idx) => (
                          <div key={idx}>{trace}</div>
                        ))}
                      </Box>
                    </CardContent>
                  </Card>
                )}
              </Paper>
            )}

            {/* Bulk Test Results */}
            {bulkResults.length > 0 && (
              <Paper sx={{ p: 3, mt: 3 }}>
                <Typography variant="h6" gutterBottom>
                  Bulk Test Results
                </Typography>
                <Divider sx={{ mb: 2 }} />
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Test Case</TableCell>
                        <TableCell>Decision</TableCell>
                        <TableCell align="right">Time (ms)</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {bulkResults.map((result, idx) => (
                        <TableRow key={idx}>
                          <TableCell>{result.test_case}</TableCell>
                          <TableCell>
                            <Chip
                              label={result.decision}
                              color={
                                result.decision === 'allow'
                                  ? 'success'
                                  : result.decision === 'deny'
                                  ? 'error'
                                  : 'default'
                              }
                              size="small"
                            />
                          </TableCell>
                          <TableCell align="right">
                            {result.time_ms.toFixed(2)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
                <Box sx={{ mt: 2 }}>
                  <Typography variant="caption" color="textSecondary">
                    Average Time:{' '}
                    {(
                      bulkResults.reduce((sum, r) => sum + r.time_ms, 0) /
                      bulkResults.length
                    ).toFixed(2)}{' '}
                    ms
                  </Typography>
                </Box>
              </Paper>
            )}
          </Grid>
        </Grid>

        {/* CSV Import Dialog */}
        <Dialog open={csvDialogOpen} onClose={() => setCsvDialogOpen(false)}>
          <DialogTitle>Import Test Cases from CSV</DialogTitle>
          <DialogContent>
            <Typography variant="body2" paragraph>
              Upload a CSV file with columns representing input fields.
              Each row will become a test case.
            </Typography>
            <input
              type="file"
              accept=".csv"
              onChange={handleImportCSV}
              style={{ marginTop: 16 }}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setCsvDialogOpen(false)}>Cancel</Button>
          </DialogActions>
        </Dialog>

        {/* Snackbar Messages */}
        <Snackbar
          open={!!error}
          autoHideDuration={6000}
          onClose={() => setError(null)}
        >
          <Alert severity="error" onClose={() => setError(null)}>
            {error}
          </Alert>
        </Snackbar>
        <Snackbar
          open={!!success}
          autoHideDuration={4000}
          onClose={() => setSuccess(null)}
        >
          <Alert severity="success" onClose={() => setSuccess(null)}>
            {success}
          </Alert>
        </Snackbar>
      </Box>
    </LicenseGate>
  );
};

export default PolicyTester;
