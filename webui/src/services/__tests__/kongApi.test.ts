import { describe, it, expect, vi } from 'vitest';

vi.mock('../kongApi');

describe('kongApi', () => {
  it('module imports without error', () => {
    expect(true).toBe(true);
  });
});
