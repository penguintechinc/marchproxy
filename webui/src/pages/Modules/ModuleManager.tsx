/**
 * ModuleManager Page
 *
 * Main page for managing MarchProxy NLB modules (NLB, ALB, DBLB, AILB, RTMP).
 * Provides enable/disable controls, status overview, and quick navigation.
 */

import React, { useState, useEffect } from 'react';
import {
  Container,
  Typography,
  Box,
  Grid,
  Alert,
  Button,
  Card,
  CardContent,
  LinearProgress,
  Tabs,
  Tab,
} from '@mui/material';
import { useNavigate } from 'react-router-dom';
import RefreshIcon from '@mui/icons-material/Refresh';
import ModuleCard from '../../components/Modules/ModuleCard';
import LicenseGate from '../../components/Common/LicenseGate';
import { useLicense } from '../../hooks/useLicense';
import {
  getModules,
  enableModule,
  disableModule,
  getModuleMetrics,
} from '../../services/modulesApi';
import type { Module, ModuleMetrics } from '../../services/types';

const ModuleManager: React.FC = () => {
  const navigate = useNavigate();
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('unified_nlb');

  const [modules, setModules] = useState<Module[]>([]);
  const [metrics, setMetrics] = useState<Record<number, ModuleMetrics>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState(0);
  const [refreshing, setRefreshing] = useState<number | null>(null);

  useEffect(() => {
    loadModules();
    // Poll for metrics every 10 seconds
    const interval = setInterval(() => {
      if (!loading) {
        loadMetrics();
      }
    }, 10000);
    return () => clearInterval(interval);
  }, []);

  const loadModules = async () => {
    try {
      setLoading(true);
      const data = await getModules();
      setModules(data);
      await loadMetrics(data);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load modules');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadMetrics = async (moduleList?: Module[]) => {
    const activeModules = (moduleList || modules).filter((m) => m.is_enabled);
    const metricsPromises = activeModules.map(async (module) => {
      try {
        const data = await getModuleMetrics(module.id);
        return { id: module.id, data };
      } catch {
        return { id: module.id, data: null };
      }
    });

    const results = await Promise.all(metricsPromises);
    const metricsMap: Record<number, ModuleMetrics> = {};
    results.forEach((result) => {
      if (result.data) {
        metricsMap[result.id] = result.data;
      }
    });
    setMetrics(metricsMap);
  };

  const handleEnable = async (id: number) => {
    try {
      setRefreshing(id);
      await enableModule(id);
      await loadModules();
    } catch (err: any) {
      setError(err.message || 'Failed to enable module');
    } finally {
      setRefreshing(null);
    }
  };

  const handleDisable = async (id: number) => {
    if (!confirm('Are you sure you want to disable this module?')) return;
    try {
      setRefreshing(id);
      await disableModule(id);
      await loadModules();
    } catch (err: any) {
      setError(err.message || 'Failed to disable module');
    } finally {
      setRefreshing(null);
    }
  };

  const handleConfigure = (id: number) => {
    navigate(`/modules/${id}/routes`);
  };

  const handleViewMetrics = (id: number) => {
    navigate(`/modules/${id}/metrics`);
  };

  const filterModules = (type?: Module['type']) => {
    if (!type) return modules;
    return modules.filter((m) => m.type === type);
  };

  const getTabLabel = (type: Module['type']) => {
    const count = modules.filter((m) => m.type === type && m.is_enabled).length;
    const total = modules.filter((m) => m.type === type).length;
    return `${type} (${count}/${total})`;
  };

  const renderModuleGrid = (moduleList: Module[]) => {
    if (moduleList.length === 0) {
      return (
        <Card>
          <CardContent>
            <Typography variant="body2" color="text.secondary" textAlign="center" py={4}>
              No modules available in this category
            </Typography>
          </CardContent>
        </Card>
      );
    }

    return (
      <Grid container spacing={3}>
        {moduleList.map((module) => (
          <Grid item xs={12} md={6} lg={4} key={module.id}>
            <ModuleCard
              module={module}
              metrics={metrics[module.id]}
              onEnable={handleEnable}
              onDisable={handleDisable}
              onConfigure={handleConfigure}
              onViewMetrics={handleViewMetrics}
              loading={refreshing === module.id}
            />
          </Grid>
        ))}
      </Grid>
    );
  };

  return (
    <LicenseGate
      featureName="Unified NLB Module Management"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Container maxWidth="xl">
        <Box py={4}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
            <Box>
              <Typography variant="h4" fontWeight="bold">
                Module Management
              </Typography>
              <Typography variant="body2" color="text.secondary" mt={1}>
                Manage NLB, ALB, DBLB, AILB, and RTMP modules for unified load balancing
              </Typography>
            </Box>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={loadModules}
              disabled={loading}
            >
              Refresh
            </Button>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          {loading && <LinearProgress sx={{ mb: 3 }} />}

          <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
            <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
              <Tab label="All Modules" />
              <Tab label={getTabLabel('NLB')} />
              <Tab label={getTabLabel('ALB')} />
              <Tab label={getTabLabel('DBLB')} />
              <Tab label={getTabLabel('AILB')} />
              <Tab label={getTabLabel('RTMP')} />
            </Tabs>
          </Box>

          {activeTab === 0 && renderModuleGrid(modules)}
          {activeTab === 1 && renderModuleGrid(filterModules('NLB'))}
          {activeTab === 2 && renderModuleGrid(filterModules('ALB'))}
          {activeTab === 3 && renderModuleGrid(filterModules('DBLB'))}
          {activeTab === 4 && renderModuleGrid(filterModules('AILB'))}
          {activeTab === 5 && renderModuleGrid(filterModules('RTMP'))}

          <Box mt={4}>
            <Card sx={{ bgcolor: 'info.main', color: 'white' }}>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  About Unified NLB Architecture
                </Typography>
                <Typography variant="body2">
                  MarchProxy's unified NLB architecture provides modular load balancing capabilities.
                  Each module can be independently enabled/disabled and supports multiple routes,
                  rate limiting, auto-scaling, and blue/green deployments.
                </Typography>
                <Box mt={2}>
                  <Typography variant="body2" component="div">
                    <strong>NLB:</strong> Layer 3/4 network load balancer with XDP/eBPF acceleration
                  </Typography>
                  <Typography variant="body2" component="div">
                    <strong>ALB:</strong> Layer 7 HTTP/HTTPS application load balancer (Envoy)
                  </Typography>
                  <Typography variant="body2" component="div">
                    <strong>DBLB:</strong> Database load balancer with connection pooling (ArticDBM)
                  </Typography>
                  <Typography variant="body2" component="div">
                    <strong>AILB:</strong> AI/LLM inference load balancer (WaddleAI integration)
                  </Typography>
                  <Typography variant="body2" component="div">
                    <strong>RTMP:</strong> Video streaming transcoder with FFmpeg (x264/x265)
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Box>
        </Box>
      </Container>
    </LicenseGate>
  );
};

export default ModuleManager;
