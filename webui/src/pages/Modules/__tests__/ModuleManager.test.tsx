import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import ModuleManager from '../ModuleManager';

describe('ModuleManager', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <ModuleManager />
      </BrowserRouter>
    );

    expect(container).toBeDefined();
  });
});
