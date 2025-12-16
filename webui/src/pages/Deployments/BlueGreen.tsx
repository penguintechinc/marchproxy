/**
 * BlueGreen Deployment Page
 *
 * Manage blue/green deployments for modules with traffic weight control.
 * Supports gradual traffic shifting, health monitoring, and instant rollback.
 */

import React, { useState, useEffect } from 'react';
import {
  Container,
  Typography,
  Box,
  Button,
  Alert,
  Card,
  CardContent,
  Grid,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControlLabel,
  Switch,
  LinearProgress,
  Stepper,
  Step,
  StepLabel,
  Divider,
} from '@mui/material';
import {
  Add as AddIcon,
  Undo as RollbackIcon,
  CheckCircle as FinalizeIcon,
  Timeline as TimelineIcon,
} from '@mui/icons-material';
import TrafficSlider from '../../components/Deployments/TrafficSlider';
import {
  getModules,
  getBlueGreenDeployments,
  getActiveDeployment,
  createBlueGreenDeployment,
  updateTrafficWeight,
  rollbackDeployment,
  finalizeDeployment,
} from '../../services/modulesApi';
import type { Module, BlueGreenDeployment } from '../../services/types';

const BlueGreen: React.FC = () => {
  const [modules, setModules] = useState<Module[]>([]);
  const [deployments, setDeployments] = useState<Record<number, BlueGreenDeployment | null>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedModule, setSelectedModule] = useState<number | null>(null);
  const [savingTraffic, setSavingTraffic] = useState<number | null>(null);

  const [formData, setFormData] = useState<Partial<BlueGreenDeployment>>({
    blue_version: '',
    green_version: '',
    traffic_weight_blue: 100,
    traffic_weight_green: 0,
    health_check_url: '',
    auto_rollback_enabled: true,
  });

  const [trafficWeights, setTrafficWeights] = useState<{
    [key: number]: { blue: number; green: number };
  }>({});

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 15000); // Refresh every 15s
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const modulesData = await getModules();
      setModules(modulesData.filter((m) => m.is_enabled));

      const deploymentsMap: Record<number, BlueGreenDeployment | null> = {};
      for (const module of modulesData.filter((m) => m.is_enabled)) {
        try {
          const deployment = await getActiveDeployment(module.id);
          deploymentsMap[module.id] = deployment;
          if (deployment) {
            setTrafficWeights((prev) => ({
              ...prev,
              [module.id]: {
                blue: deployment.traffic_weight_blue,
                green: deployment.traffic_weight_green,
              },
            }));
          }
        } catch {
          deploymentsMap[module.id] = null;
        }
      }
      setDeployments(deploymentsMap);

      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load deployments');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenDialog = (moduleId: number) => {
    setSelectedModule(moduleId);
    const module = modules.find((m) => m.id === moduleId);
    setFormData({
      blue_version: module?.version || '',
      green_version: '',
      traffic_weight_blue: 100,
      traffic_weight_green: 0,
      health_check_url: '',
      auto_rollback_enabled: true,
    });
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setSelectedModule(null);
  };

  const handleCreateDeployment = async () => {
    if (!selectedModule) return;
    try {
      await createBlueGreenDeployment(selectedModule, formData);
      await loadData();
      handleCloseDialog();
    } catch (err: any) {
      setError(err.message || 'Failed to create deployment');
    }
  };

  const handleTrafficChange = (moduleId: number, blueWeight: number, greenWeight: number) => {
    setTrafficWeights((prev) => ({
      ...prev,
      [moduleId]: { blue: blueWeight, green: greenWeight },
    }));
  };

  const handleApplyTraffic = async (moduleId: number, deploymentId: number) => {
    const weights = trafficWeights[moduleId];
    if (!weights) return;

    try {
      setSavingTraffic(moduleId);
      await updateTrafficWeight(moduleId, deploymentId, weights.blue, weights.green);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to update traffic weights');
    } finally {
      setSavingTraffic(null);
    }
  };

  const handleRollback = async (moduleId: number, deploymentId: number) => {
    if (!confirm('Are you sure you want to rollback this deployment? This will shift all traffic to the blue version.')) {
      return;
    }
    try {
      await rollbackDeployment(moduleId, deploymentId);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to rollback deployment');
    }
  };

  const handleFinalize = async (
    moduleId: number,
    deploymentId: number,
    targetVersion: 'blue' | 'green'
  ) => {
    if (!confirm(`Finalize deployment to ${targetVersion} version? This will end the blue/green deployment.`)) {
      return;
    }
    try {
      await finalizeDeployment(moduleId, deploymentId, targetVersion);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to finalize deployment');
    }
  };

  const getStatusColor = (status: BlueGreenDeployment['status']): string => {
    switch (status) {
      case 'active':
        return 'success';
      case 'transitioning':
        return 'warning';
      case 'rolled_back':
        return 'error';
      default:
        return 'default';
    }
  };

  const getDeploymentStep = (deployment: BlueGreenDeployment): number => {
    if (deployment.traffic_weight_blue === 100) return 0;
    if (deployment.traffic_weight_blue > 50) return 1;
    if (deployment.traffic_weight_blue === 50) return 2;
    if (deployment.traffic_weight_blue > 0) return 3;
    return 4;
  };

  return (
    <Container maxWidth="xl">
      <Box py={4}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
          <Box>
            <Typography variant="h4" fontWeight="bold">
              Blue/Green Deployments
            </Typography>
            <Typography variant="body2" color="text.secondary" mt={1}>
              Manage zero-downtime deployments with gradual traffic shifting
            </Typography>
          </Box>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {loading && <LinearProgress sx={{ mb: 3 }} />}

        <Grid container spacing={3}>
          {modules.map((module) => {
            const deployment = deployments[module.id];
            const weights = trafficWeights[module.id] || {
              blue: deployment?.traffic_weight_blue || 100,
              green: deployment?.traffic_weight_green || 0,
            };

            return (
              <Grid item xs={12} key={module.id}>
                <Card>
                  <CardContent>
                    <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
                      <Box>
                        <Typography variant="h6" fontWeight="bold">
                          {module.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {module.type} Module
                        </Typography>
                      </Box>
                      {!deployment && (
                        <Button
                          variant="contained"
                          startIcon={<AddIcon />}
                          onClick={() => handleOpenDialog(module.id)}
                        >
                          Start Deployment
                        </Button>
                      )}
                      {deployment && (
                        <Box display="flex" gap={1}>
                          <Chip
                            label={deployment.status}
                            color={getStatusColor(deployment.status) as any}
                            size="small"
                          />
                          {deployment.auto_rollback_enabled && (
                            <Chip label="Auto-Rollback" size="small" color="info" />
                          )}
                        </Box>
                      )}
                    </Box>

                    {!deployment ? (
                      <Box textAlign="center" py={4}>
                        <Typography variant="body2" color="text.secondary">
                          No active blue/green deployment
                        </Typography>
                      </Box>
                    ) : (
                      <Box>
                        <Stepper activeStep={getDeploymentStep(deployment)} sx={{ mb: 4 }}>
                          <Step>
                            <StepLabel>All Blue</StepLabel>
                          </Step>
                          <Step>
                            <StepLabel>Canary (75/25)</StepLabel>
                          </Step>
                          <Step>
                            <StepLabel>Split (50/50)</StepLabel>
                          </Step>
                          <Step>
                            <StepLabel>Majority Green (25/75)</StepLabel>
                          </Step>
                          <Step>
                            <StepLabel>All Green</StepLabel>
                          </Step>
                        </Stepper>

                        <Grid container spacing={3}>
                          <Grid item xs={12}>
                            <TrafficSlider
                              blueVersion={deployment.blue_version}
                              greenVersion={deployment.green_version}
                              initialBlueWeight={weights.blue}
                              initialGreenWeight={weights.green}
                              onWeightChange={(blue, green) =>
                                handleTrafficChange(module.id, blue, green)
                              }
                              disabled={savingTraffic === module.id}
                              showSaveButton={false}
                            />
                          </Grid>

                          <Grid item xs={12}>
                            <Divider />
                          </Grid>

                          <Grid item xs={12}>
                            <Box display="flex" gap={2} justifyContent="center">
                              <Button
                                variant="contained"
                                startIcon={<TimelineIcon />}
                                onClick={() => handleApplyTraffic(module.id, deployment.id)}
                                disabled={
                                  savingTraffic === module.id ||
                                  (weights.blue === deployment.traffic_weight_blue &&
                                    weights.green === deployment.traffic_weight_green)
                                }
                              >
                                Apply Traffic Changes
                              </Button>
                              <Button
                                variant="outlined"
                                color="error"
                                startIcon={<RollbackIcon />}
                                onClick={() => handleRollback(module.id, deployment.id)}
                                disabled={savingTraffic === module.id}
                              >
                                Rollback
                              </Button>
                              <Button
                                variant="outlined"
                                color="success"
                                startIcon={<FinalizeIcon />}
                                onClick={() =>
                                  handleFinalize(
                                    module.id,
                                    deployment.id,
                                    weights.green === 100 ? 'green' : 'blue'
                                  )
                                }
                                disabled={savingTraffic === module.id}
                              >
                                Finalize Deployment
                              </Button>
                            </Box>
                          </Grid>

                          {deployment.health_check_url && (
                            <Grid item xs={12}>
                              <Alert severity="info">
                                <Typography variant="caption">
                                  Health Check URL: {deployment.health_check_url}
                                </Typography>
                              </Alert>
                            </Grid>
                          )}
                        </Grid>
                      </Box>
                    )}
                  </CardContent>
                </Card>
              </Grid>
            );
          })}

          {modules.length === 0 && (
            <Grid item xs={12}>
              <Card>
                <CardContent>
                  <Typography variant="body2" color="text.secondary" textAlign="center" py={4}>
                    No enabled modules available for blue/green deployments
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          )}
        </Grid>
      </Box>

      {/* Create Deployment Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>Start Blue/Green Deployment</DialogTitle>
        <DialogContent>
          <Box pt={2}>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Blue Version (Current)"
                  value={formData.blue_version || ''}
                  onChange={(e) => setFormData({ ...formData, blue_version: e.target.value })}
                  required
                />
              </Grid>

              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Green Version (New)"
                  value={formData.green_version || ''}
                  onChange={(e) => setFormData({ ...formData, green_version: e.target.value })}
                  required
                />
              </Grid>

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Health Check URL"
                  value={formData.health_check_url || ''}
                  onChange={(e) => setFormData({ ...formData, health_check_url: e.target.value })}
                  helperText="URL to check health of green version"
                />
              </Grid>

              <Grid item xs={12}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={formData.auto_rollback_enabled ?? true}
                      onChange={(e) =>
                        setFormData({ ...formData, auto_rollback_enabled: e.target.checked })
                      }
                    />
                  }
                  label="Enable Automatic Rollback on Health Check Failure"
                />
              </Grid>

              <Grid item xs={12}>
                <Alert severity="info">
                  The deployment will start with 100% traffic to the blue version. You can gradually
                  shift traffic to the green version using the traffic slider.
                </Alert>
              </Grid>
            </Grid>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button variant="contained" onClick={handleCreateDeployment}>
            Start Deployment
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};

export default BlueGreen;
