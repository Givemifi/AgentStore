import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { CheckCircle, ShoppingCart, Sparkles, Zap } from 'lucide-react';
import { toast } from 'sonner';
import { bundlesApi, plansApi, billingApi } from '../../api/client';
import type { CreditBundle } from '../../types';
import LoadingSpinner from '../../components/LoadingSpinner';
import { EmptyState } from '../../components/app';
import { getErrorMessage } from '../../utils/errors';

function formatPrice(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

function isSafeCheckoutReturnPath(value: string): boolean {
  // Must start with a single slash
  if (!value.startsWith('/') || value.startsWith('//')) return false;

  // Separate pathname from query string for validation
  const qIndex = value.indexOf('?');
  const pathname = qIndex >= 0 ? value.slice(0, qIndex) : value;

  // Safely decode; reject malformed encoding
  let decoded: string;
  try {
    decoded = decodeURIComponent(pathname);
  } catch {
    return false;
  }

  // Reject backslashes and path-traversal segments
  if (decoded.includes('\\')) return false;
  if (decoded.split('/').some(seg => seg === '..')) return false;

  // Allowlist: exactly /dashboard, exactly /chat, or starts with /chat/
  if (decoded === '/dashboard') return true;
  if (decoded === '/chat') return true;
  if (decoded.startsWith('/chat/')) return true;

  return false;
}

function getBundleLabel(bundle: CreditBundle, index: number): string {
  if (bundle.credits >= 1000) return 'Best for teams';
  if (index === 1 || bundle.credits >= 500) return 'Best value';
  return 'Occasional use';
}

function getEstimatedMessages(bundle: CreditBundle): string {
  return `About ${bundle.credits.toLocaleString()} typical text messages`;
}

const checkoutReturnKey = 'agentstore.checkoutReturnTo';

export default function BuyCreditsPage() {
  const [bundles, setBundles] = useState<CreditBundle[]>([]);
  const [totalCredits, setTotalCredits] = useState(0);
  const [loading, setLoading] = useState(true);
  const [checkoutLoading, setCheckoutLoading] = useState<string | null>(null);
  const [searchParams] = useSearchParams();

  // Clear any stale checkoutReturnTo from a previous session on mount
  useEffect(() => {
    sessionStorage.removeItem(checkoutReturnKey);
  }, []);

  useEffect(() => {
    Promise.all([
      bundlesApi.list(),
      plansApi.list(),
    ])
      .then(([bundleData, planData]) => {
        setBundles(bundleData.bundles);
        setTotalCredits(planData.tenantSubscriptionCredits + planData.tenantPurchasedCredits);
      })
      .catch(err => toast.error(getErrorMessage(err)))
      .finally(() => setLoading(false));
  }, []);

  const handleBuy = async (bundleId: string) => {
    setCheckoutLoading(bundleId);
    const returnTo = searchParams.get('returnTo');

    try {
      const result = await billingApi.checkout({ bundleId });
      if (result.checkoutUrl) {
        // Only store returnTo after checkoutUrl is confirmed
        if (returnTo && isSafeCheckoutReturnPath(returnTo)) {
          sessionStorage.setItem(checkoutReturnKey, returnTo);
        }
        window.location.assign(result.checkoutUrl);
      } else {
        toast.error('Checkout failed. Please try again.');
      }
    } catch (err) {
      toast.error(getErrorMessage(err));
    } finally {
      setCheckoutLoading(null);
    }
  };

  if (loading) return <LoadingSpinner size="lg" className="py-20" />;

  return (
    <div className="space-y-8">
      <section className="rounded-[28px] border border-white/8 bg-dark-950/70 p-1">
        <div className="rounded-[24px] border border-primary-500/10 bg-[radial-gradient(circle_at_top_left,rgba(139,92,246,0.18),transparent_35%),linear-gradient(180deg,rgba(17,24,39,0.96),rgba(2,6,23,0.92))] px-6 py-7 sm:px-8">
          <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <div className="inline-flex items-center gap-2 rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-primary-200">
                <Sparkles className="h-3.5 w-3.5" />
                Credits
              </div>
              <h1 className="mt-4 flex items-center gap-3 text-3xl font-bold text-white">
                <Zap className="w-7 h-7 text-primary-400" />
                Buy Credits
              </h1>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-dark-300">
                Use credits to chat with Agents. Credits are charged only after a successful response.
              </p>
            </div>
            <div className="rounded-2xl border border-white/8 bg-dark-900/70 px-5 py-4">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">Current balance</p>
              <p className="mt-2 text-2xl font-semibold text-white">{totalCredits.toLocaleString()} credits</p>
            </div>
          </div>
        </div>
      </section>

      <div className="rounded-2xl border border-accent-emerald/20 bg-accent-emerald/10 p-4 text-sm text-accent-emerald">
        <div className="flex items-center gap-2 font-semibold">
          <CheckCircle className="h-4 w-4" />
          Credits are available immediately after checkout.
        </div>
      </div>

      {bundles.length === 0 ? (
        <EmptyState
          icon={Zap}
          title="No credit bundles are available right now"
          description="This workspace may currently include credits through a subscription plan only. View plans to see available usage options."
          action={{ label: 'View plans', to: '/plan' }}
        />
      ) : (
        <div className={`grid gap-6 ${
          bundles.length === 1 ? 'grid-cols-1' :
          bundles.length === 2 ? 'grid-cols-1 md:grid-cols-2' :
          'grid-cols-1 md:grid-cols-2 lg:grid-cols-3'
        }`}>
          {bundles.map((bundle, index) => (
            <div key={bundle.id} className="rounded-2xl border border-dark-800 bg-dark-900/50 p-6 transition-all hover:border-primary-500/30">
              <div className="mb-4 flex items-start justify-between gap-3">
                <div>
                  <h3 className="text-lg font-bold text-white">{bundle.name}</h3>
                  <p className="mt-1 text-sm text-dark-400">{getEstimatedMessages(bundle)}</p>
                </div>
                <span className="rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold text-primary-200">
                  {getBundleLabel(bundle, index)}
                </span>
              </div>

              <div className="mb-3 flex items-baseline gap-2">
                <Zap className="w-5 h-5 text-primary-400" />
                <span className="text-3xl font-bold text-white">{bundle.credits.toLocaleString()}</span>
                <span className="text-dark-400 text-sm">credits</span>
              </div>

              <div className="mb-6">
                <span className="text-2xl font-semibold text-primary-400">{formatPrice(bundle.priceCents)}</span>
                {bundle.credits > 0 ? (
                  <span className="text-dark-500 text-sm ml-1">({formatPrice(Math.round(bundle.priceCents / bundle.credits * 100))}/100 credits)</span>
                ) : null}
              </div>

              <button
                onClick={() => handleBuy(bundle.id)}
                disabled={checkoutLoading !== null}
                className="flex w-full items-center justify-center gap-2 rounded-xl bg-primary-500 px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-600 disabled:opacity-60"
              >
                {checkoutLoading === bundle.id ? <LoadingSpinner size="sm" /> : <ShoppingCart className="w-4 h-4" />}
                {checkoutLoading === bundle.id ? 'Starting checkout...' : `Buy ${bundle.name}`}
              </button>
            </div>
          ))}
        </div>
      )}

      <p className="text-center text-xs text-dark-500">
        Need a recurring monthly allowance? <Link to="/plan" className="text-primary-300 hover:text-primary-200">Compare plans</Link>.
      </p>
    </div>
  );
}
