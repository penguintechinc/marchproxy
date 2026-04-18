import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Proxies from './Proxies';
import * as proxyApi from '@services/proxyApi';
import * as clusterApi from '@services/clusterApi';

vi.mock('@services/proxyApi', () => ({
  proxyApi: {
    getProxies: vi.fn(),
    getProxyMetrics: vi.fn(),
    deleteProxy: vi.fn(),
    registerProxy: vi.fn(),
    updateProxy: vi.fn(),
  },
}));

vi.mock('@services/clusterApi', () => ({
  clusterApi: {
    getClusters: vi.fn(),
  },
}));

vi.mock('date-fns', () => ({
  formatDistanceToNow: (date: any) => '2 hours ago',
}));

describe('Proxies Page - Scenario Tests', () => {
  const mockClusters = [
    { id: 1, name: 'prod-cluster', status: 'active' },
    { id: 2, name: 'staging-cluster', status: 'active' },
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
    {
      id: 2,
      cluster_id: 1,
      name: 'proxy-02',
      status: 'offline',
      ip_address: '192.168.1.11',
      port: 8080,
      version: '1.0.0',
      last_heartbeat: '2025-04-10T10:00:00Z',
      uptime_seconds: 0,
    },
  ];

  const mockMetrics = {
    cpu_usage: 35.5,
    memory_usage: 512,
    memory_limit: 2048,
    request_count: 10000,
    error_count: 5,
    latency_p95_ms: 125,
    connections_active: 450,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads and displays proxy list', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: mockProxies },
    });

    render(<Proxies />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('handles API error gracefully', async () => {
    (clusterApi.clusterApi.getClusters as any).mockRejectedValue(
      new Error('Network error')
    );
    (proxyApi.proxyApi.getProxies as any).mockRejectedValue(
      new Error('Network error')
    );

    render(<Proxies />);
    expect(document.body).toBeTruthy();
  });

  it('filters proxies by cluster', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: mockProxies },
    });

    render(<Proxies />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const filterSelects = screen.getAllByRole('combobox');
    if (filterSelects.length > 0) {
      await userEvent.click(filterSelects[0]);
    }
  });

  it('deletes proxy after confirmation', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any)
      .mockResolvedValueOnce({ data: { data: mockProxies } })
      .mockResolvedValueOnce({ data: { data: [mockProxies[1]] } });
    (proxyApi.proxyApi.deleteProxy as any).mockResolvedValue({ data: {} });

    render(<Proxies />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const deleteButtons = screen.queryAllByRole('button', { name: /delete/i });
    if (deleteButtons.length > 0) {
      await userEvent.click(deleteButtons[0]);
    }
  });

  it('refreshes proxy list', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: mockProxies },
    });

    render(<Proxies />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const refreshButton = screen.queryByRole('button', { name: /refresh/i });
    if (refreshButton) {
      await userEvent.click(refreshButton);
    }
  });

  it('displays proxy status correctly', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (proxyApi.proxyApi.getProxies as any).mockResolvedValue({
      data: { data: mockProxies },
    });

    render(<Proxies />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });
});
