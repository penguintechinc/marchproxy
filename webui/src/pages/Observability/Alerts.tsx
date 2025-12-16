/**
 * Alerts Configuration Page
 *
 * Alert rule management, active alerts monitoring, and notification configuration.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Grid,
  Chip,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert,
  CircularProgress,
  Tabs,
  Tab,
  Card,
  CardContent,
  CardActions,
  Switch,
  FormControlLabel,
  Tooltip,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningIcon from '@mui/icons-material/Warning';
import ErrorIcon from '@mui/icons-material/Error';
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import NotificationsOffIcon from '@mui/icons-material/NotificationsOff';
import {
  getAlerts,
  getActiveAlerts,
  createAlert,
  updateAlert,
  deleteAlert,
  acknowledgeAlert,
  silenceAlert,
} from '../../services/observabilityApi';
import {
  AlertRule,
  Alert as AlertType,
  CreateAlertRequest,
} from '../../services/observabilityTypes';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => {
  return (
    <div hidden={value !== index}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
};

export const Alerts: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [alertRules, setAlertRules] = useState<AlertRule[]>([]);
  const [activeAlerts, setActiveAlerts] = useState<AlertType[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);

  // Form state
  const [formData, setFormData] = useState<CreateAlertRequest>({
    name: '',
    description: '',
    enabled: true,
    severity: 'warning',
    query: '',
    threshold: 0,
    operator: '>',
    duration: '5m',
    notification_channel_ids: [],
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      const [rules, active] = await Promise.all([getAlerts(), getActiveAlerts()]);
      setAlertRules(rules);
      setActiveAlerts(active);
    } catch (err: any) {
      setError(err.message || 'Failed to load alerts');
      console.error('Error loading alerts:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenDialog = (rule?: AlertRule) => {
    if (rule) {
      setEditingRule(rule);
      setFormData({
        name: rule.name,
        description: rule.description,
        enabled: rule.enabled,
        severity: rule.severity,
        query: rule.query,
        threshold: rule.threshold,
        operator: rule.operator,
        duration: rule.duration,
        labels: rule.labels,
        annotations: rule.annotations,
        notification_channel_ids: rule.notification_channels.map((nc) => nc.id),
      });
    } else {
      setEditingRule(null);
      setFormData({
        name: '',
        description: '',
        enabled: true,
        severity: 'warning',
        query: '',
        threshold: 0,
        operator: '>',
        duration: '5m',
        notification_channel_ids: [],
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingRule(null);
  };

  const handleSaveRule = async () => {
    try {
      if (editingRule) {
        await updateAlert(editingRule.id, formData);
      } else {
        await createAlert(formData);
      }
      await loadData();
      handleCloseDialog();
    } catch (err: any) {
      setError(err.message || 'Failed to save alert rule');
      console.error('Error saving alert rule:', err);
    }
  };

  const handleDeleteRule = async (id: number) => {
    if (!confirm('Are you sure you want to delete this alert rule?')) return;

    try {
      await deleteAlert(id);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to delete alert rule');
      console.error('Error deleting alert rule:', err);
    }
  };

  const handleToggleRule = async (rule: AlertRule) => {
    try {
      await updateAlert(rule.id, { enabled: !rule.enabled });
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to toggle alert rule');
      console.error('Error toggling alert rule:', err);
    }
  };

  const handleAcknowledgeAlert = async (id: number) => {
    try {
      await acknowledgeAlert(id);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to acknowledge alert');
      console.error('Error acknowledging alert:', err);
    }
  };

  const handleSilenceAlert = async (id: number) => {
    const duration = prompt('Silence duration in minutes:', '60');
    if (!duration) return;

    try {
      await silenceAlert(id, parseInt(duration) * 60);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to silence alert');
      console.error('Error silencing alert:', err);
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <ErrorIcon color="error" />;
      case 'warning':
        return <WarningIcon color="warning" />;
      case 'info':
        return <CheckCircleIcon color="info" />;
      default:
        return null;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'error';
      case 'warning':
        return 'warning';
      case 'info':
        return 'info';
      default:
        return 'default';
    }
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">Alerts</Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => handleOpenDialog()}
        >
          New Alert Rule
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)} sx={{ mb: 2 }}>
        <Tab
          label={
            <Box display="flex" alignItems="center" gap={1}>
              Active Alerts
              {activeAlerts.length > 0 && (
                <Chip
                  label={activeAlerts.length}
                  size="small"
                  color="error"
                  sx={{ height: 20 }}
                />
              )}
            </Box>
          }
        />
        <Tab label="Alert Rules" />
      </Tabs>

      {loading && (
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      )}

      <TabPanel value={tabValue} index={0}>
        {activeAlerts.length === 0 ? (
          <Alert severity="success">No active alerts</Alert>
        ) : (
          <Grid container spacing={2}>
            {activeAlerts.map((alert) => (
              <Grid item xs={12} md={6} key={alert.id}>
                <Card>
                  <CardContent>
                    <Box display="flex" alignItems="center" gap={1} mb={1}>
                      {getSeverityIcon(alert.severity)}
                      <Typography variant="h6">{alert.rule_name}</Typography>
                      <Chip
                        label={alert.severity}
                        size="small"
                        color={getSeverityColor(alert.severity) as any}
                      />
                      <Chip
                        label={alert.status}
                        size="small"
                        variant="outlined"
                      />
                    </Box>
                    <Typography variant="body2" color="text.secondary" gutterBottom>
                      {alert.message}
                    </Typography>
                    <Typography variant="caption" display="block">
                      Started: {new Date(alert.started_at).toLocaleString()}
                    </Typography>
                    <Typography variant="caption" display="block">
                      Value: {alert.value.toFixed(2)}
                    </Typography>
                  </CardContent>
                  <CardActions>
                    <Button
                      size="small"
                      onClick={() => handleAcknowledgeAlert(alert.id)}
                      disabled={alert.status === 'acknowledged'}
                    >
                      Acknowledge
                    </Button>
                    <Button
                      size="small"
                      onClick={() => handleSilenceAlert(alert.id)}
                      disabled={alert.status === 'silenced'}
                    >
                      Silence
                    </Button>
                  </CardActions>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Severity</TableCell>
                <TableCell>Query</TableCell>
                <TableCell>Threshold</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {alertRules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell>
                    <Box>
                      <Typography variant="body2" fontWeight="bold">
                        {rule.name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {rule.description}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={rule.severity}
                      size="small"
                      color={getSeverityColor(rule.severity) as any}
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption" fontFamily="monospace">
                      {rule.query}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {rule.operator} {rule.threshold}
                  </TableCell>
                  <TableCell>
                    <Tooltip title={rule.enabled ? 'Enabled' : 'Disabled'}>
                      <IconButton
                        size="small"
                        onClick={() => handleToggleRule(rule)}
                      >
                        {rule.enabled ? (
                          <NotificationsActiveIcon color="primary" />
                        ) : (
                          <NotificationsOffIcon color="disabled" />
                        )}
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => handleOpenDialog(rule)}
                    >
                      <EditIcon />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleDeleteRule(rule.id)}
                      color="error"
                    >
                      <DeleteIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </TabPanel>

      {/* Alert Rule Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>
          {editingRule ? 'Edit Alert Rule' : 'New Alert Rule'}
        </DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12} md={8}>
              <TextField
                fullWidth
                label="Name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                required
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <FormControl fullWidth>
                <InputLabel>Severity</InputLabel>
                <Select
                  value={formData.severity}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      severity: e.target.value as any,
                    })
                  }
                  label="Severity"
                >
                  <MenuItem value="info">Info</MenuItem>
                  <MenuItem value="warning">Warning</MenuItem>
                  <MenuItem value="critical">Critical</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                multiline
                rows={2}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="PromQL Query"
                value={formData.query}
                onChange={(e) =>
                  setFormData({ ...formData, query: e.target.value })
                }
                multiline
                rows={3}
                required
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <FormControl fullWidth>
                <InputLabel>Operator</InputLabel>
                <Select
                  value={formData.operator}
                  onChange={(e) =>
                    setFormData({ ...formData, operator: e.target.value as any })
                  }
                  label="Operator"
                >
                  <MenuItem value=">">Greater Than (&gt;)</MenuItem>
                  <MenuItem value="<">Less Than (&lt;)</MenuItem>
                  <MenuItem value=">=">Greater or Equal (&gt;=)</MenuItem>
                  <MenuItem value="<=">Less or Equal (&lt;=)</MenuItem>
                  <MenuItem value="=">Equal (=)</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                label="Threshold"
                type="number"
                value={formData.threshold}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    threshold: parseFloat(e.target.value),
                  })
                }
                required
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                label="Duration"
                value={formData.duration}
                onChange={(e) =>
                  setFormData({ ...formData, duration: e.target.value })
                }
                placeholder="e.g., 5m"
                required
              />
            </Grid>
            <Grid item xs={12}>
              <FormControlLabel
                control={
                  <Switch
                    checked={formData.enabled}
                    onChange={(e) =>
                      setFormData({ ...formData, enabled: e.target.checked })
                    }
                  />
                }
                label="Enabled"
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSaveRule} variant="contained">
            {editingRule ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Alerts;
