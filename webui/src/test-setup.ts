import '@testing-library/jest-dom';
import { vi } from 'vitest';
import React from 'react';

// Mock @mui/x-date-pickers to avoid date-fns adapter issues
vi.mock('@mui/x-date-pickers', () => ({
  LocalizationProvider: ({ children }: any) => React.createElement('div', {}, children),
  DateTimePicker: () => null,
  DatePicker: () => null,
  TimePicker: () => null,
}));

// Mock @mui/x-date-pickers/LocalizationProvider
vi.mock('@mui/x-date-pickers/LocalizationProvider', () => ({
  LocalizationProvider: ({ children }: any) => React.createElement('div', {}, children),
}));

// Mock @mui/x-date-pickers/DateTimePicker
vi.mock('@mui/x-date-pickers/DateTimePicker', () => ({
  DateTimePicker: () => null,
}));

// Mock @mui/x-date-pickers/AdapterDateFns to avoid date-fns internal path issues
vi.mock('@mui/x-date-pickers/AdapterDateFns', () => ({
  AdapterDateFns: class {
    constructor() {}
  },
}));
