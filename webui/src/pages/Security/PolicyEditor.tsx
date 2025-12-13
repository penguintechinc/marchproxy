/**
 * OPA Policy Editor
 *
 * Monaco Editor-based Rego policy editor with syntax highlighting,
 * validation, version control, and policy templates.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  Grid,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  ListItemButton,
  Chip,
  Alert,
  Snackbar,
  IconButton,
  Tooltip,
  Divider,
  Card,
  CardContent,
  Switch,
  FormControlLabel
} from '@mui/material';
import {
  Save as SaveIcon,
  History as HistoryIcon,
  Code as CodeIcon,
  PlayArrow as TestIcon,
  Delete as DeleteIcon,
  Add as AddIcon,
  RestorePage as RestoreIcon
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';
import {
  OPAPolicy,
  PolicyVersion,
  PolicyTemplate,
  getPolicies,
  getPolicy,
  savePolicy,
  deletePolicy,
  getPolicyVersions,
  validatePolicy,
  getPolicyTemplates
} from '../../services/securityApi';
import LicenseGate from '../../components/Common/LicenseGate';
import { useLicense } from '../../hooks/useLicense';

const PolicyEditor: React.FC = () => {
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('zero_trust');
  const [policies, setPolicies] = useState<OPAPolicy[]>([]);
  const [currentPolicy, setCurrentPolicy] = useState<Partial<OPAPolicy>>({
    name: '',
    description: '',
    rego_code: '',
    is_active: true
  });
  const [editorContent, setEditorContent] = useState<string>('');
  const [versions, setVersions] = useState<PolicyVersion[]>([]);
  const [templates, setTemplates] = useState<PolicyTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [versionDialogOpen, setVersionDialogOpen] = useState(false);
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);

  useEffect(() => {
    loadPolicies();
    loadTemplates();
  }, []);

  const loadPolicies = async () => {
    try {
      setLoading(true);
      const data = await getPolicies();
      setPolicies(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load policies');
    } finally {
      setLoading(false);
    }
  };

  const loadTemplates = async () => {
    try {
      const data = await getPolicyTemplates();
      setTemplates(data);
    } catch (err: any) {
      console.error('Failed to load templates:', err);
    }
  };

  const handlePolicySelect = async (policy: OPAPolicy) => {
    try {
      setLoading(true);
      const fullPolicy = await getPolicy(policy.id);
      setCurrentPolicy(fullPolicy);
      setEditorContent(fullPolicy.rego_code);
      setValidationErrors([]);
    } catch (err: any) {
      setError(err.message || 'Failed to load policy');
    } finally {
      setLoading(false);
    }
  };

  const handleEditorChange = (value: string | undefined) => {
    if (value !== undefined) {
      setEditorContent(value);
      setCurrentPolicy({ ...currentPolicy, rego_code: value });
    }
  };

  const handleValidate = async () => {
    try {
      setLoading(true);
      const result = await validatePolicy(editorContent);
      if (result.valid) {
        setSuccess('Policy is valid');
        setValidationErrors([]);
      } else {
        setValidationErrors(result.errors);
      }
    } catch (err: any) {
      setError(err.message || 'Validation failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setLoading(true);

      // Validate before saving
      const validationResult = await validatePolicy(editorContent);
      if (!validationResult.valid) {
        setValidationErrors(validationResult.errors);
        setError('Policy has validation errors. Please fix them before saving.');
        return;
      }

      const savedPolicy = await savePolicy({
        ...currentPolicy,
        rego_code: editorContent
      });

      setCurrentPolicy(savedPolicy);
      setSuccess(
        currentPolicy.id
          ? 'Policy updated successfully'
          : 'Policy created successfully'
      );
      loadPolicies();
    } catch (err: any) {
      setError(err.message || 'Failed to save policy');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!currentPolicy.id) return;

    try {
      setLoading(true);
      await deletePolicy(currentPolicy.id);
      setSuccess('Policy deleted successfully');
      setDeleteDialogOpen(false);
      setCurrentPolicy({
        name: '',
        description: '',
        rego_code: '',
        is_active: true
      });
      setEditorContent('');
      loadPolicies();
    } catch (err: any) {
      setError(err.message || 'Failed to delete policy');
    } finally {
      setLoading(false);
    }
  };

  const handleNewPolicy = () => {
    setCurrentPolicy({
      name: '',
      description: '',
      rego_code: '# New OPA Policy\npackage marchproxy\n\ndefault allow = false\n\nallow {\n    # Your policy logic here\n}',
      is_active: true
    });
    setEditorContent('# New OPA Policy\npackage marchproxy\n\ndefault allow = false\n\nallow {\n    # Your policy logic here\n}');
    setValidationErrors([]);
  };

  const handleViewVersions = async () => {
    if (!currentPolicy.id) return;

    try {
      setLoading(true);
      const data = await getPolicyVersions(currentPolicy.id);
      setVersions(data);
      setVersionDialogOpen(true);
    } catch (err: any) {
      setError(err.message || 'Failed to load versions');
    } finally {
      setLoading(false);
    }
  };

  const handleRestoreVersion = (version: PolicyVersion) => {
    setEditorContent(version.rego_code);
    setCurrentPolicy({ ...currentPolicy, rego_code: version.rego_code });
    setVersionDialogOpen(false);
    setSuccess(`Restored to version ${version.version}`);
  };

  const handleApplyTemplate = (template: PolicyTemplate) => {
    setEditorContent(template.rego_code);
    setCurrentPolicy({
      ...currentPolicy,
      name: template.name,
      description: template.description,
      rego_code: template.rego_code
    });
    setTemplateDialogOpen(false);
    setSuccess(`Applied template: ${template.name}`);
  };

  return (
    <LicenseGate
      featureName="Zero-Trust Security"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Box sx={{ p: 3 }}>
        <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h4">OPA Policy Editor</Typography>
          <Box>
            <Button
              variant="outlined"
              startIcon={<CodeIcon />}
              onClick={() => setTemplateDialogOpen(true)}
              sx={{ mr: 1 }}
            >
              Templates
            </Button>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleNewPolicy}
            >
              New Policy
            </Button>
          </Box>
        </Box>

        <Grid container spacing={3}>
          {/* Policy List */}
          <Grid item xs={12} md={3}>
            <Paper sx={{ p: 2, height: 'calc(100vh - 200px)', overflow: 'auto' }}>
              <Typography variant="h6" gutterBottom>
                Policies
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <List>
                {policies.map((policy) => (
                  <ListItemButton
                    key={policy.id}
                    selected={currentPolicy.id === policy.id}
                    onClick={() => handlePolicySelect(policy)}
                  >
                    <ListItemText
                      primary={policy.name}
                      secondary={
                        <Box>
                          <Typography variant="caption" display="block">
                            v{policy.version}
                          </Typography>
                          <Chip
                            label={policy.is_active ? 'Active' : 'Inactive'}
                            size="small"
                            color={policy.is_active ? 'success' : 'default'}
                            sx={{ mt: 0.5 }}
                          />
                        </Box>
                      }
                    />
                  </ListItemButton>
                ))}
              </List>
            </Paper>
          </Grid>

          {/* Editor Section */}
          <Grid item xs={12} md={9}>
            <Paper sx={{ p: 3 }}>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Policy Name"
                    value={currentPolicy.name || ''}
                    onChange={(e) =>
                      setCurrentPolicy({ ...currentPolicy, name: e.target.value })
                    }
                    required
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={currentPolicy.is_active || false}
                        onChange={(e) =>
                          setCurrentPolicy({
                            ...currentPolicy,
                            is_active: e.target.checked
                          })
                        }
                      />
                    }
                    label="Active"
                  />
                </Grid>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Description"
                    value={currentPolicy.description || ''}
                    onChange={(e) =>
                      setCurrentPolicy({
                        ...currentPolicy,
                        description: e.target.value
                      })
                    }
                    multiline
                    rows={2}
                  />
                </Grid>
              </Grid>

              {validationErrors.length > 0 && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  <Typography variant="subtitle2">Validation Errors:</Typography>
                  <ul style={{ margin: 0, paddingLeft: 20 }}>
                    {validationErrors.map((err, idx) => (
                      <li key={idx}>{err}</li>
                    ))}
                  </ul>
                </Alert>
              )}

              <Box sx={{ mb: 2, border: '1px solid #ddd', borderRadius: 1 }}>
                <Editor
                  height="500px"
                  language="plaintext"
                  value={editorContent}
                  onChange={handleEditorChange}
                  options={{
                    minimap: { enabled: true },
                    fontSize: 14,
                    wordWrap: 'on',
                    automaticLayout: true,
                    scrollBeyondLastLine: false,
                    renderWhitespace: 'selection'
                  }}
                  theme="vs-dark"
                />
              </Box>

              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                <Button
                  variant="contained"
                  startIcon={<SaveIcon />}
                  onClick={handleSave}
                  disabled={loading || !currentPolicy.name}
                >
                  Save Policy
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<TestIcon />}
                  onClick={handleValidate}
                  disabled={loading}
                >
                  Validate
                </Button>
                {currentPolicy.id && (
                  <>
                    <Button
                      variant="outlined"
                      startIcon={<HistoryIcon />}
                      onClick={handleViewVersions}
                      disabled={loading}
                    >
                      Version History
                    </Button>
                    <Button
                      variant="outlined"
                      color="error"
                      startIcon={<DeleteIcon />}
                      onClick={() => setDeleteDialogOpen(true)}
                      disabled={loading}
                    >
                      Delete
                    </Button>
                  </>
                )}
              </Box>

              {currentPolicy.id && (
                <Box sx={{ mt: 2 }}>
                  <Typography variant="caption" color="textSecondary">
                    Version: {currentPolicy.version} | Last Updated:{' '}
                    {new Date(currentPolicy.updated_at || '').toLocaleString()}
                  </Typography>
                </Box>
              )}
            </Paper>
          </Grid>
        </Grid>

        {/* Version History Dialog */}
        <Dialog
          open={versionDialogOpen}
          onClose={() => setVersionDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Version History</DialogTitle>
          <DialogContent>
            <List>
              {versions.map((version) => (
                <Card key={version.id} sx={{ mb: 2 }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Box>
                        <Typography variant="h6">Version {version.version}</Typography>
                        <Typography variant="caption" color="textSecondary">
                          Created: {new Date(version.created_at).toLocaleString()}
                          <br />
                          By: {version.created_by}
                        </Typography>
                      </Box>
                      <Button
                        variant="outlined"
                        startIcon={<RestoreIcon />}
                        onClick={() => handleRestoreVersion(version)}
                      >
                        Restore
                      </Button>
                    </Box>
                    {version.diff && (
                      <Box sx={{ mt: 2, p: 1, bgcolor: 'grey.100', borderRadius: 1 }}>
                        <Typography variant="caption" component="pre">
                          {version.diff}
                        </Typography>
                      </Box>
                    )}
                  </CardContent>
                </Card>
              ))}
            </List>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setVersionDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Template Library Dialog */}
        <Dialog
          open={templateDialogOpen}
          onClose={() => setTemplateDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Policy Templates</DialogTitle>
          <DialogContent>
            <Grid container spacing={2}>
              {templates.map((template) => (
                <Grid item xs={12} key={template.id}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6">{template.name}</Typography>
                      <Chip label={template.category} size="small" sx={{ mb: 1 }} />
                      <Typography variant="body2" color="textSecondary" paragraph>
                        {template.description}
                      </Typography>
                      <Button
                        variant="outlined"
                        onClick={() => handleApplyTemplate(template)}
                      >
                        Use Template
                      </Button>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setTemplateDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Delete Confirmation Dialog */}
        <Dialog
          open={deleteDialogOpen}
          onClose={() => setDeleteDialogOpen(false)}
        >
          <DialogTitle>Delete Policy</DialogTitle>
          <DialogContent>
            <Typography>
              Are you sure you want to delete the policy "{currentPolicy.name}"?
              This action cannot be undone.
            </Typography>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleDelete} color="error" variant="contained">
              Delete
            </Button>
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

export default PolicyEditor;
