/**
 * Kong Upstreams and Targets Management Page
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
  MenuItem,
  Collapse,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Typography,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { kongApi, KongUpstream, KongTarget } from '../../services/kongApi';

const ALGORITHMS = ['round-robin', 'consistent-hashing', 'least-connections', 'latency'];
const HASH_ON = ['none', 'consumer', 'ip', 'header', 'cookie', 'path', 'query_arg', 'uri_capture'];

const KongUpstreams: React.FC = () => {
  const [upstreams, setUpstreams] = useState<KongUpstream[]>([]);
  const [targets, setTargets] = useState<Record<string, KongTarget[]>>({});
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [targetDialogOpen, setTargetDialogOpen] = useState(false);
  const [editingUpstream, setEditingUpstream] = useState<KongUpstream | null>(null);
  const [selectedUpstreamId, setSelectedUpstreamId] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    algorithm: 'round-robin',
    hash_on: 'none',
    slots: 10000,
  });
  const [targetFormData, setTargetFormData] = useState({
    target: '',
    weight: 100,
  });

  const fetchUpstreams = useCallback(async () => {
    setLoading(true);
    try {
      const response = await kongApi.getUpstreams();
      setUpstreams(response.data.data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load upstreams');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTargets = async (upstreamId: string) => {
    try {
      const response = await kongApi.getTargets(upstreamId);
      setTargets((prev) => ({ ...prev, [upstreamId]: response.data.data || [] }));
    } catch (err) {
      console.error('Failed to load targets', err);
    }
  };

  useEffect(() => {
    fetchUpstreams();
  }, [fetchUpstreams]);

  const toggleRow = async (upstreamId: string) => {
    if (expandedRow === upstreamId) {
      setExpandedRow(null);
    } else {
      setExpandedRow(upstreamId);
      if (!targets[upstreamId]) {
        await fetchTargets(upstreamId);
      }
    }
  };

  const handleOpenDialog = (upstream?: KongUpstream) => {
    if (upstream) {
      setEditingUpstream(upstream);
      setFormData({
        name: upstream.name,
        algorithm: upstream.algorithm || 'round-robin',
        hash_on: upstream.hash_on || 'none',
        slots: upstream.slots ?? 10000,
      });
    } else {
      setEditingUpstream(null);
      setFormData({
        name: '',
        algorithm: 'round-robin',
        hash_on: 'none',
        slots: 10000,
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingUpstream(null);
  };

  const handleSubmit = async () => {
    try {
      if (editingUpstream) {
        await kongApi.updateUpstream(editingUpstream.id, formData);
      } else {
        await kongApi.createUpstream(formData);
      }
      handleCloseDialog();
      fetchUpstreams();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to save upstream';
      setError(errorMessage);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this upstream?')) {
      try {
        await kongApi.deleteUpstream(id);
        fetchUpstreams();
      } catch (err) {
        setError('Failed to delete upstream');
      }
    }
  };

  const handleOpenTargetDialog = (upstreamId: string) => {
    setSelectedUpstreamId(upstreamId);
    setTargetFormData({ target: '', weight: 100 });
    setTargetDialogOpen(true);
  };

  const handleCloseTargetDialog = () => {
    setTargetDialogOpen(false);
    setSelectedUpstreamId(null);
  };

  const handleAddTarget = async () => {
    if (!selectedUpstreamId) return;
    try {
      await kongApi.createTarget(selectedUpstreamId, targetFormData);
      handleCloseTargetDialog();
      fetchTargets(selectedUpstreamId);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add target';
      setError(errorMessage);
    }
  };

  const handleDeleteTarget = async (upstreamId: string, targetId: string) => {
    if (window.confirm('Are you sure you want to delete this target?')) {
      try {
        await kongApi.deleteTarget(upstreamId, targetId);
        fetchTargets(upstreamId);
      } catch (err) {
        setError('Failed to delete target');
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
    { field: 'name', headerName: 'Name', flex: 1, minWidth: 200 },
    { field: 'algorithm', headerName: 'Algorithm', width: 150 },
    { field: 'hash_on', headerName: 'Hash On', width: 120 },
    { field: 'slots', headerName: 'Slots', width: 100, type: 'number' },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 150,
      getActions: (params) => [
        <GridActionsCellItem
          key="targets"
          icon={<AddIcon />}
          label="Add Target"
          onClick={() => handleOpenTargetDialog(params.row.id)}
        />,
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
            Add Upstream
          </Button>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchUpstreams} sx={{ ml: 1 }}>
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

      {upstreams.map((upstream) => (
        <Box key={upstream.id} mb={1}>
          <DataGrid
            rows={[upstream]}
            columns={columns}
            loading={loading}
            hideFooter
            autoHeight
            disableRowSelectionOnClick
          />
          <Collapse in={expandedRow === upstream.id}>
            <Box sx={{ pl: 4, pr: 2, py: 1, bgcolor: 'action.hover' }}>
              <Typography variant="subtitle2" gutterBottom>
                Targets
              </Typography>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Target (host:port)</TableCell>
                    <TableCell>Weight</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(targets[upstream.id] || []).map((target) => (
                    <TableRow key={target.id}>
                      <TableCell>{target.target}</TableCell>
                      <TableCell>{target.weight}</TableCell>
                      <TableCell>
                        <IconButton
                          size="small"
                          onClick={() => handleDeleteTarget(upstream.id, target.id)}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                  {(targets[upstream.id] || []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3}>No targets</TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </Box>
          </Collapse>
        </Box>
      ))}

      {/* Upstream Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>{editingUpstream ? 'Edit Upstream' : 'Add Upstream'}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Name"
            fullWidth
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={!!editingUpstream}
          />
          <TextField
            select
            margin="dense"
            label="Algorithm"
            fullWidth
            value={formData.algorithm}
            onChange={(e) => setFormData({ ...formData, algorithm: e.target.value })}
          >
            {ALGORITHMS.map((alg) => (
              <MenuItem key={alg} value={alg}>{alg}</MenuItem>
            ))}
          </TextField>
          <TextField
            select
            margin="dense"
            label="Hash On"
            fullWidth
            value={formData.hash_on}
            onChange={(e) => setFormData({ ...formData, hash_on: e.target.value })}
          >
            {HASH_ON.map((h) => (
              <MenuItem key={h} value={h}>{h}</MenuItem>
            ))}
          </TextField>
          <TextField
            margin="dense"
            label="Slots"
            type="number"
            fullWidth
            value={formData.slots}
            onChange={(e) => setFormData({ ...formData, slots: parseInt(e.target.value) })}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">
            {editingUpstream ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Target Dialog */}
      <Dialog open={targetDialogOpen} onClose={handleCloseTargetDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Add Target</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Target (host:port)"
            fullWidth
            placeholder="192.168.1.100:8080"
            value={targetFormData.target}
            onChange={(e) => setTargetFormData({ ...targetFormData, target: e.target.value })}
          />
          <TextField
            margin="dense"
            label="Weight"
            type="number"
            fullWidth
            value={targetFormData.weight}
            onChange={(e) => setTargetFormData({ ...targetFormData, weight: parseInt(e.target.value) })}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseTargetDialog}>Cancel</Button>
          <Button onClick={handleAddTarget} variant="contained">Add</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default KongUpstreams;
