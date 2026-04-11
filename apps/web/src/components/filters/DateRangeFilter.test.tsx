import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DateRangeFilter } from './DateRangeFilter';

describe('DateRangeFilter', () => {
  beforeEach(() => {
    // Mock Date.now() for consistent testing
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2025-01-15T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('rendering', () => {
    it('renders button with "Date Range" when no range selected', () => {
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: /Date Range/i })).toBeInTheDocument();
    });

    it('shows "Last 24h" label when 24h preset is active', () => {
      const now = new Date('2025-01-15T12:00:00Z');
      const from = new Date(now.getTime() - 24 * 60 * 60 * 1000);

      render(
        <DateRangeFilter
          from={from}
          to={now}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('Last 24h')).toBeInTheDocument();
    });

    it('shows "Last 7d" label when 7d preset is active', () => {
      const now = new Date('2025-01-15T12:00:00Z');
      const from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

      render(
        <DateRangeFilter
          from={from}
          to={now}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('Last 7d')).toBeInTheDocument();
    });

    it('shows custom date range when not matching preset', () => {
      const from = new Date('2025-01-01T12:00:00');
      const to = new Date('2025-01-10T12:00:00');

      render(
        <DateRangeFilter
          from={from}
          to={to}
          onChange={vi.fn()}
        />
      );

      // Should show formatted date range (contains a dash between dates)
      const button = screen.getByRole('button');
      expect(button.textContent).toMatch(/-/);
    });
  });

  describe('dropdown behavior', () => {
    it('opens dropdown on button click', () => {
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));

      expect(screen.getByText('Quick select')).toBeInTheDocument();
      expect(screen.getByText('Custom range')).toBeInTheDocument();
    });

    it('shows all preset buttons', () => {
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));

      expect(screen.getByRole('button', { name: 'Last 24h' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Last 7d' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Last 30d' })).toBeInTheDocument();
    });
  });

  describe('preset selection', () => {
    it('calls onChange with 24h range when clicking Last 24h', () => {
      const onChange = vi.fn();
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));
      fireEvent.click(screen.getByRole('button', { name: 'Last 24h' }));

      expect(onChange).toHaveBeenCalledTimes(1);
      const [from, to] = onChange.mock.calls[0];

      // Check that range is approximately 24 hours
      const diffHours = (to.getTime() - from.getTime()) / (1000 * 60 * 60);
      expect(diffHours).toBe(24);
    });

    it('calls onChange with 7d range when clicking Last 7d', () => {
      const onChange = vi.fn();
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));
      fireEvent.click(screen.getByRole('button', { name: 'Last 7d' }));

      expect(onChange).toHaveBeenCalledTimes(1);
      const [from, to] = onChange.mock.calls[0];

      const diffDays = (to.getTime() - from.getTime()) / (1000 * 60 * 60 * 24);
      expect(diffDays).toBe(7);
    });

    it('calls onChange with 30d range when clicking Last 30d', () => {
      const onChange = vi.fn();
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));
      fireEvent.click(screen.getByRole('button', { name: 'Last 30d' }));

      expect(onChange).toHaveBeenCalledTimes(1);
      const [from, to] = onChange.mock.calls[0];

      const diffDays = (to.getTime() - from.getTime()) / (1000 * 60 * 60 * 24);
      expect(diffDays).toBe(30);
    });

    it('closes dropdown after selecting preset', () => {
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));
      fireEvent.click(screen.getByRole('button', { name: 'Last 24h' }));

      expect(screen.queryByText('Quick select')).not.toBeInTheDocument();
    });
  });

  describe('custom range', () => {
    it('shows From and To date inputs', () => {
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));

      expect(screen.getByText('From')).toBeInTheDocument();
      expect(screen.getByText('To')).toBeInTheDocument();
    });

    it('calls onChange when clicking Apply with custom dates', () => {
      const onChange = vi.fn();
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));

      // Find datetime-local inputs using their container label
      const fromInput = document.querySelector('input[type="datetime-local"]') as HTMLInputElement;
      const toInput = document.querySelectorAll('input[type="datetime-local"]')[1] as HTMLInputElement;

      expect(fromInput).toBeTruthy();
      expect(toInput).toBeTruthy();

      fireEvent.change(fromInput, { target: { value: '2025-01-01T00:00' } });
      fireEvent.change(toInput, { target: { value: '2025-01-10T00:00' } });

      fireEvent.click(screen.getByRole('button', { name: 'Apply' }));

      expect(onChange).toHaveBeenCalled();
    });

    it('syncs custom inputs with props', () => {
      const from = new Date('2025-01-05T10:00:00Z');
      const to = new Date('2025-01-10T15:00:00Z');

      render(
        <DateRangeFilter
          from={from}
          to={to}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Jan 5/i }));

      const fromInput = document.querySelector('input[type="datetime-local"]') as HTMLInputElement;
      expect(fromInput.value).toContain('2025-01-05');
    });
  });

  describe('clear range', () => {
    it('shows Clear button when range is selected', () => {
      render(
        <DateRangeFilter
          from={new Date()}
          to={new Date()}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button'));

      expect(screen.getByRole('button', { name: 'Clear' })).toBeInTheDocument();
    });

    it('hides Clear button when no range selected', () => {
      render(
        <DateRangeFilter
          from={null}
          to={null}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Date Range/i }));

      expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument();
    });

    it('calls onChange with null values when clicking Clear', () => {
      const onChange = vi.fn();
      render(
        <DateRangeFilter
          from={new Date()}
          to={new Date()}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button'));
      fireEvent.click(screen.getByRole('button', { name: 'Clear' }));

      expect(onChange).toHaveBeenCalledWith(null, null);
    });
  });

  describe('display labels', () => {
    it('shows "From {date}" when only from is set', () => {
      render(
        <DateRangeFilter
          from={new Date('2025-01-05T12:00:00')}
          to={null}
          onChange={vi.fn()}
        />
      );

      const button = screen.getByRole('button');
      expect(button.textContent).toMatch(/From\s+\w+\s+\d+/);
    });

    it('shows "Until {date}" when only to is set', () => {
      render(
        <DateRangeFilter
          from={null}
          to={new Date('2025-01-10T12:00:00')}
          onChange={vi.fn()}
        />
      );

      const button = screen.getByRole('button');
      expect(button.textContent).toMatch(/Until\s+\w+\s+\d+/);
    });
  });
});
