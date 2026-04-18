import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import BlueGreen from '../BlueGreen';

describe('BlueGreen', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <BlueGreen />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <BlueGreen />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
