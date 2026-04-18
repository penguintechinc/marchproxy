import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import KongCertificates from '../KongCertificates';

vi.mock('../../../services/certificateApi');

describe('KongCertificates', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <BrowserRouter>
        <KongCertificates />
      </BrowserRouter>
    );
    expect(container).toBeDefined();
  });

  it('renders div elements', () => {
    const { container } = render(
      <BrowserRouter>
        <KongCertificates />
      </BrowserRouter>
    );
    const divs = container.querySelectorAll('div');
    expect(divs.length).toBeGreaterThan(0);
  });
});
