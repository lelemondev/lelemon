import { describe, expect, it } from 'vitest';

import { diffStats, lineDiff, type DiffHunk } from './diff';

// Helper to extract just (type, text) tuples — keeps assertions readable.
const flatten = (h: DiffHunk[]) => h.map((x) => [x.type, x.text] as const);

describe('lineDiff', () => {
  it('returns all "equal" when texts match', () => {
    const out = lineDiff('one\ntwo\nthree', 'one\ntwo\nthree');
    expect(flatten(out)).toEqual([
      ['equal', 'one'],
      ['equal', 'two'],
      ['equal', 'three'],
    ]);
  });

  it('marks added lines when the after grows', () => {
    const out = lineDiff('one\ntwo', 'one\ntwo\nthree');
    expect(flatten(out)).toEqual([
      ['equal', 'one'],
      ['equal', 'two'],
      ['added', 'three'],
    ]);
  });

  it('marks removed lines when the before shrinks', () => {
    const out = lineDiff('one\ntwo\nthree', 'one\nthree');
    expect(flatten(out)).toEqual([
      ['equal', 'one'],
      ['removed', 'two'],
      ['equal', 'three'],
    ]);
  });

  it('handles a single line replacement as add+remove around equals', () => {
    const out = lineDiff('hello\nworld', 'hello\nfriend');
    const types = out.map((h) => h.type);
    // Order isn't fully canonical for replacements; assert presence + equal anchor.
    expect(types.includes('equal')).toBe(true);
    expect(types.includes('added')).toBe(true);
    expect(types.includes('removed')).toBe(true);
    const added = out.find((h) => h.type === 'added');
    const removed = out.find((h) => h.type === 'removed');
    expect(added?.text).toBe('friend');
    expect(removed?.text).toBe('world');
  });

  it('produces an empty middle equal run when there is no overlap', () => {
    const out = lineDiff('a\nb', 'c\nd');
    const types = out.map((h) => h.type);
    expect(types.filter((t) => t === 'removed').length).toBe(2);
    expect(types.filter((t) => t === 'added').length).toBe(2);
    expect(types.includes('equal')).toBe(false);
  });

  it('handles "before" empty (everything is added)', () => {
    const out = lineDiff('', 'new\nlines');
    // An empty `before` still split into one empty line — it matches a leading
    // empty line in `after` if present, otherwise lands as a single removed.
    const added = out.filter((h) => h.type === 'added').map((h) => h.text);
    expect(added).toContain('new');
    expect(added).toContain('lines');
  });

  it('handles "after" empty (everything is removed)', () => {
    const out = lineDiff('gone\nalso-gone', '');
    const removed = out.filter((h) => h.type === 'removed').map((h) => h.text);
    expect(removed).toContain('gone');
    expect(removed).toContain('also-gone');
  });

  it('preserves ordering — dropping `added` reconstructs `before`', () => {
    const before = 'one\ntwo\nthree\nfour';
    const after = 'one\nTWO\nthree\nfive';
    const out = lineDiff(before, after);
    const beforeOnly = out
      .filter((h) => h.type !== 'added')
      .map((h) => h.text)
      .join('\n');
    expect(beforeOnly).toBe(before);
  });

  it('preserves ordering — dropping `removed` reconstructs `after`', () => {
    const before = 'one\ntwo\nthree\nfour';
    const after = 'one\nTWO\nthree\nfive';
    const out = lineDiff(before, after);
    const afterOnly = out
      .filter((h) => h.type !== 'removed')
      .map((h) => h.text)
      .join('\n');
    expect(afterOnly).toBe(after);
  });
});

describe('diffStats', () => {
  it('counts added and removed hunks', () => {
    const hunks = lineDiff('a\nb\nc', 'a\nB\nc\nd');
    const stats = diffStats(hunks);
    expect(stats.added).toBe(2); // "B" and "d"
    expect(stats.removed).toBe(1); // "b"
  });

  it('returns zeros for identical inputs', () => {
    const stats = diffStats(lineDiff('same\nsame', 'same\nsame'));
    expect(stats).toEqual({ added: 0, removed: 0 });
  });
});
