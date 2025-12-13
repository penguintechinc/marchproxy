/**
 * LicenseGate Component
 *
 * Wraps enterprise features and shows an upgrade prompt for Community users.
 * Gracefully degrades for users without enterprise licenses.
 */

import React from 'react';
import { Button, Box, Typography, Paper } from '@mui/material';
import LockIcon from '@mui/icons-material/Lock';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunch';

interface LicenseGateProps {
  /** Child components to render if license is valid */
  children: React.ReactNode;
  /** Feature name for display */
  featureName: string;
  /** Whether the user has access to this feature */
  hasAccess: boolean;
  /** Loading state while checking license */
  isLoading?: boolean;
  /** Custom upgrade URL */
  upgradeUrl?: string;
}

const LicenseGate: React.FC<LicenseGateProps> = ({
  children,
  featureName,
  hasAccess,
  isLoading = false,
  upgradeUrl = 'https://www.penguintech.io/marchproxy/pricing'
}) => {
  // Show loading state
  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <Typography variant="body1" color="text.secondary">
          Checking license...
        </Typography>
      </Box>
    );
  }

  // If user has access, render children
  if (hasAccess) {
    return <>{children}</>;
  }

  // Show upgrade prompt for Community users
  return (
    <Box
      display="flex"
      justifyContent="center"
      alignItems="center"
      minHeight="400px"
      p={4}
    >
      <Paper
        elevation={3}
        sx={{
          maxWidth: 600,
          p: 4,
          textAlign: 'center',
          background: 'linear-gradient(135deg, #1E3A8A 0%, #0F172A 100%)',
          color: 'white'
        }}
      >
        <LockIcon sx={{ fontSize: 64, mb: 2, color: '#FFD700' }} />

        <Typography variant="h4" gutterBottom fontWeight="bold">
          Enterprise Feature
        </Typography>

        <Typography variant="h6" gutterBottom color="#FFD700">
          {featureName}
        </Typography>

        <Typography variant="body1" paragraph sx={{ mt: 3, mb: 3 }}>
          This feature is available with <strong>MarchProxy Enterprise</strong>.
          Upgrade your license to unlock advanced capabilities including:
        </Typography>

        <Box sx={{ textAlign: 'left', mb: 3, mx: 'auto', maxWidth: 400 }}>
          <Typography variant="body2" paragraph>
            ✓ Advanced Traffic Shaping & QoS
          </Typography>
          <Typography variant="body2" paragraph>
            ✓ Multi-Cloud Intelligent Routing
          </Typography>
          <Typography variant="body2" paragraph>
            ✓ Distributed Tracing & Observability
          </Typography>
          <Typography variant="body2" paragraph>
            ✓ Zero-Trust Security Policies
          </Typography>
          <Typography variant="body2" paragraph>
            ✓ Unlimited Proxy Servers
          </Typography>
          <Typography variant="body2" paragraph>
            ✓ SAML/SCIM/OAuth2 Authentication
          </Typography>
        </Box>

        <Button
          variant="contained"
          size="large"
          startIcon={<RocketLaunchIcon />}
          href={upgradeUrl}
          target="_blank"
          rel="noopener noreferrer"
          sx={{
            mt: 2,
            bgcolor: '#FFD700',
            color: '#1E1E1E',
            fontWeight: 'bold',
            '&:hover': {
              bgcolor: '#FDB813',
            }
          }}
        >
          Upgrade to Enterprise
        </Button>

        <Typography variant="caption" display="block" sx={{ mt: 3, opacity: 0.8 }}>
          Already have an Enterprise license?{' '}
          <a
            href="/settings/license"
            style={{ color: '#FFD700', textDecoration: 'underline' }}
          >
            Activate it here
          </a>
        </Typography>
      </Paper>
    </Box>
  );
};

export default LicenseGate;
