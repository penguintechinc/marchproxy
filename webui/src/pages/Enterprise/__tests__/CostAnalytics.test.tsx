import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import CostAnalytics from '../CostAnalytics.tsx';

// Mock all services
vi.mock('../../../services/api');
vi.mock('../../../services/kongApi');
vi.mock('../../../services/enterpriseApi');
vi.mock('../../../services/clusterApi');
vi.mock('../../../services/certificateApi');
vi.mock('../../../services/securityApi');
vi.mock('../../../services/mediaApi');
vi.mock('../../../services/observabilityApi');
vi.mock('../../../services/modulesApi');
vi.mock('../../../services/licenseApi');
vi.mock('../../../services/users');
vi.mock('../../../services/proxyApi');
vi.mock('../../../services/serviceApi');

describe('CostAnalytics', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <CostAnalytics />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('contains DOM elements', () => {
    const { container } = render(
      <BrowserRouter>
        <CostAnalytics />
      </BrowserRouter>
    );
    expect(container.innerHTML.length).toBeGreaterThan(0);
  });
});
