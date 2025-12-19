/**
 * Kong Gateway Management - Main Router
 */
import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Box, Tabs, Tab, Paper } from '@mui/material';
import { useNavigate, useLocation } from 'react-router-dom';
import DashboardIcon from '@mui/icons-material/Dashboard';
import DnsIcon from '@mui/icons-material/Dns';
import RouteIcon from '@mui/icons-material/AltRoute';
import StorageIcon from '@mui/icons-material/Storage';
import PeopleIcon from '@mui/icons-material/People';
import ExtensionIcon from '@mui/icons-material/Extension';
import SecurityIcon from '@mui/icons-material/Security';
import UploadFileIcon from '@mui/icons-material/UploadFile';

import KongDashboard from './KongDashboard';
import KongServices from './KongServices';
import KongRoutes from './KongRoutes';
import KongUpstreams from './KongUpstreams';
import KongConsumers from './KongConsumers';
import KongPlugins from './KongPlugins';
import KongCertificates from './KongCertificates';
import KongConfigUpload from './KongConfigUpload';

const tabs = [
  { label: 'Dashboard', path: 'dashboard', icon: <DashboardIcon /> },
  { label: 'Services', path: 'services', icon: <DnsIcon /> },
  { label: 'Routes', path: 'routes', icon: <RouteIcon /> },
  { label: 'Upstreams', path: 'upstreams', icon: <StorageIcon /> },
  { label: 'Consumers', path: 'consumers', icon: <PeopleIcon /> },
  { label: 'Plugins', path: 'plugins', icon: <ExtensionIcon /> },
  { label: 'Certificates', path: 'certificates', icon: <SecurityIcon /> },
  { label: 'Config', path: 'config', icon: <UploadFileIcon /> },
];

const KongIndex: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // Determine current tab from URL
  const currentPath = location.pathname.split('/').pop() || 'dashboard';
  const currentTab = tabs.findIndex((t) => t.path === currentPath);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    navigate(`/kong/${tabs[newValue].path}`);
  };

  return (
    <Box sx={{ width: '100%' }}>
      <Paper sx={{ mb: 2 }}>
        <Tabs
          value={currentTab >= 0 ? currentTab : 0}
          onChange={handleTabChange}
          variant="scrollable"
          scrollButtons="auto"
          sx={{
            '& .MuiTab-root': {
              minHeight: 64,
              textTransform: 'none',
            },
          }}
        >
          {tabs.map((tab) => (
            <Tab
              key={tab.path}
              icon={tab.icon}
              label={tab.label}
              iconPosition="start"
            />
          ))}
        </Tabs>
      </Paper>

      <Routes>
        <Route path="dashboard" element={<KongDashboard />} />
        <Route path="services" element={<KongServices />} />
        <Route path="routes" element={<KongRoutes />} />
        <Route path="upstreams" element={<KongUpstreams />} />
        <Route path="consumers" element={<KongConsumers />} />
        <Route path="plugins" element={<KongPlugins />} />
        <Route path="certificates" element={<KongCertificates />} />
        <Route path="config" element={<KongConfigUpload />} />
        <Route path="*" element={<Navigate to="dashboard" replace />} />
      </Routes>
    </Box>
  );
};

export default KongIndex;
