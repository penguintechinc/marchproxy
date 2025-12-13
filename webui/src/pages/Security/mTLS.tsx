/**
 * mTLS Certificate Management
 *
 * Manages client certificates, CA certificates, CRL, validation,
 * auto-rotation, and expiry alerts for mutual TLS authentication.
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
  Chip,
  Alert,
  Snackbar,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Switch,
  FormControlLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Divider,
  LinearProgress
} from '@mui/material';
import {
  Upload as UploadIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
  CheckCircle as ValidIcon,
  Cancel as InvalidIcon,
  Verified as VerifiedIcon,
  Block as RevokeIcon,
  Info as InfoIcon
} from '@mui/icons-material';
import {
  Certificate,
  CertificateUpload,
  CertificateValidation,
  CRLEntry,
  getCertificates,
  getCertificate,
  uploadCertificate,
  deleteCertificate,
  revokeCertificate,
  validateCertificate,
  getCRL,
  updateCertificateRotation,
  getExpiringCertificates
} from '../../services/securityApi';
import LicenseGate from '../../components/Common/LicenseGate';

const mTLS: React.FC = () => {
  const hasEnterpriseAccess = true; // TODO: Get from license check
  const [certificates, setCertificates] = useState<Certificate[]>([]);
  const [expiringCerts, setExpiringCerts] = useState<Certificate[]>([]);
  const [crl, setCrl] = useState<CRLEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [selectedCert, setSelectedCert] = useState<Certificate | null>(null);
  const [validation, setValidation] = useState<CertificateValidation | null>(null);
  const [uploadForm, setUploadForm] = useState<CertificateUpload>({
    name: '',
    type: 'client',
    certificate_pem: '',
    private_key_pem: '',
    auto_rotation_enabled: false
  });
  const [revokeReason, setRevokeReason] = useState('');

  useEffect(() => {
    loadCertificates();
    loadExpiringCertificates();
    loadCRL();
  }, []);

  const loadCertificates = async () => {
    try {
      setLoading(true);
      const data = await getCertificates();
      setCertificates(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load certificates');
    } finally {
      setLoading(false);
    }
  };

  const loadExpiringCertificates = async () => {
    try {
      const data = await getExpiringCertificates(30);
      setExpiringCerts(data);
    } catch (err: any) {
      console.error('Failed to load expiring certificates:', err);
    }
  };

  const loadCRL = async () => {
    try {
      const data = await getCRL();
      setCrl(data);
    } catch (err: any) {
      console.error('Failed to load CRL:', err);
    }
  };

  const handleUploadCertificate = async () => {
    try {
      setLoading(true);

      if (!uploadForm.name || !uploadForm.certificate_pem) {
        setError('Name and certificate PEM are required');
        return;
      }

      await uploadCertificate(uploadForm);
      setSuccess('Certificate uploaded successfully');
      setUploadDialogOpen(false);
      setUploadForm({
        name: '',
        type: 'client',
        certificate_pem: '',
        private_key_pem: '',
        auto_rotation_enabled: false
      });
      loadCertificates();
      loadExpiringCertificates();
    } catch (err: any) {
      setError(err.message || 'Failed to upload certificate');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteCertificate = async () => {
    if (!selectedCert) return;

    try {
      setLoading(true);
      await deleteCertificate(selectedCert.id);
      setSuccess('Certificate deleted successfully');
      setDeleteDialogOpen(false);
      setSelectedCert(null);
      loadCertificates();
    } catch (err: any) {
      setError(err.message || 'Failed to delete certificate');
    } finally {
      setLoading(false);
    }
  };

  const handleRevokeCertificate = async () => {
    if (!selectedCert) return;

    try {
      setLoading(true);
      await revokeCertificate(selectedCert.id, revokeReason);
      setSuccess('Certificate revoked successfully');
      setRevokeDialogOpen(false);
      setSelectedCert(null);
      setRevokeReason('');
      loadCertificates();
      loadCRL();
    } catch (err: any) {
      setError(err.message || 'Failed to revoke certificate');
    } finally {
      setLoading(false);
    }
  };

  const handleValidateCertificate = async (cert: Certificate) => {
    try {
      setLoading(true);
      const result = await validateCertificate(cert.id);
      setValidation(result);
      setSelectedCert(cert);
      setDetailsDialogOpen(true);
    } catch (err: any) {
      setError(err.message || 'Failed to validate certificate');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleRotation = async (cert: Certificate) => {
    try {
      setLoading(true);
      await updateCertificateRotation(cert.id, !cert.auto_rotation_enabled);
      setSuccess(
        `Auto-rotation ${!cert.auto_rotation_enabled ? 'enabled' : 'disabled'}`
      );
      loadCertificates();
    } catch (err: any) {
      setError(err.message || 'Failed to update rotation setting');
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'valid':
        return 'success';
      case 'expired':
        return 'error';
      case 'revoked':
        return 'warning';
      case 'pending':
        return 'info';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'valid':
        return <ValidIcon color="success" />;
      case 'expired':
      case 'revoked':
        return <InvalidIcon color="error" />;
      case 'pending':
        return <InfoIcon color="info" />;
      default:
        return null;
    }
  };

  const getDaysUntilExpiry = (notAfter: string) => {
    const diff = new Date(notAfter).getTime() - Date.now();
    return Math.ceil(diff / (1000 * 60 * 60 * 24));
  };

  const handleFileUpload = (
    event: React.ChangeEvent<HTMLInputElement>,
    field: 'certificate_pem' | 'private_key_pem'
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      setUploadForm({ ...uploadForm, [field]: content });
    };
    reader.readAsText(file);
  };

  return (
    <LicenseGate
      featureName="Zero-Trust Security"
      hasAccess={hasEnterpriseAccess}
      isLoading={false}
    >
      <Box sx={{ p: 3 }}>
        <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h4">mTLS Certificate Management</Typography>
          <Box>
            <Button
              variant="contained"
              startIcon={<UploadIcon />}
              onClick={() => setUploadDialogOpen(true)}
              disabled={loading}
              sx={{ mr: 1 }}
            >
              Upload Certificate
            </Button>
            <IconButton onClick={loadCertificates} disabled={loading}>
              <RefreshIcon />
            </IconButton>
          </Box>
        </Box>

        {/* Expiring Certificates Alert */}
        {expiringCerts.length > 0 && (
          <Alert severity="warning" sx={{ mb: 3 }}>
            <Typography variant="subtitle2" gutterBottom>
              {expiringCerts.length} Certificate(s) Expiring Soon
            </Typography>
            <Typography variant="body2">
              The following certificates will expire within 30 days:{' '}
              {expiringCerts.map((c) => c.name).join(', ')}
            </Typography>
          </Alert>
        )}

        {/* Statistics */}
        <Grid container spacing={3} sx={{ mb: 3 }}>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Total Certificates
                </Typography>
                <Typography variant="h3">{certificates.length}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Valid
                </Typography>
                <Typography variant="h3" color="success.main">
                  {certificates.filter((c) => c.status === 'valid').length}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Expiring Soon
                </Typography>
                <Typography variant="h3" color="warning.main">
                  {expiringCerts.length}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Revoked
                </Typography>
                <Typography variant="h3" color="error.main">
                  {crl.length}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Certificates Table */}
        <Paper sx={{ mb: 3 }}>
          <Box sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Certificates
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Name</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Subject</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Expires</TableCell>
                    <TableCell>Auto-Rotation</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {certificates.map((cert) => {
                    const daysLeft = getDaysUntilExpiry(cert.not_after);
                    return (
                      <TableRow key={cert.id}>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            {getStatusIcon(cert.status)}
                            <Typography sx={{ ml: 1 }}>{cert.name}</Typography>
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={cert.type.toUpperCase()}
                            size="small"
                            variant="outlined"
                          />
                        </TableCell>
                        <TableCell>
                          <Typography variant="caption">{cert.subject}</Typography>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={cert.status}
                            color={getStatusColor(cert.status) as any}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Box>
                            <Typography variant="body2">
                              {new Date(cert.not_after).toLocaleDateString()}
                            </Typography>
                            {cert.status === 'valid' && (
                              <Typography
                                variant="caption"
                                color={daysLeft <= 30 ? 'warning.main' : 'textSecondary'}
                              >
                                {daysLeft} days
                              </Typography>
                            )}
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Switch
                            checked={cert.auto_rotation_enabled}
                            onChange={() => handleToggleRotation(cert)}
                            size="small"
                          />
                        </TableCell>
                        <TableCell align="right">
                          <Tooltip title="Validate">
                            <IconButton
                              size="small"
                              onClick={() => handleValidateCertificate(cert)}
                            >
                              <VerifiedIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          {cert.status === 'valid' && (
                            <Tooltip title="Revoke">
                              <IconButton
                                size="small"
                                onClick={() => {
                                  setSelectedCert(cert);
                                  setRevokeDialogOpen(true);
                                }}
                              >
                                <RevokeIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          )}
                          <Tooltip title="Delete">
                            <IconButton
                              size="small"
                              onClick={() => {
                                setSelectedCert(cert);
                                setDeleteDialogOpen(true);
                              }}
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        </Paper>

        {/* Certificate Revocation List (CRL) */}
        <Paper>
          <Box sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Certificate Revocation List (CRL)
            </Typography>
            <Divider sx={{ mb: 2 }} />
            {crl.length === 0 ? (
              <Typography variant="body2" color="textSecondary">
                No revoked certificates
              </Typography>
            ) : (
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Serial Number</TableCell>
                      <TableCell>Revocation Date</TableCell>
                      <TableCell>Reason</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {crl.map((entry, idx) => (
                      <TableRow key={idx}>
                        <TableCell>
                          <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                            {entry.serial_number}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          {new Date(entry.revocation_date).toLocaleString()}
                        </TableCell>
                        <TableCell>{entry.reason}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Box>
        </Paper>

        {/* Upload Certificate Dialog */}
        <Dialog
          open={uploadDialogOpen}
          onClose={() => setUploadDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Upload Certificate</DialogTitle>
          <DialogContent>
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Certificate Name"
                  value={uploadForm.name}
                  onChange={(e) =>
                    setUploadForm({ ...uploadForm, name: e.target.value })
                  }
                  required
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <FormControl fullWidth>
                  <InputLabel>Certificate Type</InputLabel>
                  <Select
                    value={uploadForm.type}
                    onChange={(e) =>
                      setUploadForm({
                        ...uploadForm,
                        type: e.target.value as 'client' | 'ca' | 'server'
                      })
                    }
                  >
                    <MenuItem value="client">Client Certificate</MenuItem>
                    <MenuItem value="ca">CA Certificate</MenuItem>
                    <MenuItem value="server">Server Certificate</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} md={6}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={uploadForm.auto_rotation_enabled}
                      onChange={(e) =>
                        setUploadForm({
                          ...uploadForm,
                          auto_rotation_enabled: e.target.checked
                        })
                      }
                    />
                  }
                  label="Enable Auto-Rotation"
                />
              </Grid>
              <Grid item xs={12}>
                <Typography variant="subtitle2" gutterBottom>
                  Certificate PEM
                </Typography>
                <input
                  type="file"
                  accept=".pem,.crt"
                  onChange={(e) => handleFileUpload(e, 'certificate_pem')}
                  style={{ marginBottom: 8 }}
                />
                <TextField
                  fullWidth
                  multiline
                  rows={6}
                  value={uploadForm.certificate_pem}
                  onChange={(e) =>
                    setUploadForm({
                      ...uploadForm,
                      certificate_pem: e.target.value
                    })
                  }
                  placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                  sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}
                />
              </Grid>
              <Grid item xs={12}>
                <Typography variant="subtitle2" gutterBottom>
                  Private Key PEM (Optional for CA certificates)
                </Typography>
                <input
                  type="file"
                  accept=".pem,.key"
                  onChange={(e) => handleFileUpload(e, 'private_key_pem')}
                  style={{ marginBottom: 8 }}
                />
                <TextField
                  fullWidth
                  multiline
                  rows={6}
                  value={uploadForm.private_key_pem}
                  onChange={(e) =>
                    setUploadForm({
                      ...uploadForm,
                      private_key_pem: e.target.value
                    })
                  }
                  placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
                  sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}
                />
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setUploadDialogOpen(false)}>Cancel</Button>
            <Button
              variant="contained"
              onClick={handleUploadCertificate}
              disabled={loading}
            >
              Upload
            </Button>
          </DialogActions>
        </Dialog>

        {/* Certificate Details Dialog */}
        <Dialog
          open={detailsDialogOpen}
          onClose={() => setDetailsDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Certificate Validation</DialogTitle>
          <DialogContent>
            {selectedCert && validation && (
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  {validation.is_valid ? (
                    <Alert severity="success">Certificate is valid</Alert>
                  ) : (
                    <Alert severity="error">Certificate validation failed</Alert>
                  )}
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Subject
                  </Typography>
                  <Typography>{selectedCert.subject}</Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Issuer
                  </Typography>
                  <Typography>{selectedCert.issuer}</Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Valid From
                  </Typography>
                  <Typography>
                    {new Date(selectedCert.not_before).toLocaleString()}
                  </Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Valid Until
                  </Typography>
                  <Typography>
                    {new Date(selectedCert.not_after).toLocaleString()}
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Expires In
                  </Typography>
                  <Typography>
                    {validation.expires_in_days} days
                  </Typography>
                  {validation.expires_in_days <= 30 && (
                    <LinearProgress
                      variant="determinate"
                      value={(validation.expires_in_days / 30) * 100}
                      color="warning"
                      sx={{ mt: 1 }}
                    />
                  )}
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Fingerprint
                  </Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                    {selectedCert.fingerprint}
                  </Typography>
                </Grid>
                {validation.errors.length > 0 && (
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" color="error" gutterBottom>
                      Errors
                    </Typography>
                    <ul style={{ margin: 0, paddingLeft: 20 }}>
                      {validation.errors.map((err, idx) => (
                        <li key={idx}>
                          <Typography variant="body2" color="error">
                            {err}
                          </Typography>
                        </li>
                      ))}
                    </ul>
                  </Grid>
                )}
                {validation.warnings.length > 0 && (
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" color="warning.main" gutterBottom>
                      Warnings
                    </Typography>
                    <ul style={{ margin: 0, paddingLeft: 20 }}>
                      {validation.warnings.map((warn, idx) => (
                        <li key={idx}>
                          <Typography variant="body2" color="warning.main">
                            {warn}
                          </Typography>
                        </li>
                      ))}
                    </ul>
                  </Grid>
                )}
              </Grid>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDetailsDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Revoke Certificate Dialog */}
        <Dialog open={revokeDialogOpen} onClose={() => setRevokeDialogOpen(false)}>
          <DialogTitle>Revoke Certificate</DialogTitle>
          <DialogContent>
            <Typography paragraph>
              Are you sure you want to revoke certificate "{selectedCert?.name}"?
            </Typography>
            <TextField
              fullWidth
              label="Revocation Reason"
              value={revokeReason}
              onChange={(e) => setRevokeReason(e.target.value)}
              multiline
              rows={3}
              required
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setRevokeDialogOpen(false)}>Cancel</Button>
            <Button
              variant="contained"
              color="warning"
              onClick={handleRevokeCertificate}
              disabled={loading || !revokeReason}
            >
              Revoke
            </Button>
          </DialogActions>
        </Dialog>

        {/* Delete Certificate Dialog */}
        <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
          <DialogTitle>Delete Certificate</DialogTitle>
          <DialogContent>
            <Typography>
              Are you sure you want to delete certificate "{selectedCert?.name}"?
              This action cannot be undone.
            </Typography>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
            <Button
              variant="contained"
              color="error"
              onClick={handleDeleteCertificate}
              disabled={loading}
            >
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

export default mTLS;
