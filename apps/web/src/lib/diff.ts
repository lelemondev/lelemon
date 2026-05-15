// Line-by-line diff between two text blobs, no dependencies.
//
// Why hand-rolled: pulling in `jsdiff` for a single use case adds 30+ KB
// to the bundle and a maintenance surface. The classic LCS dynamic programming
// approach is ~50 lines, runs in O(m·n) which is fine for prompts (typically
// tens of lines), and the output is exactly what the renderer wants — a flat
// list of (type, text) hunks.

export type DiffOp = 'equal' | 'added' | 'removed';

export interface DiffHunk {
  type: DiffOp;
  text: string;
}

/**
 * lineDiff produces a sequence of hunks describing how `before` becomes
 * `after`, one line at a time. Equal lines appear once with type `equal`;
 * lines only in `before` are `removed`, lines only in `after` are `added`.
 *
 * The output preserves original line ordering — replaying it produces the
 * "after" text when you drop `removed` hunks, and the "before" text when you
 * drop `added` hunks. That's what unified diff renderers expect.
 */
export function lineDiff(before: string, after: string): DiffHunk[] {
  const a = before.split('\n');
  const b = after.split('\n');
  const m = a.length;
  const n = b.length;

  // dp[i][j] = length of longest common subsequence of a[..i-1] and b[..j-1].
  // Using flat arrays + index math keeps the allocation small.
  const dp: number[] = new Array((m + 1) * (n + 1)).fill(0);
  const at = (i: number, j: number) => i * (n + 1) + j;

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[at(i, j)] =
        a[i - 1] === b[j - 1]
          ? dp[at(i - 1, j - 1)] + 1
          : Math.max(dp[at(i - 1, j)], dp[at(i, j - 1)]);
    }
  }

  // Walk back from (m, n) to (0, 0), emitting hunks. We accumulate in reverse
  // and flip at the end — simpler than threading a stack.
  const out: DiffHunk[] = [];
  let i = m;
  let j = n;
  while (i > 0 && j > 0) {
    if (a[i - 1] === b[j - 1]) {
      out.push({ type: 'equal', text: a[i - 1] });
      i--;
      j--;
    } else if (dp[at(i - 1, j)] >= dp[at(i, j - 1)]) {
      out.push({ type: 'removed', text: a[i - 1] });
      i--;
    } else {
      out.push({ type: 'added', text: b[j - 1] });
      j--;
    }
  }
  while (i > 0) {
    out.push({ type: 'removed', text: a[i - 1] });
    i--;
  }
  while (j > 0) {
    out.push({ type: 'added', text: b[j - 1] });
    j--;
  }
  return out.reverse();
}

/**
 * diffStats summarises a hunk list — useful to show "+12 -3" at the top of
 * the diff view.
 */
export function diffStats(hunks: DiffHunk[]): { added: number; removed: number } {
  let added = 0;
  let removed = 0;
  for (const h of hunks) {
    if (h.type === 'added') added++;
    else if (h.type === 'removed') removed++;
  }
  return { added, removed };
}
