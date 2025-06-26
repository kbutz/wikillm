import React from 'react';
import { render, screen } from '@testing-library/react';
import App from './App';

test('renders AI Assistant app', () => {
  render(<App />);
  const linkElement = screen.getByText(/AI Assistant/i);
  expect(linkElement).toBeInTheDocument();
});
