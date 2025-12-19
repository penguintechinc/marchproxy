/**
 * Kong Configuration Upload with Validation and Preview
 */
import React, { useState, useRef } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Typography,
  Alert,
  CircularProgress,
  Divider,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  Paper,
  TextField,
} from '@mui/material';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import AddCircleIcon from '@mui/icons-material/AddCircle';
import RemoveCircleIcon from '@mui/icons-material/RemoveCircle';
import DownloadIcon from '@mui/icons-material/Download';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import { kongApi } from '../../services/kongApi';

interface ValidationResult {
  valid: boolean;
  error?: string;
  format_version?: string;
  stats?: {
    services: number;
    routes: number;
    upstreams: number;
    consumers: number;
    plugins: number;
    certificates: number;
  };
}

interface PreviewResult {
  services: { added: string[]; removed: string[]; unchanged: string[] };
  routes: { added: string[]; removed: string[]; unchanged: string[] };
  upstreams: { added: string[]; removed: string[]; unchanged: string[] };
  consumers: { added: string[]; removed: string[]; unchanged: string[] };
  plugins: { added: string[]; removed: string[]; unchanged: string[] };
}

const KongConfigUpload: React.FC = () => {
  const [yamlContent, setYamlContent] = useState('');
  const [validation, setValidation] = useState<ValidationResult | null>(null);
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const content = e.target?.result as string;
        setYamlContent(content);
        setValidation(null);
        setPreview(null);
        setError(null);
        setSuccess(null);
      };
      reader.readAsText(file);
    }
  };

  const handleValidate = async () => {
    if (!yamlContent.trim()) {
      setError('Please provide YAML configuration');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      // Simple client-side YAML validation
      const lines = yamlContent.split('\n');
      const hasFormatVersion = lines.some((line) => line.includes('_format_version'));

      if (!hasFormatVersion) {
        setValidation({ valid: false, error: 'Missing _format_version in configuration' });
        return;
      }

      // Count entities
      const countMatches = (pattern: RegExp) => {
        const matches = yamlContent.match(pattern);
        return matches ? matches.length : 0;
      };

      const stats = {
        services: countMatches(/^services:/gm) > 0 ? countMatches(/^\s+-\s+name:/gm) : 0,
        routes: countMatches(/^routes:/gm) > 0 ? countMatches(/^\s+-\s+name:/gm) : 0,
        upstreams: 0,
        consumers: 0,
        plugins: 0,
        certificates: 0,
      };

      setValidation({
        valid: true,
        format_version: '3.0',
        stats,
      });
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Validation failed';
      setValidation({ valid: false, error: errorMessage });
    } finally {
      setLoading(false);
    }
  };

  const handlePreview = async () => {
    if (!yamlContent.trim()) {
      setError('Please provide YAML configuration');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      // Get current Kong state
      const [servicesRes, routesRes, upstreamsRes, consumersRes, pluginsRes] = await Promise.all([
        kongApi.getServices(),
        kongApi.getRoutes(),
        kongApi.getUpstreams(),
        kongApi.getConsumers(),
        kongApi.getPlugins(),
      ]);

      const currentNames = {
        services: (servicesRes.data.data || []).map((s) => s.name),
        routes: (routesRes.data.data || []).map((r) => r.name),
        upstreams: (upstreamsRes.data.data || []).map((u) => u.name),
        consumers: (consumersRes.data.data || []).map((c) => c.username || c.custom_id || ''),
        plugins: (pluginsRes.data.data || []).map((p) => p.id),
      };

      // Parse new config names (simplified)
      const newNames = {
        services: [] as string[],
        routes: [] as string[],
        upstreams: [] as string[],
        consumers: [] as string[],
        plugins: [] as string[],
      };

      // Extract names from YAML (simplified parsing)
      const namePattern = /name:\s*['"]?([^'"\n]+)['"]?/g;
      let match;
      while ((match = namePattern.exec(yamlContent)) !== null) {
        newNames.services.push(match[1].trim());
      }

      const diff = (current: string[], newList: string[]) => ({
        added: newList.filter((n) => !current.includes(n)),
        removed: current.filter((n) => !newList.includes(n)),
        unchanged: current.filter((n) => newList.includes(n)),
      });

      setPreview({
        services: diff(currentNames.services, newNames.services),
        routes: diff(currentNames.routes, []),
        upstreams: diff(currentNames.upstreams, []),
        consumers: diff(currentNames.consumers, []),
        plugins: diff(currentNames.plugins, []),
      });
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Preview failed';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async () => {
    if (!yamlContent.trim()) {
      setError('Please provide YAML configuration');
      return;
    }

    if (!window.confirm('Are you sure you want to apply this configuration? This will update Kong.')) {
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(null);
    try {
      await kongApi.postConfig(yamlContent);
      setSuccess('Configuration applied successfully!');
      setPreview(null);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to apply configuration';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async () => {
    setLoading(true);
    try {
      const response = await kongApi.getConfig();
      const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `kong-config-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to export configuration';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const DiffSection: React.FC<{
    title: string;
    diff: { added: string[]; removed: string[]; unchanged: string[] };
  }> = ({ title, diff }) => (
    <Box mb={2}>
      <Typography variant="subtitle2" gutterBottom>
        {title}
      </Typography>
      <Box display="flex" gap={1} flexWrap="wrap">
        {diff.added.map((name) => (
          <Chip key={name} icon={<AddCircleIcon />} label={name} size="small" color="success" />
        ))}
        {diff.removed.map((name) => (
          <Chip key={name} icon={<RemoveCircleIcon />} label={name} size="small" color="error" />
        ))}
        {diff.unchanged.slice(0, 5).map((name) => (
          <Chip key={name} label={name} size="small" variant="outlined" />
        ))}
        {diff.unchanged.length > 5 && (
          <Chip label={`+${diff.unchanged.length - 5} more`} size="small" variant="outlined" />
        )}
      </Box>
    </Box>
  );

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        Kong Configuration Management
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

      {/* Action Buttons */}
      <Box display="flex" gap={2} mb={3}>
        <input
          type="file"
          accept=".yml,.yaml"
          hidden
          ref={fileInputRef}
          onChange={handleFileUpload}
        />
        <Button
          variant="outlined"
          startIcon={<CloudUploadIcon />}
          onClick={() => fileInputRef.current?.click()}
        >
          Upload YAML
        </Button>
        <Button variant="outlined" startIcon={<DownloadIcon />} onClick={handleExport}>
          Export Current Config
        </Button>
      </Box>

      {/* YAML Editor */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Configuration YAML
          </Typography>
          <TextField
            multiline
            rows={15}
            fullWidth
            placeholder={`_format_version: "3.0"

services:
  - name: my-service
    url: http://backend:8080
    routes:
      - name: my-route
        paths:
          - /api`}
            value={yamlContent}
            onChange={(e) => {
              setYamlContent(e.target.value);
              setValidation(null);
              setPreview(null);
            }}
            sx={{
              fontFamily: 'monospace',
              '& .MuiInputBase-input': { fontFamily: 'monospace', fontSize: '0.875rem' },
            }}
          />

          <Box display="flex" gap={2} mt={2}>
            <Button variant="outlined" onClick={handleValidate} disabled={loading || !yamlContent}>
              Validate
            </Button>
            <Button variant="outlined" onClick={handlePreview} disabled={loading || !yamlContent}>
              Preview Changes
            </Button>
            <Button
              variant="contained"
              color="primary"
              startIcon={<PlayArrowIcon />}
              onClick={handleApply}
              disabled={loading || !yamlContent}
            >
              Apply Configuration
            </Button>
            {loading && <CircularProgress size={24} />}
          </Box>
        </CardContent>
      </Card>

      {/* Validation Result */}
      {validation && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Box display="flex" alignItems="center" gap={1} mb={2}>
              <Typography variant="h6">Validation Result</Typography>
              {validation.valid ? (
                <Chip icon={<CheckCircleIcon />} label="Valid" color="success" size="small" />
              ) : (
                <Chip label="Invalid" color="error" size="small" />
              )}
            </Box>

            {validation.error && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {validation.error}
              </Alert>
            )}

            {validation.valid && validation.stats && (
              <Box display="flex" gap={2} flexWrap="wrap">
                <Chip label={`${validation.stats.services} Services`} />
                <Chip label={`${validation.stats.routes} Routes`} />
                <Chip label={`${validation.stats.upstreams} Upstreams`} />
                <Chip label={`${validation.stats.consumers} Consumers`} />
                <Chip label={`${validation.stats.plugins} Plugins`} />
              </Box>
            )}
          </CardContent>
        </Card>
      )}

      {/* Preview Result */}
      {preview && (
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Change Preview
            </Typography>
            <Typography variant="body2" color="textSecondary" gutterBottom>
              Review the changes before applying
            </Typography>
            <Divider sx={{ my: 2 }} />
            <DiffSection title="Services" diff={preview.services} />
            <DiffSection title="Routes" diff={preview.routes} />
            <DiffSection title="Upstreams" diff={preview.upstreams} />
            <DiffSection title="Consumers" diff={preview.consumers} />
          </CardContent>
        </Card>
      )}
    </Box>
  );
};

export default KongConfigUpload;
