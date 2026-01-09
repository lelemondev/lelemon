'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { EnterpriseGate } from '@/ee/components/feature-gate';

interface BillingInfo {
  plan: 'free' | 'pro' | 'enterprise';
  status: 'active' | 'cancelled' | 'past_due';
  currentPeriodEnd: string | null;
  usage: {
    traces: number;
    spans: number;
  };
  limits: {
    maxTraces: number;
    maxSpans: number;
  };
}

export default function BillingPage() {
  const [billing, setBilling] = useState<BillingInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // TODO: Fetch billing info from API
    setIsLoading(false);
    setBilling({
      plan: 'free',
      status: 'active',
      currentPeriodEnd: null,
      usage: { traces: 1234, spans: 5678 },
      limits: { maxTraces: 10000, maxSpans: 100000 },
    });
  }, []);

  const planFeatures: Record<string, string[]> = {
    free: ['3 projects', '10K traces/month', '7 day retention', '1 team member'],
    pro: ['20 projects', '1M traces/month', '30 day retention', '10 team members'],
    enterprise: ['Unlimited projects', 'Unlimited traces', '90 day retention', 'Unlimited team members'],
  };

  return (
    <EnterpriseGate
      fallback={
        <div className="p-4 sm:p-6 lg:p-8 space-y-8 max-w-4xl overflow-auto h-full">
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Billing</h1>
            <p className="text-zinc-500 dark:text-zinc-400 mt-1">
              Billing management is an enterprise feature.
            </p>
          </div>
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-zinc-500 dark:text-zinc-400 mb-4">
                  Upgrade to access billing features and higher limits.
                </p>
                <Button>View Pricing</Button>
              </div>
            </CardContent>
          </Card>
        </div>
      }
    >
      <div className="p-4 sm:p-6 lg:p-8 space-y-8 max-w-4xl overflow-auto h-full">
        <div>
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">Billing</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">
            Manage your subscription and billing details.
          </p>
        </div>

        {isLoading ? (
          <div className="space-y-6">
            <div className="h-48 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
            <div className="h-32 bg-zinc-200 dark:bg-zinc-800 rounded-2xl animate-pulse" />
          </div>
        ) : billing && (
          <>
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Current Plan</CardTitle>
                    <CardDescription>
                      {billing.currentPeriodEnd
                        ? `Renews on ${new Date(billing.currentPeriodEnd).toLocaleDateString()}`
                        : 'Free tier'}
                    </CardDescription>
                  </div>
                  <Badge
                    className={
                      billing.plan === 'enterprise'
                        ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                        : billing.plan === 'pro'
                        ? 'bg-blue-500/10 text-blue-600 dark:text-blue-400'
                        : 'bg-zinc-500/10 text-zinc-600 dark:text-zinc-400'
                    }
                  >
                    {billing.plan.charAt(0).toUpperCase() + billing.plan.slice(1)}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {planFeatures[billing.plan]?.map((feature, i) => (
                    <li key={i} className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-300">
                      <svg className="w-4 h-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      {feature}
                    </li>
                  ))}
                </ul>
                {billing.plan !== 'enterprise' && (
                  <div className="mt-6 pt-6 border-t border-zinc-200 dark:border-zinc-800">
                    <Button>Upgrade Plan</Button>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Usage This Month</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <div className="flex justify-between text-sm mb-2">
                    <span className="text-zinc-600 dark:text-zinc-400">Traces</span>
                    <span className="text-zinc-900 dark:text-white">
                      {billing.usage.traces.toLocaleString()} / {billing.limits.maxTraces.toLocaleString()}
                    </span>
                  </div>
                  <div className="h-2 bg-zinc-200 dark:bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-amber-500 rounded-full"
                      style={{ width: `${Math.min((billing.usage.traces / billing.limits.maxTraces) * 100, 100)}%` }}
                    />
                  </div>
                </div>
                <div>
                  <div className="flex justify-between text-sm mb-2">
                    <span className="text-zinc-600 dark:text-zinc-400">Spans</span>
                    <span className="text-zinc-900 dark:text-white">
                      {billing.usage.spans.toLocaleString()} / {billing.limits.maxSpans.toLocaleString()}
                    </span>
                  </div>
                  <div className="h-2 bg-zinc-200 dark:bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-amber-500 rounded-full"
                      style={{ width: `${Math.min((billing.usage.spans / billing.limits.maxSpans) * 100, 100)}%` }}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </EnterpriseGate>
  );
}
