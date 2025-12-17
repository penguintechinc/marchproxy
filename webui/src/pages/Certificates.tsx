import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Typography,
  Alert,
  Chip,
  Switch,
  FormControlLabel,
  Tabs,
  Tab,
} from '@mui/material';
import {
  DataGrid,
  GridColDef,
  GridActionsCellItem,
  GridRowParams,
} from '@mui/x-data-grid';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  CloudUpload as UploadIcon,
  Settings as VaultIcon,
  Security as InfisicalIcon,
  AutorenewOutlined as RenewIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import {
  certificateApi,
  Certificate,
  UploadCertificateRequest,
  InfisicalCertificateRequest,
  VaultCertificateRequest,
} from '@services/certificateApi';
import { formatDistanceToNow } from 'date-fns';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
  <div hidden={value !== index}>
    {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
  </div>
);

const Certificates: React.FC = () => {
  const [certificates, setCertificates] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);
  const [tabValue, setTabValue] = useState(0);

  const uploadForm = useForm<UploadCertificateRequest>({
    defaultValues: {
      name: '',
      source_type: 'upload',
      cert_data: '',
      key_data: '',
      ca_chain: '',
      auto_renew: false,
    }
  });

  const infisicalForm = useForm<InfisicalCertificateRequest>({
    defaultValues: {
      name: '',
      source_type: 'infisical',
      infisical_secret_path: '/certificates',
      infisical_project_id: '',
      infisical_environment: 'production',
      auto_renew: true,
    }
  });

  const vaultForm = useForm<VaultCertificateRequest>({
    defaultValues: {
      name: '',
      source_type: 'vault',
      vault_path: 'secret/certificates',
      vault_role: '',
      vault_common_name: '',
      auto_renew: true,
    }
  });

  useEffect(() => {
    fetchCertificates();
  }, []);

  const fetchCertificates = async () => {
    try {
      setLoading(true);
      const response = await certificateApi.list();
      setCertificates(response.items);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load certificates');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await certificateApi.delete(id);
      fetchCertificates();
      setDeleteConfirm(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to delete certificate');
    }
  };

  const handleToggleAutoRenew = async (id: number, enabled: boolean) => {
    try {
      await certificateApi.toggleAutoRenew(id, enabled);
      fetchCertificates();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to toggle auto-renewal');
    }
  };

  const handleRenew = async (id: number) => {
    try {
      await certificateApi.renew(id);
      fetchCertificates();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to renew certificate');
    }
  };

  const onUploadSubmit = async (data: UploadCertificateRequest) => {
    try {
      await certificateApi.upload(data);
      setOpenDialog(false);
      fetchCertificates();
      uploadForm.reset();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to upload certificate');
    }
  };

  const onInfisicalSubmit = async (data: InfisicalCertificateRequest) => {
    try {
      await certificateApi.configureInfisical(data);
      setOpenDialog(false);
      fetchCertificates();
      infisicalForm.reset();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to configure Infisical');
    }
  };

  const onVaultSubmit = async (data: VaultCertificateRequest) => {
    try {
      await certificateApi.configureVault(data);
      setOpenDialog(false);
      fetchCertificates();
      vaultForm.reset();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to configure Vault');
    }
  };

  const getExpiryColor = (days: number): 'success' | 'warning' | 'error' | 'default' => {
    if (days < 0) return 'error';
    if (days < 30) return 'error';
    if (days < 90) return 'warning';
    return 'success';
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    { field: 'name', headerName: 'Name', width: 200, flex: 1 },
    { field: 'common_name', headerName: 'Common Name', width: 250, flex: 1 },
    {
      field: 'source_type',
      headerName: 'Source',
      width: 120,
      renderCell: (params) => (
        <Chip
          label={params.value}
          size="small"
          color={params.value === 'upload' ? 'default' : 'primary'}
          sx={{ textTransform: 'capitalize' }}
        />
      ),
    },
    {
      field: 'days_until_expiry',
      headerName: 'Expires',
      width: 150,
      renderCell: (params) => {
        const days = params.value as number;
        const color = getExpiryColor(days);
        if (days < 0) {
          return <Chip label="Expired" size="small" color="error" />;
        }
        return (
          <Chip
            label={`${days} days`}
            size="small"
            color={color}
          />
        );
      },
    },
    {
      field: 'auto_renew',
      headerName: 'Auto-Renew',
      width: 120,
      renderCell: (params) => (
        <Switch
          checked={params.value}
          onChange={(e) => handleToggleAutoRenew(params.row.id, e.target.checked)}
          size="small"
        />
      ),
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 120,
      getActions: (params: GridRowParams<Certificate>) => [
        <GridActionsCellItem
          icon={<RenewIcon />}
          label="Renew"
          onClick={() => handleRenew(params.row.id)}
          disabled={params.row.source_type === 'upload'}
        />,
        <GridActionsCellItem
          icon={<DeleteIcon />}
          label="Delete"
          onClick={() => setDeleteConfirm(params.row.id)}
        />,
      ],
    },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4" fontWeight="bold">
          Certificate Management
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchCertificates}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setOpenDialog(true)}
          >
            Add Certificate
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <DataGrid
        rows={certificates}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{
          pagination: { paginationModel: { pageSize: 10 } },
        }}
        disableRowSelectionOnClick
      />

      {/* Add Certificate Dialog */}
      <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>Add Certificate</DialogTitle>
        <DialogContent>
          <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)}>
            <Tab icon={<UploadIcon />} label="Upload" />
            <Tab icon={<InfisicalIcon />} label="Infisical" />
            <Tab icon={<VaultIcon />} label="Vault" />
          </Tabs>

          {/* Upload Tab */}
          <TabPanel value={tabValue} index={0}>
            <form onSubmit={uploadForm.handleSubmit(onUploadSubmit)}>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Controller
                  name="name"
                  control={uploadForm.control}
                  rules={{ required: 'Name is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Certificate Name"
                      fullWidth
                      error={!!uploadForm.formState.errors.name}
                      helperText={uploadForm.formState.errors.name?.message}
                    />
                  )}
                />
                <Controller
                  name="cert_data"
                  control={uploadForm.control}
                  rules={{ required: 'Certificate is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Certificate (PEM format)"
                      fullWidth
                      multiline
                      rows={6}
                      error={!!uploadForm.formState.errors.cert_data}
                      helperText={uploadForm.formState.errors.cert_data?.message}
                    />
                  )}
                />
                <Controller
                  name="key_data"
                  control={uploadForm.control}
                  rules={{ required: 'Private key is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Private Key (PEM format)"
                      fullWidth
                      multiline
                      rows={6}
                      type="password"
                      error={!!uploadForm.formState.errors.key_data}
                      helperText={uploadForm.formState.errors.key_data?.message}
                    />
                  )}
                />
                <Controller
                  name="ca_chain"
                  control={uploadForm.control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="CA Chain (Optional, PEM format)"
                      fullWidth
                      multiline
                      rows={4}
                    />
                  )}
                />
                <Controller
                  name="auto_renew"
                  control={uploadForm.control}
                  render={({ field }) => (
                    <FormControlLabel
                      control={<Switch {...field} checked={field.value} />}
                      label="Enable Auto-Renewal (Not available for uploaded certificates)"
                      disabled
                    />
                  )}
                />
                <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2, mt: 2 }}>
                  <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
                  <Button type="submit" variant="contained">Upload</Button>
                </Box>
              </Box>
            </form>
          </TabPanel>

          {/* Infisical Tab */}
          <TabPanel value={tabValue} index={1}>
            <form onSubmit={infisicalForm.handleSubmit(onInfisicalSubmit)}>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Controller
                  name="name"
                  control={infisicalForm.control}
                  rules={{ required: 'Name is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Certificate Name"
                      fullWidth
                      error={!!infisicalForm.formState.errors.name}
                      helperText={infisicalForm.formState.errors.name?.message}
                    />
                  )}
                />
                <Controller
                  name="infisical_project_id"
                  control={infisicalForm.control}
                  rules={{ required: 'Project ID is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Project ID"
                      fullWidth
                      error={!!infisicalForm.formState.errors.infisical_project_id}
                      helperText={infisicalForm.formState.errors.infisical_project_id?.message}
                    />
                  )}
                />
                <Controller
                  name="infisical_secret_path"
                  control={infisicalForm.control}
                  rules={{ required: 'Secret Path is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Secret Path"
                      fullWidth
                      error={!!infisicalForm.formState.errors.infisical_secret_path}
                      helperText={infisicalForm.formState.errors.infisical_secret_path?.message}
                    />
                  )}
                />
                <Controller
                  name="infisical_environment"
                  control={infisicalForm.control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Environment"
                      fullWidth
                      defaultValue="production"
                    />
                  )}
                />
                <Controller
                  name="auto_renew"
                  control={infisicalForm.control}
                  render={({ field }) => (
                    <FormControlLabel
                      control={<Switch {...field} checked={field.value} />}
                      label="Enable Auto-Renewal"
                    />
                  )}
                />
                <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2, mt: 2 }}>
                  <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
                  <Button type="submit" variant="contained">Configure</Button>
                </Box>
              </Box>
            </form>
          </TabPanel>

          {/* Vault Tab */}
          <TabPanel value={tabValue} index={2}>
            <form onSubmit={vaultForm.handleSubmit(onVaultSubmit)}>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Controller
                  name="name"
                  control={vaultForm.control}
                  rules={{ required: 'Name is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Certificate Name"
                      fullWidth
                      error={!!vaultForm.formState.errors.name}
                      helperText={vaultForm.formState.errors.name?.message}
                    />
                  )}
                />
                <Controller
                  name="vault_path"
                  control={vaultForm.control}
                  rules={{ required: 'Vault Path is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Vault PKI Path"
                      fullWidth
                      placeholder="secret/certificates"
                      error={!!vaultForm.formState.errors.vault_path}
                      helperText={vaultForm.formState.errors.vault_path?.message}
                    />
                  )}
                />
                <Controller
                  name="vault_role"
                  control={vaultForm.control}
                  rules={{ required: 'Vault Role is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Vault PKI Role"
                      fullWidth
                      error={!!vaultForm.formState.errors.vault_role}
                      helperText={vaultForm.formState.errors.vault_role?.message}
                    />
                  )}
                />
                <Controller
                  name="vault_common_name"
                  control={vaultForm.control}
                  rules={{ required: 'Common Name is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Certificate Common Name"
                      fullWidth
                      placeholder="example.com"
                      error={!!vaultForm.formState.errors.vault_common_name}
                      helperText={vaultForm.formState.errors.vault_common_name?.message}
                    />
                  )}
                />
                <Controller
                  name="auto_renew"
                  control={vaultForm.control}
                  render={({ field }) => (
                    <FormControlLabel
                      control={<Switch {...field} checked={field.value} />}
                      label="Enable Auto-Renewal"
                    />
                  )}
                />
                <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2, mt: 2 }}>
                  <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
                  <Button type="submit" variant="contained">Configure</Button>
                </Box>
              </Box>
            </form>
          </TabPanel>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirm !== null} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>Are you sure you want to delete this certificate?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirm(null)}>Cancel</Button>
          <Button
            onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
            variant="contained"
            color="error"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Certificates;
