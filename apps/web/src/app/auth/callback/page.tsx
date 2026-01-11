'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';
import { LemonIcon } from '@/components/lemon-icon';

export default function AuthCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { loginWithToken } = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = searchParams.get('token');
    const errorParam = searchParams.get('error');

    if (errorParam) {
      setError(getErrorMessage(errorParam));
      return;
    }

    if (!token) {
      setError('No authentication token received');
      return;
    }

    // Store token and redirect to dashboard
    loginWithToken(token)
      .then(() => {
        router.push('/dashboard');
      })
      .catch((err) => {
        setError(err.message || 'Authentication failed');
      });
  }, [searchParams, loginWithToken, router]);

  if (error) {
    return (
      <div className="min-h-screen bg-[#FAFDF7] flex flex-col items-center justify-center px-6">
        <div className="text-center">
          <LemonIcon className="w-16 h-16 mx-auto mb-6 opacity-50" />
          <h1 className="text-2xl font-bold text-[#18181B] mb-2">Authentication Failed</h1>
          <p className="text-[#71717A] mb-6">{error}</p>
          <a
            href="/login"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-[#FACC15] text-[#18181B] font-semibold hover:bg-[#EAB308] transition-all"
          >
            Back to Login
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#FAFDF7] flex flex-col items-center justify-center px-6">
      <div className="text-center">
        <LemonIcon className="w-16 h-16 mx-auto mb-6 animate-pulse" />
        <h1 className="text-2xl font-bold text-[#18181B] mb-2">Signing you in...</h1>
        <p className="text-[#71717A]">Please wait while we complete authentication</p>
      </div>
    </div>
  );
}

function getErrorMessage(error: string): string {
  switch (error) {
    case 'invalid_state':
      return 'Invalid authentication state. Please try again.';
    case 'access_denied':
      return 'Access was denied. Please try again.';
    case 'no_code':
      return 'No authorization code received. Please try again.';
    case 'auth_failed':
      return 'Authentication failed. Please try again.';
    default:
      return `Authentication error: ${error}`;
  }
}
