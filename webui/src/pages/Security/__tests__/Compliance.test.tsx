import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Compliance from '../Compliance';

vi.mock('../../../services/securityApi');
vi.mock('../../../services/licenseApi', () => ({
  getLicenseStatus: vi.fn().mockResolvedValue({
    is_enterprise: true,
    tier: 'enterprise',
    features: ['audit_logs', 'compliance'],
    valid: true
  }),
  checkFeature: vi.fn().mockResolvedValue({ available: true }),
  refreshLicenseCache: vi.fn().mockResolvedValue({
    is_enterprise: true,
    tier: 'enterprise',
    features: ['audit_logs', 'compliance'],
    valid: true
  })
}));

describe('Compliance', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Compliance page', async () => {
    render(
      <BrowserRouter>
        <Compliance />
      </BrowserRouter>
    );
    await waitFor(() => {
      const elements = screen.queryAllByText(/Compliance|compliance|Report|Standards/i);
      expect(elements.length).toBeGreaterThan(0);
    }, { timeout: 2000 });
  });

  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <Compliance />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });
});
