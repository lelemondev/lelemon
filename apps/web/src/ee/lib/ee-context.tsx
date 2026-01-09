'use client';

import { createContext, useContext, useState, useEffect, useMemo, useCallback, ReactNode } from 'react';

interface FeaturesResponse {
  edition: 'community' | 'enterprise';
  features: Record<string, boolean>;
}

interface EEContextType {
  edition: string;
  isEnterprise: boolean;
  isLoading: boolean;
  hasFeature: (feature: string) => boolean;
  features: Record<string, boolean>;
}

const EEContext = createContext<EEContextType | undefined>(undefined);

/**
 * Hook to access EE context. Throws if used outside EEProvider.
 */
export function useEE(): EEContextType {
  const context = useContext(EEContext);
  if (context === undefined) {
    throw new Error('useEE must be used within an EEProvider');
  }
  return context;
}

interface EEProviderProps {
  children: ReactNode;
}

export function EEProvider({ children }: EEProviderProps) {
  const [edition, setEdition] = useState<'community' | 'enterprise'>('community');
  const [features, setFeatures] = useState<Record<string, boolean>>({});
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchFeatures = async () => {
      try {
        const apiUrl = process.env.NEXT_PUBLIC_API_URL || '';
        const response = await fetch(`${apiUrl}/api/v1/features`);
        if (response.ok) {
          const data: FeaturesResponse = await response.json();
          setEdition(data.edition);
          setFeatures(data.features);
        }
      } catch {
        // Silent fail - default to community edition
        console.debug('Failed to fetch features, defaulting to community edition');
      } finally {
        setIsLoading(false);
      }
    };

    fetchFeatures();
  }, []);

  // Memoize callback to prevent re-renders in consumers
  const hasFeature = useCallback((feature: string): boolean => {
    return features[feature] ?? false;
  }, [features]);

  // Memoize context value to prevent unnecessary re-renders
  const value = useMemo<EEContextType>(() => ({
    edition,
    isEnterprise: edition === 'enterprise',
    isLoading,
    hasFeature,
    features,
  }), [edition, isLoading, hasFeature, features]);

  return <EEContext.Provider value={value}>{children}</EEContext.Provider>;
}
