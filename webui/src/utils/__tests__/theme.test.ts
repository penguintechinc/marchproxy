/**
 * Tests for theme utility (theme.ts)
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('theme', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it('exports theme object', async () => {
    const { theme } = await import('../theme');
    expect(theme).toBeDefined();
  });

  it('dark mode palette', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.mode).toBe('dark');
  });

  it('primary color', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.primary.main).toBe('#1E3A8A');
  });

  it('secondary color', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.secondary.main).toBe('#FFD700');
  });

  it('background default color', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.background.default).toBe('#1E1E1E');
  });

  it('exports default theme', async () => {
    const module = await import('../theme');
    expect(module.default).toBeDefined();
  });

  it('background paper color', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.background.paper).toBe('#2C2C2C');
  });

  it('primary light color', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.primary.light).toBe('#2563EB');
  });

  it('secondary light color', async () => {
    const { theme } = await import('../theme');
    expect(theme.palette.secondary.light).toBe('#FFF176');
  });
});
