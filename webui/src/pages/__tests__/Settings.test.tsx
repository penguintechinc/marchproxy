import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Settings from '../Settings';

describe('Settings', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <Settings />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });
});
