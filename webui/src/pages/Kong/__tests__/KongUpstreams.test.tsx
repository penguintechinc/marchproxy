import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import KongUpstreams from '../KongUpstreams';

vi.mock('../../../services/kongApi');

describe('KongUpstreams', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <KongUpstreams />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <KongUpstreams />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
