/**
 * Admin Media Settings Page
 *
 * Super admin only page for configuring global media settings
 * including resolution caps, codec enforcement, and transcode ladders.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  Alert,
  AlertTitle,
  CircularProgress,
  FormControl,
  FormControlLabel,
  FormGroup,
  RadioGroup,
  Radio,
  Checkbox,
  Button,
  Divider,
  Tooltip,
  Chip,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import WarningIcon from '@mui/icons-material/Warning';
import SaveIcon from '@mui/icons-material/Save';
import RestoreIcon from '@mui/icons-material/Restore';
import {
  getAdminMediaSettings,
  updateAdminMediaSettings,
  resetAdminOverride,
  getAdminCapabilities,
  MediaSettings,
  HardwareCapabilities,
  ResolutionInfo,
  CodecInfo,
  getResolutionLabel,
} from '@services/mediaApi';
import { useAuthStore } from '@store/authStore';

const AdminMediaSettings: React.FC = () => {
  const { user } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const [settings, setSettings] = useState<MediaSettings | null>(null);
  const [hardware, setHardware] = useState<HardwareCapabilities | null>(null);
  const [resolutions, setResolutions] = useState<ResolutionInfo[]>([]);
  const [codecs, setCodecs] = useState<CodecInfo[]>([]);
  const [effectiveMax, setEffectiveMax] = useState<number>(1080);

  // Form state
  const [adminMax, setAdminMax] = useState<string>('hardware');
  const [ladderEnabled, setLadderEnabled] = useState(true);
  const [ladderResolutions, setLadderResolutions] = useState<number[]>([360, 540, 720, 1080]);

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      setLoading(true);
      const [settingsData, capsData] = await Promise.all([
        getAdminMediaSettings(),
        getAdminCapabilities(),
      ]);

      setSettings(settingsData.settings);
      setHardware(settingsData.hardware_capabilities);
      setEffectiveMax(settingsData.effective_max_resolution);
      setResolutions(capsData.resolutions);
      setCodecs(capsData.supported_codecs);

      // Initialize form state
      if (settingsData.settings.admin_max_resolution) {
        setAdminMax(settingsData.settings.admin_max_resolution.toString());
      } else {
        setAdminMax('hardware');
      }
      setLadderEnabled(settingsData.settings.transcode_ladder_enabled);
      setLadderResolutions(settingsData.settings.transcode_ladder_resolutions);

      setError(null);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to load settings';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      setError(null);
      setSuccess(null);

      const newSettings = {
        admin_max_resolution: adminMax === 'hardware' ? null : parseInt(adminMax),
        transcode_ladder_enabled: ladderEnabled,
        transcode_ladder_resolutions: ladderResolutions,
      };

      await updateAdminMediaSettings(newSettings);
      setSuccess('Settings saved successfully');
      await fetchSettings();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to save settings';
      setError(message);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = async () => {
    if (!confirm('Reset resolution limit to hardware default?')) return;

    try {
      setSaving(true);
      setError(null);
      setSuccess(null);

      await resetAdminOverride();
      setSuccess('Settings reset to hardware default');
      await fetchSettings();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to reset settings';
      setError(message);
    } finally {
      setSaving(false);
    }
  };

  const handleLadderChange = (resolution: number, checked: boolean) => {
    if (checked) {
      setLadderResolutions((prev) => [...prev, resolution].sort((a, b) => a - b));
    } else {
      setLadderResolutions((prev) => prev.filter((r) => r !== resolution));
    }
  };

  const getDisabledReason = (res: ResolutionInfo): string => {
    if (res.supported) return '';
    return res.disabled_reason || 'Not available';
  };

  if (user?.role !== 'administrator') {
    return (
      <Alert severity="error">
        <AlertTitle>Access Denied</AlertTitle>
        This page is only accessible to super administrators.
      </Alert>
    );
  }

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" fontWeight="bold" mb={3}>
        Media Settings
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

      {/* Hardware Info */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" mb={2}>
            Hardware Capabilities
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={6} md={3}>
              <Typography variant="body2" color="text.secondary">
                GPU Type
              </Typography>
              <Typography variant="body1">
                {hardware?.gpu_type !== 'none' ? hardware?.gpu_type?.toUpperCase() : 'None (CPU)'}
              </Typography>
            </Grid>
            <Grid item xs={6} md={3}>
              <Typography variant="body2" color="text.secondary">
                GPU Model
              </Typography>
              <Typography variant="body1">{hardware?.gpu_model || 'N/A'}</Typography>
            </Grid>
            <Grid item xs={6} md={3}>
              <Typography variant="body2" color="text.secondary">
                VRAM
              </Typography>
              <Typography variant="body1">{hardware?.vram_gb || 0} GB</Typography>
            </Grid>
            <Grid item xs={6} md={3}>
              <Typography variant="body2" color="text.secondary">
                Hardware Max
              </Typography>
              <Typography variant="body1">
                {hardware ? getResolutionLabel(hardware.hardware_max_resolution) : '-'}
              </Typography>
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />

          <Box display="flex" gap={1} flexWrap="wrap">
            {codecs.map((codec) => (
              <Chip
                key={codec.id}
                label={`${codec.name} ${codec.hardware_accelerated ? '(HW)' : '(SW)'}`}
                color={codec.supported ? 'success' : 'default'}
                variant={codec.supported ? 'filled' : 'outlined'}
              />
            ))}
          </Box>
        </CardContent>
      </Card>

      {/* Resolution Limit */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" mb={2}>
            Global Resolution Limit
          </Typography>

          <Alert severity="warning" sx={{ mb: 2 }} icon={<WarningIcon />}>
            <AlertTitle>Applies to ALL Streams</AlertTitle>
            This setting enforces a maximum resolution across the entire system. Streams exceeding
            this limit will be rejected.
          </Alert>

          <FormControl component="fieldset">
            <RadioGroup value={adminMax} onChange={(e) => setAdminMax(e.target.value)}>
              <FormControlLabel
                value="hardware"
                control={<Radio />}
                label={
                  <Box display="flex" alignItems="center" gap={1}>
                    Use hardware default ({hardware ? getResolutionLabel(hardware.hardware_max_resolution) : '-'})
                    <Chip label="Recommended" size="small" color="primary" />
                  </Box>
                }
              />

              {resolutions.map((res) => (
                <Tooltip
                  key={res.height}
                  title={getDisabledReason(res)}
                  placement="right"
                  disableHoverListener={res.supported}
                >
                  <span>
                    <FormControlLabel
                      value={res.height.toString()}
                      control={<Radio />}
                      disabled={!res.supported && res.height > (hardware?.hardware_max_resolution || 1080)}
                      label={
                        <Box display="flex" alignItems="center" gap={1}>
                          {res.label}
                          {res.requires_gpu && <Chip label="GPU" size="small" />}
                          {!res.supported && <InfoIcon fontSize="small" color="disabled" />}
                        </Box>
                      }
                    />
                  </span>
                </Tooltip>
              ))}
            </RadioGroup>
          </FormControl>
        </CardContent>
      </Card>

      {/* Transcode Ladder */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" mb={2}>
            Transcode Ladder
          </Typography>

          <Alert severity="info" sx={{ mb: 2 }} icon={<InfoIcon />}>
            Pre-transcode all incoming streams to these resolutions. This enables instant quality
            switching for viewers and restreaming to platforms like Twitch/YouTube.
          </Alert>

          <FormControlLabel
            control={
              <Checkbox
                checked={ladderEnabled}
                onChange={(e) => setLadderEnabled(e.target.checked)}
              />
            }
            label="Enable transcode ladder"
          />

          {ladderEnabled && (
            <FormGroup sx={{ ml: 3, mt: 1 }}>
              {[360, 480, 540, 720, 1080, 1440, 2160].map((height) => {
                const bitrates: Record<number, string> = {
                  360: '0.8 Mbps',
                  480: '1.5 Mbps',
                  540: '2 Mbps',
                  720: '3 Mbps',
                  1080: '5 Mbps',
                  1440: '8 Mbps',
                  2160: '20 Mbps',
                };
                const isDisabled = height > effectiveMax;

                return (
                  <Tooltip
                    key={height}
                    title={isDisabled ? `Exceeds current limit (${getResolutionLabel(effectiveMax)})` : ''}
                    placement="right"
                    disableHoverListener={!isDisabled}
                  >
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={ladderResolutions.includes(height)}
                          onChange={(e) => handleLadderChange(height, e.target.checked)}
                          disabled={isDisabled}
                        />
                      }
                      label={`${getResolutionLabel(height)} (${bitrates[height]})`}
                    />
                  </Tooltip>
                );
              })}
            </FormGroup>
          )}
        </CardContent>
      </Card>

      {/* Current Effective Settings */}
      <Alert severity="info" sx={{ mb: 3 }}>
        <AlertTitle>Current Effective Limits</AlertTitle>
        <Typography variant="body2">
          <strong>Hardware capability:</strong> {hardware ? getResolutionLabel(hardware.hardware_max_resolution) : '-'}
          {' | '}
          <strong>Admin limit:</strong> {settings?.admin_max_resolution ? getResolutionLabel(settings.admin_max_resolution) : 'None'}
          {' | '}
          <strong>Effective maximum:</strong> {getResolutionLabel(effectiveMax)}
        </Typography>
      </Alert>

      {/* Action Buttons */}
      <Box display="flex" gap={2}>
        <Button
          variant="contained"
          startIcon={<SaveIcon />}
          onClick={handleSave}
          disabled={saving}
        >
          {saving ? 'Saving...' : 'Save Settings'}
        </Button>
        <Button
          variant="outlined"
          startIcon={<RestoreIcon />}
          onClick={handleReset}
          disabled={saving}
        >
          Reset to Hardware Default
        </Button>
      </Box>
    </Box>
  );
};

export default AdminMediaSettings;
