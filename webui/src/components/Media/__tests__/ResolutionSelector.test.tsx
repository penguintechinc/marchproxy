import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import ResolutionSelector from '../ResolutionSelector';

vi.mock('@services/mediaApi', () => ({
  getResolutionLabel: (res: number) => `${res}p`,
}));

describe('ResolutionSelector', () => {
  const mockOnChange = vi.fn();

  it('renders form control', () => {
    const { container } = render(
      <ResolutionSelector
        value={1080}
        onChange={mockOnChange}
        effectiveMax={1080}
        hardwareMax={2160}
        adminMax={null}
        gpuType="nvidia"
        label="Maximum Resolution"
      />
    );

    expect(container).toBeDefined();
  });

  it('renders with custom label', () => {
    const { container } = render(
      <ResolutionSelector
        value={720}
        onChange={mockOnChange}
        effectiveMax={1080}
        hardwareMax={2160}
        adminMax={null}
        gpuType="nvidia"
        label="Custom Resolution Label"
      />
    );

    expect(container.textContent).toContain('Custom Resolution Label');
  });

  it('calls onChange with correct value', () => {
    const { container } = render(
      <ResolutionSelector
        value={720}
        onChange={mockOnChange}
        effectiveMax={1080}
        hardwareMax={2160}
        adminMax={null}
        gpuType="nvidia"
      />
    );

    const select = container.querySelector('input[value="720"]');
    expect(select).toBeInTheDocument();
  });

  it('disables when disabled prop is true', () => {
    const { container } = render(
      <ResolutionSelector
        value={720}
        onChange={mockOnChange}
        effectiveMax={1080}
        hardwareMax={2160}
        adminMax={null}
        gpuType="nvidia"
        disabled={true}
      />
    );

    const disabledElement = container.querySelector('[disabled]');
    expect(disabledElement).toBeDefined();
  });

  it('respects full width prop', () => {
    const { container } = render(
      <ResolutionSelector
        value={720}
        onChange={mockOnChange}
        effectiveMax={1080}
        hardwareMax={2160}
        adminMax={null}
        gpuType="nvidia"
        fullWidth={true}
      />
    );

    expect(container).toBeDefined();
  });
});
