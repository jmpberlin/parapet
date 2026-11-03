import { describe, test, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import Pagination from './Pagination';

describe('Pagination', () => {
  test('renders page info and both buttons', () => {
    render(<Pagination page={2} totalPages={5} onChange={vi.fn()} />);

    expect(screen.getByText('Page 2 of 5')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '← Previous' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next →' })).toBeInTheDocument();
  });

  test('disables Previous on the first page', () => {
    render(<Pagination page={1} totalPages={5} onChange={vi.fn()} />);

    expect(screen.getByRole('button', { name: '← Previous' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next →' })).toBeEnabled();
  });

  test('disables Next on the last page', () => {
    render(<Pagination page={5} totalPages={5} onChange={vi.fn()} />);

    expect(screen.getByRole('button', { name: '← Previous' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Next →' })).toBeDisabled();
  });

  test('calls onChange with the previous page on Previous click', () => {
    const onChange = vi.fn();
    render(<Pagination page={3} totalPages={5} onChange={onChange} />);

    fireEvent.click(screen.getByRole('button', { name: '← Previous' }));

    expect(onChange).toHaveBeenCalledWith(2);
  });

  test('calls onChange with the next page on Next click', () => {
    const onChange = vi.fn();
    render(<Pagination page={3} totalPages={5} onChange={onChange} />);

    fireEvent.click(screen.getByRole('button', { name: 'Next →' }));

    expect(onChange).toHaveBeenCalledWith(4);
  });

  test('does not call onChange when disabled', () => {
    const onChange = vi.fn();
    render(<Pagination page={2} totalPages={5} onChange={onChange} disabled />);

    fireEvent.click(screen.getByRole('button', { name: 'Next →' }));
    fireEvent.click(screen.getByRole('button', { name: '← Previous' }));

    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Next →' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '← Previous' })).toBeDisabled();
  });
});
