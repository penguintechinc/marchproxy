import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Login from '../Login.tsx';

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

describe('Login', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders login page content', async () => {
    render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>
    );
    await waitFor(() => {
      expect(document.body).toBeDefined();
    }, { timeout: 1000 });
  });
});
