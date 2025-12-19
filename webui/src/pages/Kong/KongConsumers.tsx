/**
 * Kong Consumers Management Page
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
  Chip,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import { kongApi, KongConsumer } from '../../services/kongApi';

const KongConsumers: React.FC = () => {
  const [consumers, setConsumers] = useState<KongConsumer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingConsumer, setEditingConsumer] = useState<KongConsumer | null>(null);
  const [formData, setFormData] = useState({
    username: '',
    custom_id: '',
    tags: [] as string[],
  });
  const [tagInput, setTagInput] = useState('');

  const fetchConsumers = useCallback(async () => {
    setLoading(true);
    try {
      const response = await kongApi.getConsumers();
      setConsumers(response.data.data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load consumers');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConsumers();
  }, [fetchConsumers]);

  const handleOpenDialog = (consumer?: KongConsumer) => {
    if (consumer) {
      setEditingConsumer(consumer);
      setFormData({
        username: consumer.username || '',
        custom_id: consumer.custom_id || '',
        tags: consumer.tags || [],
      });
    } else {
      setEditingConsumer(null);
      setFormData({
        username: '',
        custom_id: '',
        tags: [],
      });
    }
    setTagInput('');
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingConsumer(null);
  };

  const handleSubmit = async () => {
    try {
      const payload: Partial<KongConsumer> = {};
      if (formData.username) payload.username = formData.username;
      if (formData.custom_id) payload.custom_id = formData.custom_id;
      if (formData.tags.length > 0) payload.tags = formData.tags;

      if (editingConsumer) {
        await kongApi.updateConsumer(editingConsumer.id, payload);
      } else {
        await kongApi.createConsumer(payload);
      }
      handleCloseDialog();
      fetchConsumers();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to save consumer';
      setError(errorMessage);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this consumer?')) {
      try {
        await kongApi.deleteConsumer(id);
        fetchConsumers();
      } catch (err) {
        setError('Failed to delete consumer');
      }
    }
  };

  const addTag = () => {
    if (tagInput && !formData.tags.includes(tagInput)) {
      setFormData({ ...formData, tags: [...formData.tags, tagInput] });
      setTagInput('');
    }
  };

  const columns: GridColDef[] = [
    { field: 'username', headerName: 'Username', flex: 1, minWidth: 200 },
    { field: 'custom_id', headerName: 'Custom ID', flex: 1, minWidth: 200 },
    {
      field: 'tags',
      headerName: 'Tags',
      flex: 1,
      renderCell: (params) =>
        (params.value || []).map((tag: string) => (
          <Chip key={tag} label={tag} size="small" sx={{ mr: 0.5 }} />
        )),
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
      width: 100,
      getActions: (params) => [
        <GridActionsCellItem
          key="edit"
          icon={<EditIcon />}
          label="Edit"
          onClick={() => handleOpenDialog(params.row)}
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
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => handleOpenDialog()}>
            Add Consumer
          </Button>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchConsumers} sx={{ ml: 1 }}>
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

      <DataGrid
        rows={consumers}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{ pagination: { paginationModel: { pageSize: 10 } } }}
        disableRowSelectionOnClick
      />

      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>{editingConsumer ? 'Edit Consumer' : 'Add Consumer'}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Username"
            fullWidth
            value={formData.username}
            onChange={(e) => setFormData({ ...formData, username: e.target.value })}
            helperText="At least one of username or custom_id is required"
          />
          <TextField
            margin="dense"
            label="Custom ID"
            fullWidth
            value={formData.custom_id}
            onChange={(e) => setFormData({ ...formData, custom_id: e.target.value })}
          />

          <Box display="flex" gap={1} alignItems="center" mt={2}>
            <TextField
              label="Add Tag"
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              size="small"
              onKeyPress={(e) => e.key === 'Enter' && addTag()}
            />
            <Button onClick={addTag} size="small">Add</Button>
          </Box>
          <Box display="flex" gap={0.5} flexWrap="wrap" mt={1}>
            {formData.tags.map((tag) => (
              <Chip
                key={tag}
                label={tag}
                onDelete={() => setFormData({ ...formData, tags: formData.tags.filter((t) => t !== tag) })}
                size="small"
              />
            ))}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">
            {editingConsumer ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default KongConsumers;
