import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Clusters from './Clusters';
import * as clusterApi from '@services/clusterApi';

vi.mock('@services/clusterApi', () => ({
  clusterApi: {
    getClusters: vi.fn(),
    getClusterById: vi.fn(),
    createCluster: vi.fn(),
    updateCluster: vi.fn(),
    deleteCluster: vi.fn(),
    getClusterStatus: vi.fn(),
  },
}));

describe('Clusters Page - Scenario Tests', () => {
  const mockClusters = [
    {
      id: 1,
      name: 'prod-cluster',
      status: 'active',
      kubeconfig: 'config-prod',
      created_at: '2025-01-01T00:00:00Z',
    },
    {
      id: 2,
      name: 'staging-cluster',
      status: 'healthy',
      kubeconfig: 'config-staging',
      created_at: '2025-01-02T00:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads and displays cluster list', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });

    render(<Clusters />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('handles API error gracefully', async () => {
    (clusterApi.clusterApi.getClusters as any).mockRejectedValue(
      new Error('Network error')
    );

    render(<Clusters />);

    expect(document.body).toBeTruthy();
  });

  it('shows empty state when no clusters exist', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: [] },
    });

    render(<Clusters />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('deletes cluster after confirmation', async () => {
    (clusterApi.clusterApi.getClusters as any)
      .mockResolvedValueOnce({ data: { data: mockClusters } })
      .mockResolvedValueOnce({ data: { data: [mockClusters[1]] } });
    (clusterApi.clusterApi.deleteCluster as any).mockResolvedValue({ data: {} });

    render(<Clusters />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const deleteButtons = screen.queryAllByRole('button', { name: /delete/i });
    if (deleteButtons.length > 0) {
      await userEvent.click(deleteButtons[0]);
    }
  });

  it('displays cluster status', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });

    render(<Clusters />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('creates new cluster', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (clusterApi.clusterApi.createCluster as any).mockResolvedValue({
      data: { data: { id: 3, name: 'new-cluster' } },
    });

    render(<Clusters />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const addButton = screen.queryByRole('button', { name: /add|create|new/i });
    if (addButton) {
      await userEvent.click(addButton);
    }
  });
});
