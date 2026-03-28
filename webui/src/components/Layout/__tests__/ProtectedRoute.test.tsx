/**
 * Tests for ProtectedRoute component
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';
import { MemoryRouter } from 'react-router-dom';

vi.mock('@store/authStore', () => ({
  useAuthStore: vi.fn(),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    Navigate: vi.fn(({ to }: { to: string }) => (
      <div data-testid="navigate" data-to={to} />
    )),
  };
});

import ProtectedRoute from '../ProtectedRoute';
import { useAuthStore } from '@store/authStore';

const mockUseAuthStore = useAuthStore as ReturnType<typeof vi.fn>;

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders children when user is authenticated', () => {
    mockUseAuthStore.mockReturnValue({ isAuthenticated: true });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div data-testid="protected-content">Protected Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.getByTestId('protected-content')).toBeDefined();
    expect(screen.queryByTestId('navigate')).toBeNull();
  });

  it('renders Navigate to /login when user is not authenticated', () => {
    mockUseAuthStore.mockReturnValue({ isAuthenticated: false });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div data-testid="protected-content">Protected Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    const navigateEl = screen.getByTestId('navigate');
    expect(navigateEl).toBeDefined();
    expect(navigateEl.getAttribute('data-to')).toBe('/login');
  });

  it('does not render children when user is not authenticated', () => {
    mockUseAuthStore.mockReturnValue({ isAuthenticated: false });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div data-testid="protected-content">Protected Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.queryByTestId('protected-content')).toBeNull();
  });

  it('renders multiple children when authenticated', () => {
    mockUseAuthStore.mockReturnValue({ isAuthenticated: true });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div data-testid="child-1">Child 1</div>
          <div data-testid="child-2">Child 2</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.getByTestId('child-1')).toBeDefined();
    expect(screen.getByTestId('child-2')).toBeDefined();
  });
});
