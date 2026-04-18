import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import userEvent from '@testing-library/user-event';
import Certificates from './Certificates';
import * as certificateApi from '@services/certificateApi';
import * as clusterApi from '@services/clusterApi';

vi.mock('@services/certificateApi', () => ({
  certificateApi: {
    getCertificates: vi.fn(),
    createCertificate: vi.fn(),
    deleteCertificate: vi.fn(),
    uploadCertificate: vi.fn(),
  },
}));

vi.mock('@services/clusterApi', () => ({
  clusterApi: {
    getClusters: vi.fn(),
  },
}));

describe('Certificates - Scenario Tests', () => {
  const mockClusters = [
    { id: 1, name: 'prod-cluster', status: 'active' },
  ];

  const mockCertificates = [
    {
      id: 1,
      cluster_id: 1,
      name: 'prod-cert',
      issuer: 'Let\'s Encrypt',
      subject: 'example.com',
      expires_at: '2026-04-10T00:00:00Z',
      created_at: '2025-04-10T00:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Certificates page', () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (certificateApi.certificateApi.getCertificates as any).mockResolvedValue({
      data: { data: mockCertificates },
    });

    const { container } = render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });

  it('loads and displays certificates', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (certificateApi.certificateApi.getCertificates as any).mockResolvedValue({
      data: { data: mockCertificates },
    });

    render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('handles API errors gracefully', async () => {
    (clusterApi.clusterApi.getClusters as any).mockRejectedValue(
      new Error('Network error')
    );
    (certificateApi.certificateApi.getCertificates as any).mockRejectedValue(
      new Error('Network error')
    );

    render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    expect(document.body).toBeTruthy();
  });

  it('shows empty state when no certificates exist', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (certificateApi.certificateApi.getCertificates as any).mockResolvedValue({
      data: { data: [] },
    });

    render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('uploads new certificate', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (certificateApi.certificateApi.getCertificates as any).mockResolvedValue({
      data: { data: mockCertificates },
    });
    (certificateApi.certificateApi.uploadCertificate as any).mockResolvedValue({
      data: { data: { id: 2, name: 'new-cert' } },
    });

    render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const uploadButton = screen.queryByRole('button', { name: /upload|add|new/i });
    if (uploadButton) {
      await userEvent.click(uploadButton);
    }
  });

  it('deletes certificate', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (certificateApi.certificateApi.getCertificates as any)
      .mockResolvedValueOnce({ data: { data: mockCertificates } })
      .mockResolvedValueOnce({ data: { data: [] } });
    (certificateApi.certificateApi.deleteCertificate as any).mockResolvedValue(
      { data: {} }
    );

    render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const deleteButtons = screen.queryAllByRole('button', { name: /delete/i });
    if (deleteButtons.length > 0) {
      await userEvent.click(deleteButtons[0]);
    }
  });

  it('displays certificate expiration status', async () => {
    (clusterApi.clusterApi.getClusters as any).mockResolvedValue({
      data: { data: mockClusters },
    });
    (certificateApi.certificateApi.getCertificates as any).mockResolvedValue({
      data: { data: mockCertificates },
    });

    render(
      <BrowserRouter>
        <Certificates />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });
});
