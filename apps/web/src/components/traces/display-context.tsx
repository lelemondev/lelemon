'use client';

import { createContext, useContext, ReactNode } from 'react';

interface DisplayConfig {
  /** Model ID → short display name */
  modelAliases: Record<string, string>;
  /** Tag/name → hex color */
  spanColors: Record<string, string>;
}

const defaultConfig: DisplayConfig = {
  modelAliases: {},
  spanColors: {},
};

const DisplayConfigContext = createContext<DisplayConfig>(defaultConfig);

export function DisplayConfigProvider({
  modelAliases,
  spanColors,
  children,
}: Partial<DisplayConfig> & { children: ReactNode }) {
  return (
    <DisplayConfigContext.Provider value={{
      modelAliases: modelAliases ?? {},
      spanColors: spanColors ?? {},
    }}>
      {children}
    </DisplayConfigContext.Provider>
  );
}

export function useDisplayConfig(): DisplayConfig {
  return useContext(DisplayConfigContext);
}

/** Resolve a model ID to its display name (alias or shortened default) */
export function resolveModelName(model: string, aliases: Record<string, string>): string {
  // Check for exact alias
  if (aliases[model]) return aliases[model];

  // Auto-shorten common patterns
  return model
    .replace(/^us\.anthropic\./, '')
    .replace(/^anthropic\./, '')
    .replace(/^models\//, '')
    .replace(/-\d{8}-v\d+:\d+$/, '') // remove version suffix like -20251001-v1:0
    .replace(/-v\d+:\d+$/, '');
}
