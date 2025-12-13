import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Alert,
  Stack,
  Typography,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
} from '@mui/material';
import {
  Save,
  Delete,
  Add,
  Code,
  PlayArrow,
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';
import { apiClient } from '../../services/api';

interface Policy {
  name: string;
  type: string;
  content: string;
  description: string;
}

const defaultPolicies: Record<string, string> = {
  rbac: `# RBAC Policy Template
package marchproxy.rbac

import rego.v1

default allow := false

allow if {
    input.user != ""
    user_roles := data.users[input.user].roles
    some role in user_roles
    role_permissions := data.roles[role].permissions
    required_permission := concat(":", [input.action, input.resource])
    required_permission in role_permissions
}`,
  rate_limit: `# Rate Limiting Policy Template
package marchproxy.rate_limit

import rego.v1

default_rate_limit := {
    "requests_per_second": 100,
    "requests_per_minute": 1000,
    "burst_size": 50,
}

rate_limit contains result if {
    input.service != ""
    service_config := data.rate_limits[input.service]
    service_config != null
    result := service_config
}`,
  compliance: `# Compliance Policy Template
package marchproxy.compliance

import rego.v1

soc2_compliant if {
    authentication_required
    audit_trail_intact
    encryption_enabled
}

authentication_required if {
    input.user != ""
}`,
};

const PolicyEditor: React.FC = () => {
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [selectedPolicy, setSelectedPolicy] = useState<string>('');
  const [policyContent, setPolicyContent] = useState<string>('');
  const [policyType, setPolicyType] = useState<string>('rbac');
  const [policyName, setPolicyName] = useState<string>('');
  const [policyDescription, setPolicyDescription] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [newPolicyDialog, setNewPolicyDialog] = useState(false);

  useEffect(() => {
    fetchPolicies();
  }, []);

  const fetchPolicies = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get('/api/v1/zerotrust/policies');
      setPolicies(response.data.policies || []);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch policies');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectPolicy = (policyName: string) => {
    const policy = policies.find(p => p.name === policyName);
    if (policy) {
      setSelectedPolicy(policyName);
      setPolicyContent(policy.content);
      setPolicyType(policy.type);
      setPolicyName(policy.name);
      setPolicyDescription(policy.description);
    }
  };

  const handleNewPolicy = () => {
    setNewPolicyDialog(true);
    setPolicyName('');
    setPolicyDescription('');
    setPolicyType('rbac');
    setPolicyContent(defaultPolicies.rbac);
  };

  const handleSavePolicy = async () => {
    if (!policyName || !policyContent) {
      setError('Policy name and content are required');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setSuccess(null);

      await apiClient.post('/api/v1/zerotrust/policies', {
        name: policyName,
        type: policyType,
        content: policyContent,
        description: policyDescription,
      });

      setSuccess('Policy saved successfully');
      setNewPolicyDialog(false);
      await fetchPolicies();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to save policy');
    } finally {
      setLoading(false);
    }
  };

  const handleDeletePolicy = async (policyName: string) => {
    if (!confirm(`Are you sure you want to delete policy "${policyName}"?`)) {
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setSuccess(null);

      await apiClient.delete(`/api/v1/zerotrust/policies/${policyName}`);

      setSuccess('Policy deleted successfully');
      setSelectedPolicy('');
      setPolicyContent('');
      await fetchPolicies();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to delete policy');
    } finally {
      setLoading(false);
    }
  };

  const handleValidatePolicy = async () => {
    try {
      setLoading(true);
      setError(null);
      setSuccess(null);

      const response = await apiClient.post('/api/v1/zerotrust/policies/validate', {
        content: policyContent,
      });

      if (response.data.valid) {
        setSuccess('Policy is valid');
      } else {
        setError(`Policy validation failed: ${response.data.error}`);
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to validate policy');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box>
      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <FormControl sx={{ minWidth: 300 }}>
          <InputLabel>Select Policy</InputLabel>
          <Select
            value={selectedPolicy}
            onChange={(e) => handleSelectPolicy(e.target.value)}
            label="Select Policy"
          >
            {policies.map((policy) => (
              <MenuItem key={policy.name} value={policy.name}>
                {policy.name} ({policy.type})
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <Button
          variant="contained"
          startIcon={<Add />}
          onClick={handleNewPolicy}
        >
          New Policy
        </Button>

        <Box sx={{ flexGrow: 1 }} />

        {selectedPolicy && (
          <>
            <Tooltip title="Validate Policy">
              <IconButton color="primary" onClick={handleValidatePolicy}>
                <PlayArrow />
              </IconButton>
            </Tooltip>

            <Tooltip title="Save Policy">
              <IconButton color="primary" onClick={handleSavePolicy}>
                <Save />
              </IconButton>
            </Tooltip>

            <Tooltip title="Delete Policy">
              <IconButton color="error" onClick={() => handleDeletePolicy(selectedPolicy)}>
                <Delete />
              </IconButton>
            </Tooltip>
          </>
        )}
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {success && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      )}

      {selectedPolicy && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="body2" color="text.secondary">
            {policyDescription}
          </Typography>
        </Box>
      )}

      <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1 }}>
        <Editor
          height="500px"
          language="rego"
          theme="vs-dark"
          value={policyContent}
          onChange={(value) => setPolicyContent(value || '')}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            formatOnPaste: true,
            formatOnType: true,
          }}
        />
      </Box>

      {/* New Policy Dialog */}
      <Dialog open={newPolicyDialog} onClose={() => setNewPolicyDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>Create New Policy</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <TextField
              label="Policy Name"
              value={policyName}
              onChange={(e) => setPolicyName(e.target.value)}
              fullWidth
              required
            />

            <FormControl fullWidth>
              <InputLabel>Policy Type</InputLabel>
              <Select
                value={policyType}
                onChange={(e) => {
                  setPolicyType(e.target.value);
                  setPolicyContent(defaultPolicies[e.target.value] || '');
                }}
                label="Policy Type"
              >
                <MenuItem value="rbac">RBAC</MenuItem>
                <MenuItem value="rate_limit">Rate Limiting</MenuItem>
                <MenuItem value="compliance">Compliance</MenuItem>
                <MenuItem value="custom">Custom</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Description"
              value={policyDescription}
              onChange={(e) => setPolicyDescription(e.target.value)}
              fullWidth
              multiline
              rows={2}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setNewPolicyDialog(false)}>Cancel</Button>
          <Button onClick={handleSavePolicy} variant="contained" disabled={loading}>
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default PolicyEditor;
