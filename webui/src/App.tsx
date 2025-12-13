import React, { useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Box, CircularProgress } from '@mui/material';
import { useAuthStore } from '@store/authStore';
import MainLayout from '@components/Layout/MainLayout';
import ProtectedRoute from '@components/Layout/ProtectedRoute';
import Login from '@pages/Login';
import Dashboard from '@pages/Dashboard';
import Clusters from '@pages/Clusters';
import Services from '@pages/Services';
import Proxies from '@pages/Proxies';
import Certificates from '@pages/Certificates';
import Settings from '@pages/Settings';
import Tracing from '@pages/Observability/Tracing';
import Metrics from '@pages/Observability/Metrics';
import Alerts from '@pages/Observability/Alerts';

const App: React.FC = () => {
  const { isLoading, loadUser } = useAuthStore();

  useEffect(() => {
    loadUser();
  }, [loadUser]);

  if (isLoading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="100vh"
        bgcolor="background.default"
      >
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="clusters" element={<Clusters />} />
        <Route path="services" element={<Services />} />
        <Route path="proxies" element={<Proxies />} />
        <Route path="certificates" element={<Certificates />} />
        <Route path="observability/tracing" element={<Tracing />} />
        <Route path="observability/metrics" element={<Metrics />} />
        <Route path="observability/alerts" element={<Alerts />} />
        <Route path="settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};

export default App;
