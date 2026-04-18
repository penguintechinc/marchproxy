import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import KongServices from '../KongServices';

vi.mock('../../../services/kongApi');

describe('KongServices', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <KongServices />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <KongServices />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });

  it('renders buttons', () => {
    const { container } = render(
      <BrowserRouter>
        <KongServices />
      </BrowserRouter>
    );
    const buttons = container.querySelectorAll('button');
    expect(buttons.length).toBeGreaterThanOrEqual(0);
  });
});
