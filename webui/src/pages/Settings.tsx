import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Alert,
  Switch,
  FormControlLabel,
  Divider,
  Tabs,
  Tab,
  Grid,
  Chip,
} from '@mui/material';
import {
  Person as PersonIcon,
  Lock as LockIcon,
  Security as SecurityIcon,
  BusinessCenter as LicenseIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { apiClient } from '@services/api';
import { License } from '@services/types';
import { useAuthStore } from '@store/authStore';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
  <div hidden={value !== index}>
    {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
  </div>
);

interface ProfileFormData {
  username: string;
  email: string;
}

interface PasswordFormData {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

interface SystemSettingsData {
  syslog_server: string;
  syslog_port: number;
  license_key: string;
}

const Settings: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [twoFAEnabled, setTwoFAEnabled] = useState(false);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [license, setLicense] = useState<License | null>(null);
  const user = useAuthStore((state) => state.user);

  const profileForm = useForm<ProfileFormData>({
    defaultValues: {
      username: user?.username || '',
      email: user?.email || '',
    }
  });

  const passwordForm = useForm<PasswordFormData>({
    defaultValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
    }
  });

  const systemForm = useForm<SystemSettingsData>({
    defaultValues: {
      syslog_server: '',
      syslog_port: 514,
      license_key: '',
    }
  });

  useEffect(() => {
    fetchUserSettings();
    fetchLicense();
  }, []);

  const fetchUserSettings = async () => {
    try {
      const response = await apiClient.get('/api/user/settings');
      setTwoFAEnabled(response.data.totp_enabled);
      profileForm.reset({
        username: response.data.username,
        email: response.data.email,
      });
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load settings');
    }
  };

  const fetchLicense = async () => {
    try {
      const response = await apiClient.get<License>('/api/license');
      setLicense(response.data);
    } catch (err: any) {
      console.error('Failed to load license:', err);
    }
  };

  const onProfileSubmit = async (data: ProfileFormData) => {
    try {
      await apiClient.put('/api/user/profile', data);
      setSuccess('Profile updated successfully');
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to update profile');
    }
  };

  const onPasswordSubmit = async (data: PasswordFormData) => {
    if (data.new_password !== data.confirm_password) {
      setError('Passwords do not match');
      return;
    }

    try {
      await apiClient.put('/api/user/password', {
        current_password: data.current_password,
        new_password: data.new_password,
      });
      setSuccess('Password changed successfully');
      setError(null);
      passwordForm.reset();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to change password');
    }
  };

  const handleEnable2FA = async () => {
    try {
      const response = await apiClient.post('/api/user/2fa/enable');
      setQrCode(response.data.qr_code);
      setTwoFAEnabled(true);
      setSuccess('2FA enabled. Scan the QR code with your authenticator app.');
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to enable 2FA');
    }
  };

  const handleDisable2FA = async () => {
    try {
      await apiClient.post('/api/user/2fa/disable');
      setTwoFAEnabled(false);
      setQrCode(null);
      setSuccess('2FA disabled successfully');
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to disable 2FA');
    }
  };

  const onSystemSettingsSubmit = async (data: SystemSettingsData) => {
    try {
      await apiClient.put('/api/settings/system', data);
      setSuccess('System settings updated successfully');
      setError(null);
      fetchLicense();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to update system settings');
    }
  };

  return (
    <Box>
      <Typography variant="h4" fontWeight="bold" mb={3}>
        Settings
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {success && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      )}

      <Card>
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)}>
          <Tab icon={<PersonIcon />} label="Profile" />
          <Tab icon={<LockIcon />} label="Password" />
          <Tab icon={<SecurityIcon />} label="Security" />
          <Tab icon={<LicenseIcon />} label="License" />
          {user?.role === 'administrator' && (
            <Tab icon={<LicenseIcon />} label="System" />
          )}
        </Tabs>

        <CardContent>
          {/* Profile Tab */}
          <TabPanel value={tabValue} index={0}>
            <form onSubmit={profileForm.handleSubmit(onProfileSubmit)}>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, maxWidth: 500 }}>
                <Controller
                  name="username"
                  control={profileForm.control}
                  rules={{ required: 'Username is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Username"
                      fullWidth
                      disabled
                      error={!!profileForm.formState.errors.username}
                      helperText={profileForm.formState.errors.username?.message}
                    />
                  )}
                />
                <Controller
                  name="email"
                  control={profileForm.control}
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
                      error={!!profileForm.formState.errors.email}
                      helperText={profileForm.formState.errors.email?.message}
                    />
                  )}
                />
                <Box>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    Role
                  </Typography>
                  <Chip
                    label={user?.role || 'N/A'}
                    color="primary"
                    sx={{ textTransform: 'capitalize' }}
                  />
                </Box>
                <Button type="submit" variant="contained" sx={{ alignSelf: 'flex-start' }}>
                  Update Profile
                </Button>
              </Box>
            </form>
          </TabPanel>

          {/* Password Tab */}
          <TabPanel value={tabValue} index={1}>
            <form onSubmit={passwordForm.handleSubmit(onPasswordSubmit)}>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, maxWidth: 500 }}>
                <Controller
                  name="current_password"
                  control={passwordForm.control}
                  rules={{ required: 'Current password is required' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Current Password"
                      type="password"
                      fullWidth
                      error={!!passwordForm.formState.errors.current_password}
                      helperText={passwordForm.formState.errors.current_password?.message}
                    />
                  )}
                />
                <Controller
                  name="new_password"
                  control={passwordForm.control}
                  rules={{
                    required: 'New password is required',
                    minLength: { value: 8, message: 'Password must be at least 8 characters' }
                  }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="New Password"
                      type="password"
                      fullWidth
                      error={!!passwordForm.formState.errors.new_password}
                      helperText={passwordForm.formState.errors.new_password?.message}
                    />
                  )}
                />
                <Controller
                  name="confirm_password"
                  control={passwordForm.control}
                  rules={{ required: 'Please confirm your password' }}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Confirm New Password"
                      type="password"
                      fullWidth
                      error={!!passwordForm.formState.errors.confirm_password}
                      helperText={passwordForm.formState.errors.confirm_password?.message}
                    />
                  )}
                />
                <Button type="submit" variant="contained" sx={{ alignSelf: 'flex-start' }}>
                  Change Password
                </Button>
              </Box>
            </form>
          </TabPanel>

          {/* Security Tab */}
          <TabPanel value={tabValue} index={2}>
            <Box sx={{ maxWidth: 600 }}>
              <Typography variant="h6" gutterBottom>
                Two-Factor Authentication (2FA)
              </Typography>
              <Typography variant="body2" color="text.secondary" paragraph>
                Add an extra layer of security to your account by enabling two-factor authentication.
              </Typography>
              <FormControlLabel
                control={
                  <Switch
                    checked={twoFAEnabled}
                    onChange={(e) => e.target.checked ? handleEnable2FA() : handleDisable2FA()}
                  />
                }
                label={twoFAEnabled ? '2FA Enabled' : '2FA Disabled'}
              />
              {qrCode && (
                <Box sx={{ mt: 3, p: 2, bgcolor: 'background.default', borderRadius: 1 }}>
                  <Typography variant="body2" gutterBottom>
                    Scan this QR code with your authenticator app:
                  </Typography>
                  <Box
                    component="img"
                    src={`data:image/png;base64,${qrCode}`}
                    alt="2FA QR Code"
                    sx={{ maxWidth: 200, mt: 2 }}
                  />
                </Box>
              )}
            </Box>
          </TabPanel>

          {/* License Tab */}
          <TabPanel value={tabValue} index={3}>
            <Box sx={{ maxWidth: 600 }}>
              <Typography variant="h6" gutterBottom>
                License Information
              </Typography>
              {license ? (
                <Grid container spacing={2} sx={{ mt: 1 }}>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="body2" color="text.secondary">
                      License Key
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>
                      {license.key}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="body2" color="text.secondary">
                      Tier
                    </Typography>
                    <Chip
                      label={license.tier}
                      color={license.tier === 'enterprise' ? 'success' : 'default'}
                      sx={{ textTransform: 'capitalize', mt: 0.5 }}
                    />
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="body2" color="text.secondary">
                      Max Proxies
                    </Typography>
                    <Typography variant="body1">
                      {license.max_proxies === -1 ? 'Unlimited' : license.max_proxies}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="body2" color="text.secondary">
                      Valid Until
                    </Typography>
                    <Typography variant="body1">
                      {new Date(license.valid_until).toLocaleDateString()}
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary" gutterBottom>
                      Features
                    </Typography>
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                      {license.features.map((feature) => (
                        <Chip key={feature} label={feature} size="small" variant="outlined" />
                      ))}
                    </Box>
                  </Grid>
                </Grid>
              ) : (
                <Alert severity="warning" sx={{ mt: 2 }}>
                  No license information available
                </Alert>
              )}
            </Box>
          </TabPanel>

          {/* System Settings Tab (Admin only) */}
          {user?.role === 'administrator' && (
            <TabPanel value={tabValue} index={4}>
              <form onSubmit={systemForm.handleSubmit(onSystemSettingsSubmit)}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, maxWidth: 600 }}>
                  <Typography variant="h6" gutterBottom>
                    System Settings
                  </Typography>
                  <Divider />
                  <Controller
                    name="syslog_server"
                    control={systemForm.control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Default Syslog Server"
                        fullWidth
                        placeholder="syslog.example.com"
                      />
                    )}
                  />
                  <Controller
                    name="syslog_port"
                    control={systemForm.control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Default Syslog Port"
                        type="number"
                        fullWidth
                      />
                    )}
                  />
                  <Divider />
                  <Controller
                    name="license_key"
                    control={systemForm.control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="License Key"
                        fullWidth
                        placeholder="PENG-XXXX-XXXX-XXXX-XXXX-ABCD"
                        helperText="Enter a new license key to update the system license"
                      />
                    )}
                  />
                  <Button type="submit" variant="contained" sx={{ alignSelf: 'flex-start' }}>
                    Update System Settings
                  </Button>
                </Box>
              </form>
            </TabPanel>
          )}
        </CardContent>
      </Card>
    </Box>
  );
};

export default Settings;
