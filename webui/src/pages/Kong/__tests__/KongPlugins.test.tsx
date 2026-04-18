import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import KongPlugins from '../KongPlugins';

vi.mock('../../../services/kongApi');

describe('KongPlugins', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <KongPlugins />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <KongPlugins />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
