import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ProtocolBadge from '../ProtocolBadge';

describe('ProtocolBadge', () => {
  it('renders protocol name', () => {
    render(<ProtocolBadge protocol="RTMP" />);
    expect(screen.getByText('RTMP')).toBeInTheDocument();
  });

  it('renders RTMP badge', () => {
    const { container } = render(<ProtocolBadge protocol="RTMP" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders RTSP badge', () => {
    const { container } = render(<ProtocolBadge protocol="RTSP" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders HLS badge', () => {
    const { container } = render(<ProtocolBadge protocol="HLS" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders DASH badge', () => {
    const { container } = render(<ProtocolBadge protocol="DASH" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders SRT badge', () => {
    const { container } = render(<ProtocolBadge protocol="SRT" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders WebRTC badge', () => {
    const { container } = render(<ProtocolBadge protocol="WebRTC" />);
    const badge = container.firstChild;
    expect(badge).toBeInTheDocument();
  });

  it('renders unknown protocol', () => {
    render(<ProtocolBadge protocol="UNKNOWN" />);
    expect(screen.getByText('UNKNOWN')).toBeInTheDocument();
  });
});
