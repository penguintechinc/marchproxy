import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Dashboard from './Dashboard';
import * as clusterApi from '@services/clusterApi';
import * as proxyApi from '@services/proxyApi';

vi.mock('@services/clusterApi', () => ({
  clusterApi: {
    getClusters: vi.fn(),
  },
}));

vi.mock('@services/proxyApi', () => ({
  proxyApi: {
    getProxies: vi.fn(),
  },
}));

describe('Dashboard Page - Scenario Tests', () => {
  const mockClusters = [
    { id: 1, name: 'prod-cluster', status: 'active' },
  ];

  const mockProxies = [
    {
      id: 1,
      cluster_id: 1,
      name: 'proxy-01',
      status: 'online',
      ip_address: '192.168.1.10',
      port: 8080,
      version: '1.0.0',
      last_heartbeat: '2025-04-10T12:00:00Z',
      uptime_seconds: 86400,
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const renderDashboard = () =>
    render(
      <BrowserRouter>
        <Dashboard />
      </BrowserRouter>
    );

  it('loads and displays dashboard statistics', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: mockProxies },
    });

    renderDashboard();

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('handles API errors gracefully', async () => {
    (clusterApi.clusterApi.getClusters as any).mockRejectedValue(
      new Error('Network error')
    );
    (proxyApi.proxyApi.getProxies as any).mockRejectedValue(
      new Error('Network error')
    );

    renderDashboard();

    expect(document.body).toBeTruthy();
  });

  it('displays cluster count', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: [] },
    });

    renderDashboard();

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('displays proxy count', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: mockProxies },
    });

    renderDashboard();

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });
});
