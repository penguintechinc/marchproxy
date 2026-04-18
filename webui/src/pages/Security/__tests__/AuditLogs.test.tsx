import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import AuditLogs from '../AuditLogs';

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

describe('AuditLogs', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Audit Logs page', async () => {
    render(
      <BrowserRouter>
        <AuditLogs />
      </BrowserRouter>
    );
    await waitFor(() => {
      const elements = screen.queryAllByText(/Audit|audit|Log|Export|Refresh/i);
      expect(elements.length).toBeGreaterThan(0);
    }, { timeout: 2000 });
  });

  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <AuditLogs />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });
});
