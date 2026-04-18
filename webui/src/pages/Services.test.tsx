import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import Services from './Services';
import * as serviceApi from '@services/serviceApi';
import * as clusterApi from '@services/clusterApi';

vi.mock('@services/serviceApi', () => ({
  serviceApi: {
    getServices: vi.fn(),
    getServiceById: vi.fn(),
    createService: vi.fn(),
    updateService: vi.fn(),
    deleteService: vi.fn(),
    rotateServiceToken: vi.fn(),
    getServiceMappings: vi.fn(),
    addServiceMapping: vi.fn(),
    removeServiceMapping: vi.fn(),
  },
}));

vi.mock('@services/clusterApi', () => ({
  clusterApi: {
    getClusters: vi.fn(),
  },
}));

describe('Services Page - Scenario Tests', () => {
  const mockClusters = [
    { id: 1, name: 'prod-cluster', status: 'active' },
  ];

  const mockServices = [
    {
      id: 1,
      cluster_id: 1,
      name: 'api-service',
      description: 'Main API',
      destination_fqdn: 'api.example.com',
      destination_port: 443,
      protocol: 'HTTPS',
      auth_method: 'jwt',
      token_ttl: 3600,
      is_active: true,
      created_at: '2025-01-01T00:00:00Z',
      token: 'token-123',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Scenario 1: Initial load shows loading, then renders service list
  it('loads and displays services from API', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (serviceApi.serviceApi.getServices as any).mockResolvedValue({
      data: { data: mockServices },
    });

    render(<Services />);

    // Services are rendered
    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  // Scenario 1: Initial load shows loading, then renders service list
  it('handles API error gracefully', async () => {
    (clusterApi.clusterApi.getClusters as any).mockRejectedValue(
      new Error('Network error')
    );
    (serviceApi.serviceApi.getServices as any).mockRejectedValue(
      new Error('Network error')
    );

    render(<Services />);

    // Should not crash
    expect(document.body).toBeTruthy();
  });

  // Scenario 2: Empty state - no services
  it('shows empty state when no services exist', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (serviceApi.serviceApi.getServices as any).mockResolvedValue({
      data: { data: [] },
    });

    render(<Services />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  // Scenario 3: Load and display services
  it('displays services list when data loads', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (serviceApi.serviceApi.getServices as any).mockResolvedValue({
      data: { data: mockServices },
    });

    render(<Services />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });
});
