import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import ModuleRoutes from '../ModuleRoutes';

describe('ModuleRoutes', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <ModuleRoutes />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });
});
