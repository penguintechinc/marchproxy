import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import MTLS from '../mTLS';

vi.mock('../../../services/securityApi');

describe('mTLS', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <MTLS />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <MTLS />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
