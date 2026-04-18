import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Users from '../Users';

describe('Users', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <Users />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });
});
