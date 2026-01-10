'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth-context';
import { LemonIcon } from '@/components/lemon-icon';

const API_URL = process.env.NEXT_PUBLIC_API_URL || '';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await login(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
      setLoading(false);
    }
  };

  const handleGoogleLogin = () => {
    window.location.href = `${API_URL}/api/v1/auth/google`;
  };

  return (
    <div className="min-h-screen bg-[#FAFDF7] flex flex-col">
      <div className="fixed inset-0 pointer-events-none overflow-hidden">
        <div className="absolute top-0 right-0 w-[600px] h-[600px] bg-gradient-to-bl from-[#FACC15]/15 via-[#FEF08A]/8 to-transparent rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 w-[400px] h-[400px] bg-gradient-to-tr from-[#FACC15]/8 to-transparent rounded-full blur-3xl" />
      </div>

      <header className="relative z-10 p-6">
        <Link href="/" className="inline-flex items-center gap-2.5 group">
          <LemonIcon className="w-9 h-9 transition-transform group-hover:rotate-12 group-hover:scale-110" />
          <span className="font-bold text-xl tracking-tight text-[#18181B]">Lelemon</span>
        </Link>
      </header>

      <main className="relative z-10 flex-1 flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-md">
          <div className="bg-white rounded-2xl shadow-xl shadow-[#FACC15]/10 border border-[#18181B]/5 p-8">
            <div className="text-center mb-8">
              <h1 className="text-2xl font-bold text-[#18181B] mb-2">Welcome back</h1>
              <p className="text-[#71717A]">Sign in to your account</p>
            </div>

            <div className="mb-6">
              <button
                onClick={handleGoogleLogin}
                className="w-full flex items-center justify-center gap-3 px-4 py-3 rounded-xl border border-[#18181B]/10 bg-white hover:bg-[#F4F4F5] transition-colors font-medium text-[#18181B] cursor-pointer"
              >
                <svg className="w-5 h-5" viewBox="0 0 24 24">
                  <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                  <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                  <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
                  <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
                </svg>
                Continue with Google
              </button>
            </div>

            <div className="relative my-6">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-[#18181B]/10" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-4 bg-white text-[#A1A1AA]">or continue with email</span>
              </div>
            </div>

            <form onSubmit={handleLogin} className="space-y-4">
              {error && (
                <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
                  {error}
                </div>
              )}

              <div>
                <label htmlFor="email" className="block text-sm font-medium text-[#18181B] mb-2">
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="w-full px-4 py-3 rounded-xl border border-[#18181B]/10 bg-[#F4F4F5] focus:bg-white focus:border-[#FACC15] focus:ring-2 focus:ring-[#FACC15]/20 outline-none transition-all text-[#18181B] placeholder:text-[#A1A1AA]"
                  placeholder="you@example.com"
                />
              </div>

              <div>
                <label htmlFor="password" className="block text-sm font-medium text-[#18181B] mb-2">
                  Password
                </label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  className="w-full px-4 py-3 rounded-xl border border-[#18181B]/10 bg-[#F4F4F5] focus:bg-white focus:border-[#FACC15] focus:ring-2 focus:ring-[#FACC15]/20 outline-none transition-all text-[#18181B] placeholder:text-[#A1A1AA]"
                  placeholder="********"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full py-3 px-4 rounded-xl bg-[#FACC15] text-[#18181B] font-semibold hover:bg-[#EAB308] transition-all shadow-[0_4px_20px_-4px_rgba(255,216,77,0.5)] hover:shadow-[0_8px_30px_-4px_rgba(255,216,77,0.6)] disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
              >
                {loading ? 'Signing in...' : 'Sign in'}
              </button>
            </form>

            <p className="mt-6 text-center text-sm text-[#71717A]">
              Don&apos;t have an account?{' '}
              <Link href="/signup" className="text-[#A16207] font-medium hover:underline">
                Sign up
              </Link>
            </p>
          </div>

          {/* Beta notice */}
          <div className="mt-6 text-center">
            <div className="inline-flex flex-col items-center gap-1 px-4 py-3 rounded-xl bg-[#18181B]/5 border border-[#18181B]/5">
              <p className="text-sm font-medium text-[#18181B]/70">
                <span className="mr-1">🍋</span> Fresh from main. We ship daily.
              </p>
              <p className="text-xs text-[#71717A]">
                Found a bug?{' '}
                <a
                  href="https://github.com/lelemondev/lelemon/issues"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#A16207] hover:underline"
                >
                  Tell us
                </a>
                {' '}before our CEO does.
              </p>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
