'use client';

import { useState } from 'react';

export function BetaBanner() {
  const [isHovered, setIsHovered] = useState(false);
  const [clickCount, setClickCount] = useState(0);

  const easterEggMessages = [
    "Found a bug? Tell us before our CEO does.",
    "Yes, we know about that bug. We're ignoring it.",
    "Have you tried turning it off and on again?",
    "It's not a bug, it's a surprise feature.",
    "Our tests passed. The tests were wrong.",
    "Works on my machine ¯\\_(ツ)_/¯",
  ];

  const handleLemonClick = () => {
    setClickCount((prev) => (prev + 1) % easterEggMessages.length);
  };

  return (
    <div className="mt-6">
      <div
        className="relative overflow-hidden rounded-2xl bg-[#1a1a1a] border border-[#333] shadow-2xl"
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        {/* Animated gradient border */}
        <div className="absolute inset-0 bg-gradient-to-r from-yellow-500/20 via-orange-500/20 to-yellow-500/20 opacity-0 group-hover:opacity-100 transition-opacity" />

        {/* Terminal header */}
        <div className="flex items-center gap-2 px-4 py-2 bg-[#252525] border-b border-[#333]">
          <div className="flex gap-1.5">
            <div className="w-3 h-3 rounded-full bg-[#ff5f57]" />
            <div className="w-3 h-3 rounded-full bg-[#ffbd2e]" />
            <div className="w-3 h-3 rounded-full bg-[#28ca41]" />
          </div>
          <span className="ml-2 text-xs text-[#666] font-mono">status.tsx</span>
          <div className="ml-auto flex items-center gap-2">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-yellow-500" />
            </span>
            <span className="text-xs text-yellow-500 font-mono">BETA</span>
          </div>
        </div>

        {/* Terminal content */}
        <div className="p-4 font-mono text-sm">
          {/* Main message */}
          <div className="flex items-start gap-3">
            <button
              onClick={handleLemonClick}
              className={`
                text-2xl transition-all duration-300 cursor-pointer
                hover:scale-125 active:scale-95
                ${isHovered ? 'animate-bounce' : ''}
              `}
              title="Click me!"
            >
              🍋
            </button>
            <div className="flex-1">
              <div className="flex items-center gap-2 text-[#4ade80]">
                <span className="text-[#666]">$</span>
                <span className="typing-animation">
                  Fresh from <span className="text-[#f472b6]">main</span>. We ship daily.
                </span>
                <span className="animate-pulse text-[#4ade80]">▊</span>
              </div>

              <div className="mt-2 text-[#888] text-xs leading-relaxed">
                <span className="text-[#666]">{'//'}</span>{' '}
                <span
                  className={`transition-all duration-300 ${
                    clickCount > 0 ? 'text-yellow-400' : ''
                  }`}
                >
                  {easterEggMessages[clickCount]}
                </span>
              </div>
            </div>
          </div>

          {/* Action row */}
          <div className="mt-4 pt-3 border-t border-[#333] flex items-center justify-between">
            <div className="flex items-center gap-4 text-xs">
              <div className="flex items-center gap-1.5 text-[#666]">
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
                <span>v0.1.0</span>
              </div>
              <div className="flex items-center gap-1.5 text-[#666]">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-500" />
                <span>All systems operational</span>
              </div>
            </div>

            <a
              href="https://github.com/lelemondev/lelemon/issues"
              target="_blank"
              rel="noopener noreferrer"
              className="
                group flex items-center gap-2 px-3 py-1.5 rounded-lg
                bg-yellow-500/10 hover:bg-yellow-500/20
                border border-yellow-500/30 hover:border-yellow-500/50
                text-yellow-500 text-xs font-medium
                transition-all duration-200
                hover:shadow-[0_0_20px_rgba(234,179,8,0.3)]
              "
            >
              <svg className="w-3.5 h-3.5 transition-transform group-hover:rotate-12" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              Report Bug
              <svg className="w-3 h-3 transition-transform group-hover:translate-x-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </a>
          </div>
        </div>

        {/* Decorative corner accents */}
        <div className="absolute top-0 right-0 w-20 h-20 bg-gradient-to-bl from-yellow-500/10 to-transparent pointer-events-none" />
        <div className="absolute bottom-0 left-0 w-16 h-16 bg-gradient-to-tr from-yellow-500/5 to-transparent pointer-events-none" />
      </div>

      {/* Pro tip that appears on hover */}
      <div
        className={`
          mt-2 text-center text-xs text-[#999] font-mono
          transition-all duration-300
          ${isHovered ? 'opacity-100 translate-y-0' : 'opacity-0 -translate-y-2'}
        `}
      >
        <span className="text-yellow-600">pro tip:</span> click the lemon 🍋
      </div>
    </div>
  );
}
