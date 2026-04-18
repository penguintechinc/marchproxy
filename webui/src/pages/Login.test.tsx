import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Login from './Login';

// Mock the LoginPageBuilder to render a simple div for testing
vi.mock('@penguintechinc/react-libs', () => ({
  LoginPageBuilder: ({ branding }: any) => (
    <div data-testid="login-form">
      <h1>{branding?.appName || 'Login'}</h1>
      <input type="email" placeholder="Email" />
      <input type="password" placeholder="Password" />
      <button>Sign In</button>
    </div>
  ),
}));

vi.mock('@store/authStore', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    user: null,
  }),
}));

describe('Login', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders login page', () => {
    render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>
    );
    expect(screen.getByTestId('login-form')).toBeInTheDocument();
    expect(screen.getByText('MarchProxy')).toBeInTheDocument();
  });

  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('displays login form elements', () => {
    render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>
    );
    const buttons = screen.queryAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });
});
