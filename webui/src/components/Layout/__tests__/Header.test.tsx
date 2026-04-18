import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Header from '../Header';

vi.mock('@store/authStore', () => ({
  useAuthStore: vi.fn(() => ({
    user: null,
    logout: vi.fn(),
  })),
}));

describe('Header', () => {
  const mockOnMenuClick = vi.fn();

  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <Header drawerWidth={240} onMenuClick={mockOnMenuClick} isMobile={false} />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });

  it('renders on mobile view', () => {
    const { container } = render(
      <BrowserRouter>
        <Header drawerWidth={240} onMenuClick={mockOnMenuClick} isMobile={true} />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });
});
