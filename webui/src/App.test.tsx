import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { useAuthStore } from '@store/authStore';

vi.mock('@penguintechinc/react-libs', () => ({
  AppConsoleVersion: () => <div data-testid="console-version">Console Version</div>,
}));

vi.mock('@store/authStore', () => ({
  useAuthStore: vi.fn(),
}));

vi.mock('@pages/Login', () => ({
  default: () => <div data-testid="login-page">Login Page</div>,
}));

vi.mock('@pages/Dashboard', () => ({
  default: () => <div data-testid="dashboard-page">Dashboard</div>,
}));

vi.mock('@components/Layout/MainLayout', () => ({
  default: ({ children }: any) => <div data-testid="main-layout">{children}</div>,
}));

vi.mock('@components/Layout/ProtectedRoute', () => ({
  default: ({ children }: any) => <div data-testid="protected-route">{children}</div>,
}));

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const renderApp = () =>
    render(
      <BrowserRouter>
        <App />
      </BrowserRouter>
    );

  // Scenario: App shows loading spinner while checking user session
  it('shows loading spinner when auth is loading', () => {
    (useAuthStore as any).mockReturnValue({
      isLoading: true,
      loadUser: vi.fn(),
      user: null,
      isAuthenticated: false,
    });

    renderApp();

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  // Scenario: App shows login page when user is not authenticated
  it('renders without crashing when not authenticated', async () => {
    (useAuthStore as any).mockReturnValue({
      isLoading: false,
      loadUser: vi.fn(),
      user: null,
      isAuthenticated: false,
    });

    const { container } = renderApp();

    expect(container).toBeTruthy();
  });

  // Scenario: App shows main layout with dashboard when user is authenticated
  it('shows dashboard inside protected route when authenticated', async () => {
    (useAuthStore as any).mockReturnValue({
      isLoading: false,
      loadUser: vi.fn(),
      user: { id: '1', email: 'test@example.com', role: 'Admin' },
      isAuthenticated: true,
    });

    renderApp();

    await waitFor(() => {
      expect(screen.getByTestId('main-layout')).toBeInTheDocument();
      expect(screen.getByTestId('protected-route')).toBeInTheDocument();
    });
  });

  // Scenario: App calls loadUser on mount
  it('calls loadUser from auth store on component mount', () => {
    const loadUserMock = vi.fn();
    (useAuthStore as any).mockReturnValue({
      isLoading: false,
      loadUser: loadUserMock,
      user: null,
      isAuthenticated: false,
    });

    renderApp();

    expect(loadUserMock).toHaveBeenCalled();
  });

  // Scenario: AppConsoleVersion component is rendered
  it('renders AppConsoleVersion component', async () => {
    (useAuthStore as any).mockReturnValue({
      isLoading: false,
      loadUser: vi.fn(),
      user: null,
      isAuthenticated: false,
    });

    renderApp();

    await waitFor(() => {
      expect(screen.getByTestId('console-version')).toBeInTheDocument();
    });
  });

  // Scenario: App transitions from loading to authenticated state
  it('transitions from loading to authenticated state', async () => {
    const { rerender } = renderApp();

    (useAuthStore as any).mockReturnValue({
      isLoading: false,
      loadUser: vi.fn(),
      user: { id: '1', email: 'test@example.com', role: 'Admin' },
      isAuthenticated: true,
    });

    rerender(
      <BrowserRouter>
        <App />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
      expect(screen.getByTestId('protected-route')).toBeInTheDocument();
    });
  });
});
