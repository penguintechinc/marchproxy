import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import CodecBadge from '../CodecBadge';

describe('CodecBadge', () => {
  it('renders codec name', () => {
    render(<CodecBadge codec="H.264" />);
    expect(screen.getByText('H.264')).toBeInTheDocument();
  });

  it('renders H.264 with correct styling', () => {
    const { container } = render(<CodecBadge codec="H.264" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders H.265 with correct styling', () => {
    const { container } = render(<CodecBadge codec="H.265" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders VP8 with correct styling', () => {
    const { container } = render(<CodecBadge codec="VP8" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders VP9 with correct styling', () => {
    const { container } = render(<CodecBadge codec="VP9" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders AV1 with correct styling', () => {
    const { container } = render(<CodecBadge codec="AV1" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders unknown codec', () => {
    render(<CodecBadge codec="UNKNOWN" />);
    expect(screen.getByText('UNKNOWN')).toBeInTheDocument();
  });
});
