export function LemonIcon({ className, id = 'lemon-gradient' }: { className?: string; id?: string }) {
  return (
    <svg className={className} viewBox="0 0 64 64" fill="none">
      <ellipse cx="32" cy="34" rx="24" ry="26" fill="#EAB308" />
      <ellipse cx="32" cy="34" rx="24" ry="26" fill={`url(#${id})`} />
      <ellipse cx="28" cy="30" rx="16" ry="18" fill="#FEF9C3" opacity="0.45" />
      <path d="M32 8C32 8 28 4 32 2C36 4 32 8 32 8Z" fill="#15803D" />
      <path d="M30 8C28 6 24 7 24 7" stroke="#15803D" strokeWidth="2" strokeLinecap="round" />
      <defs>
        <linearGradient id={id} x1="8" y1="8" x2="56" y2="60">
          <stop stopColor="#FACC15" />
          <stop offset="1" stopColor="#CA8A04" />
        </linearGradient>
      </defs>
    </svg>
  );
}
