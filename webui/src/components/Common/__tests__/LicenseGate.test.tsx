import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import LicenseGate from '../LicenseGate';

describe('LicenseGate', () => {
  beforeEach(() => {
    // Cleanup if needed
  });

  it('renders children when user has access', () => {
    render(
      <LicenseGate hasAccess={true} featureName="Advanced Analytics">
        <div data-testid="protected-content">Protected Content</div>
      </LicenseGate>
    );

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
  });

  it('shows loading state when isLoading is true', () => {
    render(
      <LicenseGate hasAccess={false} featureName="Advanced Analytics" isLoading={true}>
        <div data-testid="protected-content">Protected Content</div>
      </LicenseGate>
    );

    expect(screen.getByText(/Checking license/i)).toBeInTheDocument();
  });

  it('shows upgrade prompt when user has no access', () => {
    render(
      <LicenseGate hasAccess={false} featureName="Advanced Analytics">
        <div data-testid="protected-content">Protected Content</div>
      </LicenseGate>
    );

    expect(screen.getByText(/Enterprise Feature/i)).toBeInTheDocument();
    expect(screen.getByText(/Advanced Analytics/i)).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('displays upgrade features list', () => {
    render(
      <LicenseGate hasAccess={false} featureName="Advanced Analytics">
        <div>Content</div>
      </LicenseGate>
    );

    expect(screen.getByText(/Advanced Traffic Shaping/i)).toBeInTheDocument();
    expect(screen.getByText(/Multi-Cloud Intelligent Routing/i)).toBeInTheDocument();
    expect(screen.getByText(/Zero-Trust Security Policies/i)).toBeInTheDocument();
  });

  it('renders upgrade button with correct href', () => {
    render(
      <LicenseGate
        hasAccess={false}
        featureName="Advanced Analytics"
        upgradeUrl="https://example.com/pricing"
      >
        <div>Content</div>
      </LicenseGate>
    );

    const upgradeButton = screen.getByRole('link', { name: /Upgrade to Enterprise/i });
    expect(upgradeButton).toHaveAttribute('href', 'https://example.com/pricing');
  });

  it('renders settings link for activating existing license', () => {
    render(
      <LicenseGate hasAccess={false} featureName="Advanced Analytics">
        <div>Content</div>
      </LicenseGate>
    );

    const settingsLink = screen.getByRole('link', { name: /Activate it here/i });
    expect(settingsLink).toHaveAttribute('href', '/settings/license');
  });

  it('does not render protected content when no access', () => {
    render(
      <LicenseGate hasAccess={false} featureName="Advanced Analytics">
        <div data-testid="protected-content">Protected Content</div>
      </LicenseGate>
    );

    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });
});
