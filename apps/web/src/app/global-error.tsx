'use client';

/**
 * Global error boundary for the root layout.
 * This component renders OUTSIDE of the root layout, so it cannot use:
 * - ThemeProvider (next-themes)
 * - AuthProvider
 * - Any other context providers
 *
 * It must include its own <html> and <body> tags.
 */
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en">
      <body style={{
        backgroundColor: '#09090b',
        color: '#fafafa',
        fontFamily: 'system-ui, -apple-system, sans-serif',
        margin: 0,
        padding: 0,
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <div style={{ textAlign: 'center', padding: '2rem' }}>
          <h1 style={{
            fontSize: '1.5rem',
            fontWeight: 600,
            marginBottom: '1rem',
          }}>
            Something went wrong
          </h1>
          <p style={{
            color: '#a1a1aa',
            marginBottom: '1.5rem',
            fontSize: '0.875rem',
          }}>
            {error.message || 'An unexpected error occurred'}
          </p>
          <button
            onClick={reset}
            style={{
              backgroundColor: '#3b82f6',
              color: 'white',
              border: 'none',
              padding: '0.5rem 1rem',
              borderRadius: '0.375rem',
              fontSize: '0.875rem',
              fontWeight: 500,
              cursor: 'pointer',
            }}
          >
            Try again
          </button>
        </div>
      </body>
    </html>
  );
}
