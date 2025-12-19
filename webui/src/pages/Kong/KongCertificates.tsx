/**
 * Kong Certificates and SNIs Management Page
 */
import React, { useEffect, useState, useCallback } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Alert,
  IconButton,
  Tooltip,
  Typography,
  Chip,
  Collapse,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import SecurityIcon from '@mui/icons-material/Security';
import { kongApi, KongCertificate, KongSNI } from '../../services/kongApi';

const KongCertificates: React.FC = () => {
  const [certificates, setCertificates] = useState<KongCertificate[]>([]);
  const [snis, setSNIs] = useState<KongSNI[]>([]);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [sniDialogOpen, setSNIDialogOpen] = useState(false);
  const [selectedCertId, setSelectedCertId] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    cert: '',
    key: '',
  });
  const [sniFormData, setSNIFormData] = useState({
    name: '',
  });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [certsRes, snisRes] = await Promise.all([
        kongApi.getCertificates(),
        kongApi.getSNIs(),
      ]);
      setCertificates(certsRes.data.data || []);
      setSNIs(snisRes.data.data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load certificates');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const toggleRow = (certId: string) => {
    setExpandedRow(expandedRow === certId ? null : certId);
  };

  const getCertSNIs = (certId: string): KongSNI[] => {
    return snis.filter((sni) => sni.certificate?.id === certId);
  };

  const handleOpenDialog = () => {
    setFormData({ cert: '', key: '' });
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
  };

  const handleSubmit = async () => {
    try {
      await kongApi.createCertificate(formData);
      handleCloseDialog();
      fetchData();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to save certificate';
      setError(errorMessage);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this certificate?')) {
      try {
        await kongApi.deleteCertificate(id);
        fetchData();
      } catch (err) {
        setError('Failed to delete certificate');
      }
    }
  };

  const handleOpenSNIDialog = (certId: string) => {
    setSelectedCertId(certId);
    setSNIFormData({ name: '' });
    setSNIDialogOpen(true);
  };

  const handleCloseSNIDialog = () => {
    setSNIDialogOpen(false);
    setSelectedCertId(null);
  };

  const handleAddSNI = async () => {
    if (!selectedCertId) return;
    try {
      await kongApi.createSNI({
        name: sniFormData.name,
        certificate: { id: selectedCertId },
      });
      handleCloseSNIDialog();
      fetchData();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add SNI';
      setError(errorMessage);
    }
  };

  const handleDeleteSNI = async (sniId: string) => {
    if (window.confirm('Are you sure you want to delete this SNI?')) {
      try {
        await kongApi.deleteSNI(sniId);
        fetchData();
      } catch (err) {
        setError('Failed to delete SNI');
      }
    }
  };

  const columns: GridColDef[] = [
    {
      field: 'expand',
      headerName: '',
      width: 50,
      renderCell: (params) => (
        <IconButton size="small" onClick={() => toggleRow(params.row.id)}>
          {expandedRow === params.row.id ? <ExpandLessIcon /> : <ExpandMoreIcon />}
        </IconButton>
      ),
    },
    {
      field: 'id',
      headerName: 'Certificate ID',
      flex: 1,
      minWidth: 300,
      renderCell: (params) => (
        <Box display="flex" alignItems="center" gap={1}>
          <SecurityIcon color="primary" fontSize="small" />
          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
            {params.value}
          </Typography>
        </Box>
      ),
    },
    {
      field: 'snis_count',
      headerName: 'SNIs',
      width: 100,
      valueGetter: (_, row) => getCertSNIs(row.id).length,
      renderCell: (params) => (
        <Chip label={params.value} size="small" color="primary" variant="outlined" />
      ),
    },
    {
      field: 'created_at',
      headerName: 'Created',
      width: 180,
      valueFormatter: (value) =>
        value ? new Date(value * 1000).toLocaleString() : '-',
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 120,
      getActions: (params) => [
        <GridActionsCellItem
          key="add-sni"
          icon={<AddIcon />}
          label="Add SNI"
          onClick={() => handleOpenSNIDialog(params.row.id)}
        />,
        <GridActionsCellItem
          key="delete"
          icon={<DeleteIcon />}
          label="Delete"
          onClick={() => handleDelete(params.row.id)}
        />,
      ],
    },
  ];

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Box>
          <Button variant="contained" startIcon={<AddIcon />} onClick={handleOpenDialog}>
            Add Certificate
          </Button>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchData} sx={{ ml: 1 }}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {certificates.map((cert) => (
        <Box key={cert.id} mb={1}>
          <DataGrid
            rows={[cert]}
            columns={columns}
            loading={loading}
            hideFooter
            autoHeight
            disableRowSelectionOnClick
          />
          <Collapse in={expandedRow === cert.id}>
            <Box sx={{ pl: 4, pr: 2, py: 1, bgcolor: 'action.hover' }}>
              <Typography variant="subtitle2" gutterBottom>
                SNIs (Server Name Indications)
              </Typography>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Domain Name</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {getCertSNIs(cert.id).map((sni) => (
                    <TableRow key={sni.id}>
                      <TableCell>{sni.name}</TableCell>
                      <TableCell>
                        <IconButton size="small" onClick={() => handleDeleteSNI(sni.id)}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                  {getCertSNIs(cert.id).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={2}>No SNIs configured</TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </Box>
          </Collapse>
        </Box>
      ))}

      {certificates.length === 0 && !loading && (
        <Alert severity="info">No certificates configured. Add one to enable HTTPS.</Alert>
      )}

      {/* Certificate Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>Add Certificate</DialogTitle>
        <DialogContent>
          <TextField
            margin="dense"
            label="Certificate (PEM format)"
            multiline
            rows={8}
            fullWidth
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
            value={formData.cert}
            onChange={(e) => setFormData({ ...formData, cert: e.target.value })}
            sx={{ fontFamily: 'monospace' }}
          />
          <TextField
            margin="dense"
            label="Private Key (PEM format)"
            multiline
            rows={8}
            fullWidth
            placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
            value={formData.key}
            onChange={(e) => setFormData({ ...formData, key: e.target.value })}
            sx={{ fontFamily: 'monospace' }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* SNI Dialog */}
      <Dialog open={sniDialogOpen} onClose={handleCloseSNIDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Add SNI</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Domain Name"
            fullWidth
            placeholder="api.example.com"
            value={sniFormData.name}
            onChange={(e) => setSNIFormData({ name: e.target.value })}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseSNIDialog}>Cancel</Button>
          <Button onClick={handleAddSNI} variant="contained">Add</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default KongCertificates;
