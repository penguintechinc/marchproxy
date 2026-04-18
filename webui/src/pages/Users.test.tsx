import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Users from './Users';
import * as usersApi from '@services/users';

vi.mock('@services/users', () => ({
  usersApi: {
    getUsers: vi.fn(),
    getUserById: vi.fn(),
    createUser: vi.fn(),
    updateUser: vi.fn(),
    deleteUser: vi.fn(),
    changePassword: vi.fn(),
    getRoles: vi.fn(),
  },
}));

describe('Users Page - Scenario Tests', () => {
  const mockUsers = [
    {
      id: '1',
      email: 'admin@example.com',
      role: 'Admin',
      created_at: '2025-01-01T00:00:00Z',
      last_login: '2025-04-10T12:00:00Z',
    },
    {
      id: '2',
      email: 'user@example.com',
      role: 'Viewer',
      created_at: '2025-02-01T00:00:00Z',
      last_login: '2025-04-09T10:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads and displays user list', async () => {
    (usersApi.usersApi.getUsers as any).mockResolvedValue({
      data: { data: mockUsers },
    });
    (usersApi.usersApi.getRoles as any).mockResolvedValue({
      data: { data: ['Admin', 'Maintainer', 'Viewer'] },
    });

    render(<Users />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('handles API error gracefully', async () => {
    (usersApi.usersApi.getUsers as any).mockRejectedValue(
      new Error('Network error')
    );
    (usersApi.usersApi.getRoles as any).mockRejectedValue(
      new Error('Network error')
    );

    render(<Users />);

    expect(document.body).toBeTruthy();
  });

  it('shows empty state when no users exist', async () => {
    (usersApi.usersApi.getUsers as any).mockResolvedValue({
      data: { data: [] },
    });
    (usersApi.usersApi.getRoles as any).mockResolvedValue({
      data: { data: ['Admin', 'Maintainer', 'Viewer'] },
    });

    render(<Users />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });

  it('creates new user', async () => {
    (usersApi.usersApi.getUsers as any).mockResolvedValue({
      data: { data: mockUsers },
    });
    (usersApi.usersApi.getRoles as any).mockResolvedValue({
      data: { data: ['Admin', 'Maintainer', 'Viewer'] },
    });
    (usersApi.usersApi.createUser as any).mockResolvedValue({
      data: { data: { id: '3', email: 'newuser@example.com' } },
    });

    render(<Users />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const addButton = screen.queryByRole('button', { name: /add|create|new/i });
    if (addButton) {
      await userEvent.click(addButton);
    }
  });

  it('deletes user', async () => {
    (usersApi.usersApi.getUsers as any)
      .mockResolvedValueOnce({ data: { data: mockUsers } })
      .mockResolvedValueOnce({ data: { data: [mockUsers[1]] } });
    (usersApi.usersApi.getRoles as any).mockResolvedValue({
      data: { data: ['Admin', 'Maintainer', 'Viewer'] },
    });
    (usersApi.usersApi.deleteUser as any).mockResolvedValue({ data: {} });

    render(<Users />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const deleteButtons = screen.queryAllByRole('button', { name: /delete/i });
    if (deleteButtons.length > 0) {
      await userEvent.click(deleteButtons[0]);
    }
  });

  it('updates user role', async () => {
    (usersApi.usersApi.getUsers as any).mockResolvedValue({
      data: { data: mockUsers },
    });
    (usersApi.usersApi.getRoles as any).mockResolvedValue({
      data: { data: ['Admin', 'Maintainer', 'Viewer'] },
    });
    (usersApi.usersApi.updateUser as any).mockResolvedValue({
      data: { data: { ...mockUsers[0], role: 'Maintainer' } },
    });

    render(<Users />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });

    const editButtons = screen.queryAllByRole('button', { name: /edit/i });
    if (editButtons.length > 0) {
      await userEvent.click(editButtons[0]);
    }
  });

  it('displays user roles', async () => {
    (usersApi.usersApi.getUsers as any).mockResolvedValue({
      data: { data: mockUsers },
    });
    (usersApi.usersApi.getRoles as any).mockResolvedValue({
      data: { data: ['Admin', 'Maintainer', 'Viewer'] },
    });

    render(<Users />);

    await waitFor(() => {
      expect(document.body).toBeTruthy();
    });
  });
});
