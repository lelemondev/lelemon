import { ImageResponse } from 'next/og';

export const runtime = 'edge';

export const alt = 'Lelemon - Lightweight LLM Observability';
export const size = {
  width: 1200,
  height: 630,
};
export const contentType = 'image/png';

export default async function Image() {
  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: '#FAFDF7',
          backgroundImage: 'radial-gradient(circle at 80% 20%, rgba(250, 204, 21, 0.15) 0%, transparent 50%)',
        }}
      >
        {/* Lemon Icon */}
        <svg
          width="120"
          height="120"
          viewBox="0 0 64 64"
          style={{ marginBottom: 24 }}
        >
          <defs>
            <linearGradient id="g" x1="8" y1="8" x2="56" y2="60">
              <stop stopColor="#FACC15" />
              <stop offset="1" stopColor="#CA8A04" />
            </linearGradient>
          </defs>
          <ellipse cx="32" cy="34" rx="24" ry="26" fill="url(#g)" />
          <ellipse cx="28" cy="30" rx="16" ry="18" fill="#FEF9C3" opacity="0.45" />
          <path d="M32 8C32 8 28 4 32 2C36 4 32 8 32 8Z" fill="#15803D" />
          <path d="M30 8C28 6 24 7 24 7" stroke="#15803D" strokeWidth="2" strokeLinecap="round" fill="none" />
        </svg>

        {/* Title */}
        <div
          style={{
            display: 'flex',
            fontSize: 64,
            fontWeight: 800,
            color: '#18181B',
            marginBottom: 16,
            letterSpacing: '-0.02em',
          }}
        >
          Lelemon
        </div>

        {/* Tagline */}
        <div
          style={{
            display: 'flex',
            fontSize: 32,
            color: '#3F3F46',
            marginBottom: 32,
          }}
        >
          Lightweight LLM Observability
        </div>

        {/* Features */}
        <div
          style={{
            display: 'flex',
            gap: 32,
            fontSize: 20,
            color: '#71717A',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ color: '#FACC15' }}>⚡</span> Zero config
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ color: '#22C55E' }}>📦</span> Super lightweight SDK
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ color: '#FACC15' }}>✓</span> No overhead
          </div>
        </div>
      </div>
    ),
    {
      ...size,
    }
  );
}
