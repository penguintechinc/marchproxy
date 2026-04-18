import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import KongRoutes from '../KongRoutes';

vi.mock('../../../services/kongApi');

describe('KongRoutes', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <KongRoutes />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <KongRoutes />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
