/**
 * ModuleRoutes Page
 *
 * Configure routes for a specific module.
 * Supports multiple routes per module with protocol-specific settings.
 */

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Typography,
  Box,
  Button,
  Alert,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Breadcrumbs,
  Link,
  LinearProgress,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ArrowBack as BackIcon,
} from '@mui/icons-material';
import RouteEditor from '../../components/Modules/RouteEditor';
import {
  getModule,
  getModuleRoutes,
  createModuleRoute,
  updateModuleRoute,
  deleteModuleRoute,
} from '../../services/modulesApi';
import type { Module, ModuleRoute } from '../../services/types';

const ModuleRoutesPage: React.FC = () => {
  const { moduleId } = useParams<{ moduleId: string }>();
  const navigate = useNavigate();

  const [module, setModule] = useState<Module | null>(null);
  const [routes, setRoutes] = useState<ModuleRoute[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRoute, setEditingRoute] = useState<ModuleRoute | undefined>(undefined);
  const [savingRoute, setSavingRoute] = useState(false);

  useEffect(() => {
    if (moduleId) {
      loadData();
    }
  }, [moduleId]);

  const loadData = async () => {
    if (!moduleId) return;
    try {
      setLoading(true);
      const [moduleData, routesData] = await Promise.all([
        getModule(parseInt(moduleId)),
        getModuleRoutes(parseInt(moduleId)),
      ]);
      setModule(moduleData);
      setRoutes(routesData);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load module routes');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRoute = () => {
    setEditingRoute(undefined);
    setEditorOpen(true);
  };

  const handleEditRoute = (route: ModuleRoute) => {
    setEditingRoute(route);
    setEditorOpen(true);
  };

  const handleSaveRoute = async (routeData: Partial<ModuleRoute>) => {
    if (!moduleId) return;
    try {
      setSavingRoute(true);
      if (editingRoute?.id) {
        await updateModuleRoute(parseInt(moduleId), editingRoute.id, routeData);
      } else {
        await createModuleRoute(parseInt(moduleId), routeData);
      }
      await loadData();
      setEditorOpen(false);
    } catch (err: any) {
      throw new Error(err.message || 'Failed to save route');
    } finally {
      setSavingRoute(false);
    }
  };

  const handleDeleteRoute = async (routeId: number) => {
    if (!moduleId) return;
    if (!confirm('Are you sure you want to delete this route?')) return;
    try {
      await deleteModuleRoute(parseInt(moduleId), routeId);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to delete route');
    }
  };

  const getPriorityColor = (priority: ModuleRoute['priority']): string => {
    switch (priority) {
      case 'P0':
        return 'error';
      case 'P1':
        return 'warning';
      case 'P2':
        return 'info';
      case 'P3':
        return 'success';
      default:
        return 'default';
    }
  };

  if (loading) {
    return (
      <Container maxWidth="xl">
        <Box py={4}>
          <LinearProgress />
        </Box>
      </Container>
    );
  }

  if (!module) {
    return (
      <Container maxWidth="xl">
        <Box py={4}>
          <Alert severity="error">Module not found</Alert>
        </Box>
      </Container>
    );
  }

  return (
    <Container maxWidth="xl">
      <Box py={4}>
        <Breadcrumbs sx={{ mb: 2 }}>
          <Link
            component="button"
            variant="body2"
            onClick={() => navigate('/modules')}
            sx={{ textDecoration: 'none' }}
          >
            Modules
          </Link>
          <Typography color="text.primary">{module.name}</Typography>
          <Typography color="text.primary">Routes</Typography>
        </Breadcrumbs>

        <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
          <Box>
            <Box display="flex" alignItems="center" gap={2} mb={1}>
              <IconButton onClick={() => navigate('/modules')}>
                <BackIcon />
              </IconButton>
              <Typography variant="h4" fontWeight="bold">
                {module.name} Routes
              </Typography>
              <Chip label={module.type} color="primary" />
            </Box>
            <Typography variant="body2" color="text.secondary" ml={7}>
              Configure multiple routes for {module.name} module
            </Typography>
          </Box>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreateRoute}
            disabled={!module.is_enabled}
          >
            Add Route
          </Button>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {!module.is_enabled && (
          <Alert severity="warning" sx={{ mb: 3 }}>
            This module is currently disabled. Enable it in the Module Manager to configure routes.
          </Alert>
        )}

        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Configured Routes ({routes.length})
            </Typography>

            {routes.length === 0 ? (
              <Box textAlign="center" py={4}>
                <Typography variant="body2" color="text.secondary">
                  No routes configured yet. Click "Add Route" to create one.
                </Typography>
              </Box>
            ) : (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>Name</TableCell>
                      <TableCell>Protocol</TableCell>
                      <TableCell>Backend</TableCell>
                      <TableCell>Priority</TableCell>
                      <TableCell>Rate Limits</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {routes.map((route) => (
                      <TableRow key={route.id}>
                        <TableCell>
                          <Typography variant="body2" fontWeight="medium">
                            {route.name}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip label={route.protocol} size="small" variant="outlined" />
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" noWrap>
                            {route.backend_url}:{route.backend_port}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={route.priority}
                            size="small"
                            color={getPriorityColor(route.priority) as any}
                          />
                        </TableCell>
                        <TableCell>
                          <Box display="flex" flexDirection="column" gap={0.5}>
                            {route.rate_limit_rps && (
                              <Typography variant="caption">
                                {route.rate_limit_rps} req/s
                              </Typography>
                            )}
                            {route.rate_limit_connections && (
                              <Typography variant="caption">
                                {route.rate_limit_connections} conns
                              </Typography>
                            )}
                            {route.rate_limit_bandwidth_mbps && (
                              <Typography variant="caption">
                                {route.rate_limit_bandwidth_mbps} Mbps
                              </Typography>
                            )}
                            {!route.rate_limit_rps &&
                              !route.rate_limit_connections &&
                              !route.rate_limit_bandwidth_mbps && (
                                <Typography variant="caption" color="text.secondary">
                                  No limits
                                </Typography>
                              )}
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={route.is_active ? 'Active' : 'Inactive'}
                            size="small"
                            color={route.is_active ? 'success' : 'default'}
                          />
                        </TableCell>
                        <TableCell align="right">
                          <IconButton size="small" onClick={() => handleEditRoute(route)}>
                            <EditIcon />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => handleDeleteRoute(route.id)}
                          >
                            <DeleteIcon />
                          </IconButton>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </CardContent>
        </Card>
      </Box>

      <RouteEditor
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        onSave={handleSaveRoute}
        route={editingRoute}
        moduleType={module.type}
        loading={savingRoute}
      />
    </Container>
  );
};

export default ModuleRoutesPage;
