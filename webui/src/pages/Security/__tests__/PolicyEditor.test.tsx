import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import PolicyEditor from '../PolicyEditor';

vi.mock('../../../services/securityApi');

describe('PolicyEditor', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <PolicyEditor />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <PolicyEditor />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
