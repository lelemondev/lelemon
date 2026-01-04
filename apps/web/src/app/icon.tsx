import { ImageResponse } from 'next/og';

export const size = {
  width: 64,
  height: 64,
};
export const contentType = 'image/png';

export default function Icon() {
  return new ImageResponse(
    (
      <svg viewBox="0 0 64 64" style={{ width: '100%', height: '100%' }}>
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
    ),
    {
      ...size,
    }
  );
}
