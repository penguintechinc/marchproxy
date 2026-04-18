import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import TrafficSlider from '../TrafficSlider';

describe('TrafficSlider', () => {
  const mockOnWeightChange = vi.fn();
  const mockOnSave = vi.fn();

  it('renders traffic distribution title', () => {
    render(
      <TrafficSlider
        blueVersion="v1.0.0"
        greenVersion="v1.1.0"
        initialBlueWeight={80}
        initialGreenWeight={20}
        onWeightChange={mockOnWeightChange}
      />
    );

    expect(screen.getByText('Traffic Distribution')).toBeInTheDocument();
  });

  it('displays blue and green versions', () => {
    render(
      <TrafficSlider
        blueVersion="v1.0.0"
        greenVersion="v1.1.0"
        initialBlueWeight={80}
        initialGreenWeight={20}
        onWeightChange={mockOnWeightChange}
      />
    );

    expect(screen.getByText('v1.0.0')).toBeInTheDocument();
    expect(screen.getByText('v1.1.0')).toBeInTheDocument();
  });

  it('renders quick preset buttons', () => {
    render(
      <TrafficSlider
        blueVersion="v1.0.0"
        greenVersion="v1.1.0"
        initialBlueWeight={50}
        initialGreenWeight={50}
        onWeightChange={mockOnWeightChange}
      />
    );

    expect(screen.getByRole('button', { name: /All Blue/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /All Green/i })).toBeInTheDocument();
  });

  it('shows save button when showSaveButton is true', () => {
    render(
      <TrafficSlider
        blueVersion="v1.0.0"
        greenVersion="v1.1.0"
        initialBlueWeight={50}
        initialGreenWeight={50}
        onWeightChange={mockOnWeightChange}
        onSave={mockOnSave}
        showSaveButton={true}
      />
    );

    expect(screen.getByRole('button', { name: /Apply Traffic Changes/i })).toBeInTheDocument();
  });

  it('renders without crashing when disabled', () => {
    const { container } = render(
      <TrafficSlider
        blueVersion="v1.0.0"
        greenVersion="v1.1.0"
        initialBlueWeight={50}
        initialGreenWeight={50}
        onWeightChange={mockOnWeightChange}
        disabled={true}
      />
    );

    expect(container).toBeDefined();
  });
});
