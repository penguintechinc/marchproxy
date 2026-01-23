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
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  OutlinedInput,
  SelectChangeEvent,
} from '@mui/material';
import {
  DataGrid,
  GridColDef,
  GridActionsCellItem,
  GridRowParams,
} from '@mui/x-data-grid';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  Person as PersonIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { usersApi, CreateUserRequest, UpdateUserRequest } from '@services/users';
import { clusterApi } from '@services/clusterApi';
import { User, Cluster } from '@services/types';

interface UserFormData {
  username: string;
  email: string;
  password?: string;
  role: 'administrator' | 'service_owner';
  clusters: number[];
}

const Users: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);

  const { control, handleSubmit, reset, watch, formState: { errors } } = useForm<UserFormData>({
    defaultValues: {
      username: '',
      email: '',
      password: '',
      role: 'service_owner',
      clusters: [],
    }
  });

  const roleValue = watch('role');

  useEffect(() => {
    fetchClusters();
    fetchUsers();
  }, []);

  const fetchClusters = async () => {
    try {
      const response = await clusterApi.list();
      setClusters(response.items);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load clusters');
    }
  };

  const fetchUsers = async () => {
    try {
      setLoading(true);
      const response = await usersApi.list();
      setUsers(response.items);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load users');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditMode(false);
    setSelectedUser(null);
    reset({
      username: '',
      email: '',
      password: '',
      role: 'service_owner',
      clusters: [],
    });
    setOpenDialog(true);
  };

  const handleEdit = (user: User) => {
    setEditMode(true);
    setSelectedUser(user);
    reset({
      username: user.username,
      email: user.email,
      password: '',
      role: user.role,
      clusters: user.clusters || [],
    });
    setOpenDialog(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await usersApi.delete(id);
      fetchUsers();
      setDeleteConfirm(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to delete user');
    }
  };

  const onSubmit = async (data: UserFormData) => {
    try {
      if (editMode && selectedUser) {
        const updateData: UpdateUserRequest = {
          username: data.username,
          email: data.email,
          role: data.role,
          clusters: data.clusters,
        };
        await usersApi.update(selectedUser.id, updateData);
      } else {
        const createData: CreateUserRequest = {
          username: data.username,
          email: data.email,
          password: data.password || '',
          role: data.role,
          clusters: data.clusters,
        };
        await usersApi.create(createData);
      }
      setOpenDialog(false);
      fetchUsers();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to save user');
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getClusterNames = (clusterIds?: number[]) => {
    if (!clusterIds || clusterIds.length === 0) return 'None';
    return clusterIds
      .map(id => clusters.find(c => c.id === id)?.name || `ID:${id}`)
      .join(', ');
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    {
      field: 'username',
      headerName: 'Username',
      width: 150,
      flex: 1,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <PersonIcon fontSize="small" color="action" />
          {params.value}
        </Box>
      ),
    },
    {
      field: 'email',
      headerName: 'Email',
      width: 200,
      flex: 1,
    },
    {
      field: 'role',
      headerName: 'Role',
      width: 150,
      renderCell: (params) => (
        <Chip
          label={params.value === 'administrator' ? 'Administrator' : 'Service Owner'}
          size="small"
          color={params.value === 'administrator' ? 'error' : 'primary'}
          variant="outlined"
        />
      ),
    },
    {
      field: 'clusters',
      headerName: 'Clusters',
      width: 200,
      flex: 1,
      renderCell: (params) => {
        const clusterCount = params.value?.length || 0;
        return (
          <Chip
            label={clusterCount === 0 ? 'None' : `${clusterCount} cluster${clusterCount > 1 ? 's' : ''}`}
            size="small"
            color={clusterCount > 0 ? 'success' : 'default'}
          />
        );
      },
    },
    {
      field: 'created_at',
      headerName: 'Created',
      width: 180,
      renderCell: (params) => formatDate(params.value),
    },
    {
      field: 'updated_at',
      headerName: 'Last Updated',
      width: 180,
      renderCell: (params) => formatDate(params.value),
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 120,
      getActions: (params: GridRowParams<User>) => [
        <GridActionsCellItem
          icon={<EditIcon />}
          label="Edit"
          onClick={() => handleEdit(params.row)}
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
          User Management
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchUsers}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
          >
            Add User
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <DataGrid
        rows={users}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{
          pagination: { paginationModel: { pageSize: 10 } },
        }}
        disableRowSelectionOnClick
      />

      {/* Create/Edit Dialog */}
      <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editMode ? 'Edit User' : 'Create User'}</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DialogContent>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Controller
                name="username"
                control={control}
                rules={{
                  required: 'Username is required',
                  minLength: { value: 3, message: 'Username must be at least 3 characters' },
                  pattern: {
                    value: /^[a-zA-Z0-9_-]+$/,
                    message: 'Username can only contain letters, numbers, underscores, and hyphens'
                  }
                }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Username"
                    fullWidth
                    error={!!errors.username}
                    helperText={errors.username?.message}
                    disabled={editMode}
                  />
                )}
              />
              <Controller
                name="email"
                control={control}
                rules={{
                  required: 'Email is required',
                  pattern: {
                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                    message: 'Invalid email address'
                  }
                }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Email"
                    type="email"
                    fullWidth
                    error={!!errors.email}
                    helperText={errors.email?.message}
                  />
                )}
              />
              {!editMode && (
                <Controller
                  name="password"
                  control={control}
                  rules={{
                    required: editMode ? false : 'Password is required',
                    minLength: { value: 8, message: 'Password must be at least 8 characters' }
                  }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Password"
                      type="password"
                      fullWidth
                      error={!!errors.password}
                      helperText={errors.password?.message || 'Minimum 8 characters'}
                    />
                  )}
                />
              )}
              <Controller
                name="role"
                control={control}
                rules={{ required: 'Role is required' }}
                render={({ field }) => (
                  <FormControl fullWidth error={!!errors.role}>
                    <InputLabel>Role</InputLabel>
                    <Select {...field} label="Role">
                      <MenuItem value="service_owner">Service Owner</MenuItem>
                      <MenuItem value="administrator">Administrator</MenuItem>
                    </Select>
                  </FormControl>
                )}
              />
              {roleValue === 'service_owner' && (
                <Controller
                  name="clusters"
                  control={control}
                  render={({ field }) => (
                    <FormControl fullWidth>
                      <InputLabel>Cluster Assignments</InputLabel>
                      <Select
                        {...field}
                        multiple
                        label="Cluster Assignments"
                        input={<OutlinedInput label="Cluster Assignments" />}
                        renderValue={(selected) => (
                          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                            {(selected as number[]).map((value) => {
                              const cluster = clusters.find(c => c.id === value);
                              return (
                                <Chip
                                  key={value}
                                  label={cluster?.name || `ID:${value}`}
                                  size="small"
                                />
                              );
                            })}
                          </Box>
                        )}
                      >
                        {clusters.map((cluster) => (
                          <MenuItem key={cluster.id} value={cluster.id}>
                            {cluster.name}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  )}
                />
              )}
              {roleValue === 'administrator' && (
                <Alert severity="info">
                  Administrators have access to all clusters automatically.
                </Alert>
              )}
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
            <Button type="submit" variant="contained">
              {editMode ? 'Update' : 'Create'}
            </Button>
          </DialogActions>
        </form>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirm !== null} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete this user? This action cannot be undone.
          </Typography>
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

export default Users;
