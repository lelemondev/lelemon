import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TagsFilter } from './TagsFilter';

describe('TagsFilter', () => {
  const defaultTags = ['org:abc', 'campaign:123', 'env:production'];

  describe('rendering', () => {
    it('renders button with Tags label', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={[]}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: /Tags/i })).toBeInTheDocument();
    });

    it('returns null when no available tags', () => {
      const { container } = render(
        <TagsFilter
          availableTags={[]}
          selectedTags={[]}
          onChange={vi.fn()}
        />
      );

      expect(container.firstChild).toBeNull();
    });

    it('shows badge with selected count', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc', 'campaign:123']}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('2')).toBeInTheDocument();
    });

    it('shows selected tags as chips below button', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc', 'campaign:123']}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('org:abc')).toBeInTheDocument();
      expect(screen.getByText('campaign:123')).toBeInTheDocument();
    });
  });

  describe('dropdown behavior', () => {
    it('opens dropdown on button click', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={[]}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));

      expect(screen.getByText('Select tags (OR logic)')).toBeInTheDocument();
      expect(screen.getByText('org:abc')).toBeInTheDocument();
      expect(screen.getByText('campaign:123')).toBeInTheDocument();
      expect(screen.getByText('env:production')).toBeInTheDocument();
    });

    it('closes dropdown when clicking button again', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={[]}
          onChange={vi.fn()}
        />
      );

      const button = screen.getByRole('button', { name: /Tags/i });
      fireEvent.click(button);
      expect(screen.getByText('Select tags (OR logic)')).toBeInTheDocument();

      fireEvent.click(button);
      expect(screen.queryByText('Select tags (OR logic)')).not.toBeInTheDocument();
    });
  });

  describe('tag selection', () => {
    it('calls onChange with added tag when clicking unchecked tag', () => {
      const onChange = vi.fn();
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={[]}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));
      fireEvent.click(screen.getByLabelText('org:abc'));

      expect(onChange).toHaveBeenCalledWith(['org:abc']);
    });

    it('calls onChange with removed tag when clicking checked tag', () => {
      const onChange = vi.fn();
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc', 'campaign:123']}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));
      fireEvent.click(screen.getByLabelText('org:abc'));

      expect(onChange).toHaveBeenCalledWith(['campaign:123']);
    });

    it('shows checkboxes as checked for selected tags', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc']}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));

      const checkbox = screen.getByRole('checkbox', { name: 'org:abc' }) as HTMLInputElement;
      expect(checkbox.checked).toBe(true);
    });
  });

  describe('tag removal from chips', () => {
    it('calls onChange without tag when clicking chip', () => {
      const onChange = vi.fn();
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc', 'campaign:123']}
          onChange={onChange}
        />
      );

      // Click the chip (not the dropdown item)
      const chips = screen.getAllByText('org:abc');
      // The chip is rendered below the button
      fireEvent.click(chips[0]);

      expect(onChange).toHaveBeenCalledWith(['campaign:123']);
    });
  });

  describe('clear all', () => {
    it('shows clear all button when tags are selected', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc']}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));

      expect(screen.getByRole('button', { name: /Clear all/i })).toBeInTheDocument();
    });

    it('hides clear all button when no tags selected', () => {
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={[]}
          onChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));

      expect(screen.queryByRole('button', { name: /Clear all/i })).not.toBeInTheDocument();
    });

    it('calls onChange with empty array when clicking clear all', () => {
      const onChange = vi.fn();
      render(
        <TagsFilter
          availableTags={defaultTags}
          selectedTags={['org:abc', 'campaign:123']}
          onChange={onChange}
        />
      );

      fireEvent.click(screen.getByRole('button', { name: /Tags/i }));
      fireEvent.click(screen.getByRole('button', { name: /Clear all/i }));

      expect(onChange).toHaveBeenCalledWith([]);
    });
  });
});
