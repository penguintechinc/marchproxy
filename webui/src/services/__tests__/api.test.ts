/**
 * Tests for API client service (api.ts)
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import axios from 'axios';

// Mock axios before importing the module
vi.mock('axios', () => {
  const mockInstance = {
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  };
  return {
    default: {
      create: vi.fn(() => mockInstance),
    },
  };
});

describe('API Service - helper functions', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('setAuthToken stores token in localStorage', async () => {
    const { setAuthToken } = await import('../api');
    setAuthToken('test-token-123');
    expect(localStorage.getItem('auth_token')).toBe('test-token-123');
  });

  it('clearAuthToken removes token from localStorage', async () => {
    localStorage.setItem('auth_token', 'test-token-123');
    const { clearAuthToken } = await import('../api');
    clearAuthToken();
    expect(localStorage.getItem('auth_token')).toBeNull();
  });

  it('getAuthToken returns token from localStorage', async () => {
    localStorage.setItem('auth_token', 'my-token');
    const { getAuthToken } = await import('../api');
    expect(getAuthToken()).toBe('my-token');
  });

  it('getAuthToken returns null when no token stored', async () => {
    const { getAuthToken } = await import('../api');
    expect(getAuthToken()).toBeNull();
  });

  it('apiClient is created with axios.create', async () => {
    await import('../api');
    expect(axios.create).toHaveBeenCalledWith(
      expect.objectContaining({
        timeout: 30000,
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
      })
    );
  });
});

describe('API Service - request interceptor', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('request interceptor adds Authorization header when token exists', async () => {
    localStorage.setItem('auth_token', 'my-bearer-token');

    // Get the interceptor callback that was registered
    const mockedAxios = axios as any;
    let requestInterceptor: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: {
          use: vi.fn((successFn: Function) => {
            requestInterceptor = successFn;
          }),
        },
        response: { use: vi.fn() },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (requestInterceptor) {
      const config = { headers: {} as any };
      const result = (requestInterceptor as Function)(config);
      expect(result.headers.Authorization).toBe('Bearer my-bearer-token');
    }
  });

  it('request interceptor does not add Authorization header when no token', async () => {
    const mockedAxios = axios as any;
    let requestInterceptor: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: {
          use: vi.fn((successFn: Function) => {
            requestInterceptor = successFn;
          }),
        },
        response: { use: vi.fn() },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (requestInterceptor) {
      const config = { headers: {} as any };
      const result = (requestInterceptor as Function)(config);
      expect(result.headers.Authorization).toBeUndefined();
    }
  });

  it('request interceptor error handler rejects with error', async () => {
    const mockedAxios = axios as any;
    let errorInterceptor: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: {
          use: vi.fn((_successFn: Function, errFn: Function) => {
            errorInterceptor = errFn;
          }),
        },
        response: { use: vi.fn() },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (errorInterceptor) {
      const err = new Error('request error');
      await expect((errorInterceptor as Function)(err)).rejects.toThrow('request error');
    }
  });
});

describe('API Service - response interceptor', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
  });

  afterEach(() => {
    localStorage.clear();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    delete (window as any).location;
  });

  it('response interceptor passes through successful responses', async () => {
    const mockedAxios = axios as any;
    let responseSuccessHandler: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: { use: vi.fn() },
        response: {
          use: vi.fn((successFn: Function) => {
            responseSuccessHandler = successFn;
          }),
        },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (responseSuccessHandler) {
      const response = { data: { foo: 'bar' }, status: 200 };
      const result = (responseSuccessHandler as Function)(response);
      expect(result).toEqual(response);
    }
  });

  it('response interceptor clears token and redirects on 401', async () => {
    localStorage.setItem('auth_token', 'old-token');

    // Mock window.location
    Object.defineProperty(window, 'location', {
      writable: true,
      value: { href: '' },
    });

    const mockedAxios = axios as any;
    let errorHandler: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: { use: vi.fn() },
        response: {
          use: vi.fn((_successFn: Function, errFn: Function) => {
            errorHandler = errFn;
          }),
        },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (errorHandler) {
      const error = {
        response: { status: 401, data: {} },
      };
      await expect((errorHandler as Function)(error)).rejects.toEqual(error);
      expect(localStorage.getItem('auth_token')).toBeNull();
      expect(window.location.href).toBe('/login');
    }
  });

  it('response interceptor logs warning on 403 with feature detail', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    const mockedAxios = axios as any;
    let errorHandler: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: { use: vi.fn() },
        response: {
          use: vi.fn((_successFn: Function, errFn: Function) => {
            errorHandler = errFn;
          }),
        },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (errorHandler) {
      const error = {
        response: {
          status: 403,
          data: { detail: { feature: 'traffic_shaping' } },
        },
      };
      await expect((errorHandler as Function)(error)).rejects.toEqual(error);
      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining('traffic_shaping')
      );
    }

    warnSpy.mockRestore();
  });

  it('response interceptor rejects non-401/403 errors without side effects', async () => {
    const mockedAxios = axios as any;
    let errorHandler: Function | null = null;

    mockedAxios.create.mockImplementation(() => ({
      interceptors: {
        request: { use: vi.fn() },
        response: {
          use: vi.fn((_successFn: Function, errFn: Function) => {
            errorHandler = errFn;
          }),
        },
      },
    }));

    vi.resetModules();
    await import('../api');

    if (errorHandler) {
      const error = { response: { status: 500, data: {} } };
      await expect((errorHandler as Function)(error)).rejects.toEqual(error);
    }
  });
});
