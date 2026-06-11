import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { CheckCircle, CreditCard, ShoppingCart, Smartphone, Sparkles, Wallet, Zap } from 'lucide-react';
import { toast } from 'sonner';
import { bundlesApi, plansApi, billingApi } from '../../api/client';
import type { CreditBundle } from '../../types';
import LoadingSpinner from '../../components/LoadingSpinner';
import { EmptyState } from '../../components/app';
import { getErrorMessage } from '../../utils/errors';

type PaymentMethod = 'stripe' | 'wechat_h5' | 'alipay';

interface PaymentMethodOption {
  id: PaymentMethod;
  label: string;
  description: string;
  icon: React.ReactNode;
  color: string;
}

const PAYMENT_METHOD_OPTS: PaymentMethodOption[] = [
  {
    id: 'stripe',
    label: '信用卡 / 借记卡',
    description: 'Visa · Mastercard · Amex',
    icon: <CreditCard className="h-5 w-5" />,
    color: 'text-purple-400 border-purple-500/30 bg-purple-500/10',
  },
  {
    id: 'wechat_h5',
    label: '微信支付',
    description: '在手机微信内完成支付',
    icon: <Smartphone className="h-5 w-5" />,
    color: 'text-emerald-400 border-emerald-500/30 bg-emerald-500/10',
  },
  {
    id: 'alipay',
    label: '支付宝',
    description: '电脑端 / 手机端均可',
    icon: <Wallet className="h-5 w-5" />,
    color: 'text-blue-400 border-blue-500/30 bg-blue-500/10',
  },
];

function formatUSD(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

/** Approximate CNY price. Real conversion happens server-side; this is display-only. */
function formatCNY(cents: number, cnyPerUsd: number): string {
  const fen = Math.round(cents * cnyPerUsd);
  return `¥${(fen / 100).toFixed(2)}`;
}

function isSafeCheckoutReturnPath(value: string): boolean {
  if (!value.startsWith('/') || value.startsWith('//')) return false;
  const qIndex = value.indexOf('?');
  const pathname = qIndex >= 0 ? value.slice(0, qIndex) : value;
  let decoded: string;
  try { decoded = decodeURIComponent(pathname); } catch { return false; }
  if (decoded.includes('\\')) return false;
  if (decoded.split('/').some(seg => seg === '..')) return false;
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
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('stripe');
  const [enabledMethods, setEnabledMethods] = useState<string[]>(['stripe']);
  // Approximate CNY rate loaded from billing config (display only, not authoritative).
  const CNY_PER_USD = 7.2;
  const [searchParams] = useSearchParams();

  useEffect(() => {
    sessionStorage.removeItem(checkoutReturnKey);
  }, []);

  useEffect(() => {
    Promise.all([bundlesApi.list(), plansApi.list(), billingApi.getConfig()])
      .then(([bundleData, planData, cfgData]) => {
        setBundles(bundleData.bundles);
        setTotalCredits(planData.tenantSubscriptionCredits + planData.tenantPurchasedCredits);
        if (cfgData.paymentMethods?.length) {
          setEnabledMethods(cfgData.paymentMethods);
          // Default to first enabled method.
          const first = cfgData.paymentMethods[0] as PaymentMethod;
          setPaymentMethod(first);
        }
      })
      .catch(err => toast.error(getErrorMessage(err)))
      .finally(() => setLoading(false));
  }, []);

  const visibleMethods = PAYMENT_METHOD_OPTS.filter(m => enabledMethods.includes(m.id));

  const handleBuy = async (bundleId: string) => {
    setCheckoutLoading(bundleId);
    const returnTo = searchParams.get('returnTo');

    try {
      const result = await billingApi.checkout({ bundleId, paymentMethod });
      if (result.waived) {
        toast.success('Credits added!');
        return;
      }
      if (result.checkoutUrl) {
        // For domestic payments, store outTradeNo alongside the returnTo key.
        if (returnTo && isSafeCheckoutReturnPath(returnTo)) {
          sessionStorage.setItem(checkoutReturnKey, returnTo);
        }
        if (result.outTradeNo) {
          sessionStorage.setItem('agentstore.outTradeNo', result.outTradeNo);
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

  const isDomestic = paymentMethod === 'wechat_h5' || paymentMethod === 'alipay';

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

      {/* Payment method selector — only shown when >1 method is configured */}
      {visibleMethods.length > 1 && (
        <div className="space-y-3">
          <p className="text-sm font-semibold text-dark-300">支付方式</p>
          <div className={`grid gap-3 ${visibleMethods.length === 2 ? 'grid-cols-2' : 'grid-cols-3'}`}>
            {visibleMethods.map(m => (
              <button
                key={m.id}
                onClick={() => setPaymentMethod(m.id)}
                className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-all ${
                  paymentMethod === m.id
                    ? m.color + ' ring-1 ring-current/30'
                    : 'border-dark-800 bg-dark-900/50 text-dark-400 hover:border-dark-700'
                }`}
              >
                <span className="shrink-0">{m.icon}</span>
                <span>
                  <span className="block text-sm font-semibold">{m.label}</span>
                  <span className="block text-xs opacity-70">{m.description}</span>
                </span>
              </button>
            ))}
          </div>
        </div>
      )}

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
                {isDomestic ? (
                  <div>
                    <span className="text-2xl font-semibold text-primary-400">
                      {formatCNY(bundle.priceCents, CNY_PER_USD)}
                    </span>
                    <span className="text-dark-500 text-xs ml-1 opacity-60">≈ {formatUSD(bundle.priceCents)}</span>
                  </div>
                ) : (
                  <div>
                    <span className="text-2xl font-semibold text-primary-400">{formatUSD(bundle.priceCents)}</span>
                    {bundle.credits > 0 ? (
                      <span className="text-dark-500 text-sm ml-1">
                        ({formatUSD(Math.round(bundle.priceCents / bundle.credits * 100))}/100 credits)
                      </span>
                    ) : null}
                  </div>
                )}
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

      {isDomestic && (
        <p className="text-center text-xs text-dark-500">
          人民币金额为参考价格，实际以支付时汇率为准。
        </p>
      )}

      <p className="text-center text-xs text-dark-500">
        Need a recurring monthly allowance? <Link to="/plan" className="text-primary-300 hover:text-primary-200">Compare plans</Link>.
      </p>
    </div>
  );
}
