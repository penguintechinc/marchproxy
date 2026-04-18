import { describe, it, expect } from 'vitest';
import type {
  User,
  LoginRequest,
  LoginResponse,
  Cluster,
  DashboardStats,
} from '../types';

describe('TypeScript Type Definitions', () => {
  it('should allow User object creation', () => {
    const user: User = {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
      role: 'administrator',
      is_active: true,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    };

    expect(user.id).toBe(1);
    expect(user.username).toBe('testuser');
    expect(user.role).toBe('administrator');
  });

  it('should allow LoginRequest object creation', () => {
    const request: LoginRequest = {
      username: 'testuser',
      password: 'password123',
      totp_code: '123456',
    };

    expect(request.username).toBe('testuser');
    expect(request.password).toBe('password123');
    expect(request.totp_code).toBe('123456');
  });

  it('should allow LoginResponse object creation', () => {
    const response: LoginResponse = {
      user_id: 1,
      username: 'testuser',
      email: 'test@example.com',
      is_admin: true,
      requires_2fa: false,
      access_token: 'token123',
      token_type: 'Bearer',
      expires_in: 3600,
    };

    expect(response.user_id).toBe(1);
    expect(response.is_admin).toBe(true);
    expect(response.access_token).toBe('token123');
  });

  it('should allow Cluster object creation', () => {
    const cluster: Cluster = {
      id: 1,
      name: 'Test Cluster',
      description: 'Test cluster description',
      api_key_hash: 'hash123',
      log_auth: true,
      log_netflow: false,
      log_debug: false,
    };

    expect(cluster.id).toBe(1);
    expect(cluster.name).toBe('Test Cluster');
    expect(cluster.log_auth).toBe(true);
  });

  it('should allow DashboardStats object creation', () => {
    const stats: DashboardStats = {
      totalClusters: 5,
      totalServices: 10,
      totalProxies: 20,
      activeConnections: 1000,
      totalDataTransferred: '10.5GB',
      uptime: '99.99%',
    };

    expect(stats.totalClusters).toBe(5);
    expect(stats.totalServices).toBe(10);
    expect(stats.uptime).toBe('99.99%');
  });
});
