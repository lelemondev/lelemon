'use client';

import { ChevronDown } from 'lucide-react';

export type Provider = 'bedrock' | 'anthropic' | 'openai' | 'gemini';

interface ProviderSelectProps {
  value: Provider;
  onChange: (value: Provider) => void;
}

const providers: { value: Provider; label: string; icon: string }[] = [
  { value: 'bedrock', label: 'AWS Bedrock', icon: '☁️' },
  { value: 'anthropic', label: 'Anthropic', icon: '🔮' },
  { value: 'openai', label: 'OpenAI', icon: '🤖' },
  { value: 'gemini', label: 'Google Gemini', icon: '💎' },
];

export function ProviderSelect({ value, onChange }: ProviderSelectProps) {
  const selected = providers.find(p => p.value === value);

  return (
    <div className="relative">
      <select
        value={value}
        onChange={e => onChange(e.target.value as Provider)}
        className="appearance-none bg-zinc-900 border border-zinc-700 rounded-lg px-4 py-2 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
      >
        {providers.map(provider => (
          <option key={provider.value} value={provider.value}>
            {provider.icon} {provider.label}
          </option>
        ))}
      </select>
      <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500 pointer-events-none" />
    </div>
  );
}
