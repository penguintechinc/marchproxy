import React, { useEffect, useState } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  Storage as ClusterIcon,
  Dns as ServiceIcon,
  Router as ProxyIcon,
  CheckCircle as ActiveIcon,
} from '@mui/icons-material';
import { apiClient } from '@services/api';
import { DashboardStats } from '@services/types';

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ReactElement;
  color: string;
}

const StatCard: React.FC<StatCardProps> = ({ title, value, icon, color }) => (
  <Card>
    <CardContent>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
        <Box
          sx={{
            bgcolor: color,
            borderRadius: 2,
            p: 1,
            mr: 2,
            display: 'flex',
          }}
        >
          {React.cloneElement(icon, { sx: { color: 'white', fontSize: 32 } })}
        </Box>
        <Box sx={{ flexGrow: 1 }}>
          <Typography variant="h4" component="div" fontWeight="bold">
            {value}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {title}
          </Typography>
        </Box>
      </Box>
    </CardContent>
  </Card>
);

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchDashboardStats();
  }, []);

  const fetchDashboardStats = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get<DashboardStats>(
        '/api/dashboard/stats'
      );
      setStats(response.data);
      setError(null);
    } catch (err: any) {
      setError(
        err.response?.data?.detail || 'Failed to load dashboard statistics'
      );
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="400px"
      >
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom fontWeight="bold">
        Dashboard
      </Typography>
      <Typography variant="body1" color="text.secondary" gutterBottom mb={3}>
        Overview of your MarchProxy deployment
      </Typography>

      <Grid container spacing={3}>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Total Proxies"
            value={stats?.total_proxies || 0}
            icon={<ProxyIcon />}
            color="#1E3A8A"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Active Proxies"
            value={stats?.active_proxies || 0}
            icon={<ActiveIcon />}
            color="#10B981"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Total Services"
            value={stats?.total_services || 0}
            icon={<ServiceIcon />}
            color="#FFD700"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Total Clusters"
            value={stats?.total_clusters || 0}
            icon={<ClusterIcon />}
            color="#F59E0B"
          />
        </Grid>
      </Grid>

      <Box mt={4}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              License Information
            </Typography>
            <Box sx={{ display: 'flex', gap: 2, mt: 2 }}>
              <Box>
                <Typography variant="body2" color="text.secondary">
                  Tier
                </Typography>
                <Typography variant="h6" sx={{ textTransform: 'capitalize' }}>
                  {stats?.license_tier || 'Community'}
                </Typography>
              </Box>
              <Box>
                <Typography variant="body2" color="text.secondary">
                  Status
                </Typography>
                <Typography
                  variant="h6"
                  color={stats?.license_valid ? 'success.main' : 'error.main'}
                >
                  {stats?.license_valid ? 'Valid' : 'Invalid'}
                </Typography>
              </Box>
            </Box>
          </CardContent>
        </Card>
      </Box>
    </Box>
  );
};

export default Dashboard;
