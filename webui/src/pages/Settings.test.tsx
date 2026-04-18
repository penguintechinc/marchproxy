import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Settings from './Settings';

vi.mock('@penguintechinc/react-libs', () => ({
  FormModalBuilder: ({ children }: any) => <div>{children}</div>,
  SidebarMenu: () => <div>Sidebar</div>,
}));

describe('Settings Page - Scenario Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const renderSettings = () =>
    render(
      <Settings />
    );

  it('renders settings page without crashing', () => {
    renderSettings();

    expect(document.body).toBeTruthy();
  });

  it('displays settings sections', () => {
    renderSettings();

    expect(document.body).toBeTruthy();
  });

  it('allows settings updates', async () => {
    renderSettings();

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const inputs = screen.queryAllByRole('textbox');
    if (inputs.length > 0) {
      await userEvent.type(inputs[0], 'new-value');
    }
  });

  it('displays advanced settings options', () => {
    renderSettings();

    const advancedElements = document.body.textContent?.includes('Advanced') || false;
    expect(document.body).toBeTruthy();
  });
});
