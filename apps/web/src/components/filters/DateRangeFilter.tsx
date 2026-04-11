'use client';

import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

interface DateRangeFilterProps {
  from: Date | null;
  to: Date | null;
  onChange: (from: Date | null, to: Date | null) => void;
}

const PRESETS = [
  { label: 'Last 24h', hours: 24 },
  { label: 'Last 7d', hours: 24 * 7 },
  { label: 'Last 30d', hours: 24 * 30 },
];

export function DateRangeFilter({ from, to, onChange }: DateRangeFilterProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [customFrom, setCustomFrom] = useState('');
  const [customTo, setCustomTo] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Sync custom inputs with props
  useEffect(() => {
    if (from) {
      setCustomFrom(from.toISOString().slice(0, 16));
    } else {
      setCustomFrom('');
    }
    if (to) {
      setCustomTo(to.toISOString().slice(0, 16));
    } else {
      setCustomTo('');
    }
  }, [from, to]);

  const applyPreset = (hours: number) => {
    const now = new Date();
    const fromDate = new Date(now.getTime() - hours * 60 * 60 * 1000);
    onChange(fromDate, now);
    setIsOpen(false);
  };

  const applyCustomRange = () => {
    const fromDate = customFrom ? new Date(customFrom) : null;
    const toDate = customTo ? new Date(customTo) : null;
    onChange(fromDate, toDate);
    setIsOpen(false);
  };

  const clearRange = () => {
    onChange(null, null);
    setCustomFrom('');
    setCustomTo('');
    setIsOpen(false);
  };

  const getDisplayLabel = (): string => {
    if (!from && !to) return 'Date Range';

    // Check if it matches a preset
    if (from && to) {
      const diffHours = (to.getTime() - from.getTime()) / (1000 * 60 * 60);
      const preset = PRESETS.find(p => Math.abs(p.hours - diffHours) < 1);
      if (preset) return preset.label;
    }

    // Show custom range
    const formatDate = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    if (from && to) return `${formatDate(from)} - ${formatDate(to)}`;
    if (from) return `From ${formatDate(from)}`;
    if (to) return `Until ${formatDate(to)}`;
    return 'Date Range';
  };

  const hasRange = from || to;

  return (
    <div className="relative" ref={dropdownRef}>
      <Button
        variant="outline"
        size="sm"
        className="h-9"
        onClick={() => setIsOpen(!isOpen)}
      >
        <svg className="w-4 h-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        {getDisplayLabel()}
        <svg className="w-4 h-4 ml-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </Button>

      {isOpen && (
        <div className="absolute z-50 mt-1 w-72 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg">
          {/* Presets */}
          <div className="p-2 border-b border-zinc-200 dark:border-zinc-700">
            <span className="text-xs text-zinc-500 dark:text-zinc-400 block mb-2">
              Quick select
            </span>
            <div className="flex gap-1">
              {PRESETS.map((preset) => (
                <Button
                  key={preset.label}
                  variant="outline"
                  size="sm"
                  className="flex-1 h-7 text-xs"
                  onClick={() => applyPreset(preset.hours)}
                >
                  {preset.label}
                </Button>
              ))}
            </div>
          </div>

          {/* Custom Range */}
          <div className="p-2">
            <span className="text-xs text-zinc-500 dark:text-zinc-400 block mb-2">
              Custom range
            </span>
            <div className="space-y-2">
              <div>
                <label className="text-xs text-zinc-600 dark:text-zinc-400">From</label>
                <Input
                  type="datetime-local"
                  value={customFrom}
                  onChange={(e) => setCustomFrom(e.target.value)}
                  className="h-8 text-sm"
                />
              </div>
              <div>
                <label className="text-xs text-zinc-600 dark:text-zinc-400">To</label>
                <Input
                  type="datetime-local"
                  value={customTo}
                  onChange={(e) => setCustomTo(e.target.value)}
                  className="h-8 text-sm"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  className="flex-1 h-8 text-xs bg-amber-500 hover:bg-amber-600 text-zinc-900"
                  onClick={applyCustomRange}
                >
                  Apply
                </Button>
                {hasRange && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-8 text-xs"
                    onClick={clearRange}
                  >
                    Clear
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
