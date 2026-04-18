import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import MainLayout from '../MainLayout';

vi.mock('../Header', () => ({
  default: ({ isMobile, onMenuClick }: { isMobile: boolean; onMenuClick: () => void }) => (
    <div data-testid="header" data-mobile={isMobile} onClick={onMenuClick}>
      Header
    </div>
  ),
}));

vi.mock('../Sidebar', () => ({
  default: ({
    mobileOpen,
    onClose,
    isMobile,
  }: {
    mobileOpen: boolean;
    onClose: () => void;
    isMobile: boolean;
  }) => (
    <div data-testid="sidebar" data-mobile-open={mobileOpen} data-mobile={isMobile} onClick={onClose}>
      Sidebar
    </div>
  ),
}));

describe('MainLayout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Header and Sidebar', () => {
    render(
      <BrowserRouter>
        <MainLayout />
      </BrowserRouter>
    );

    expect(screen.getByTestId('header')).toBeInTheDocument();
    expect(screen.getByTestId('sidebar')).toBeInTheDocument();
  });

  it('renders main content area with Outlet', () => {
    render(
      <BrowserRouter>
        <MainLayout />
      </BrowserRouter>
    );

    const mainElement = screen.getByRole('main');
    expect(mainElement).toBeInTheDocument();
  });

  it('passes correct drawer width to Header and Sidebar', () => {
    render(
      <BrowserRouter>
        <MainLayout />
      </BrowserRouter>
    );

    expect(screen.getByTestId('header')).toBeInTheDocument();
    expect(screen.getByTestId('sidebar')).toBeInTheDocument();
  });
});
