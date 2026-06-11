import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowRight, CheckCircle, Clock, MessageCircle, Receipt, Sparkles } from 'lucide-react';
import { billingApi } from '../../api/client';

const checkoutReturnKey = 'agentstore.checkoutReturnTo';
const outTradeNoKey = 'agentstore.outTradeNo';

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

export default function BillingSuccessPage() {
  const navigate = useNavigate();

  const returnTo = useMemo(() => {
    const value = sessionStorage.getItem(checkoutReturnKey);
    sessionStorage.removeItem(checkoutReturnKey);
    return value;
  }, []);
  const safeReturnTo = returnTo && isSafeCheckoutReturnPath(returnTo) ? returnTo : '';

  // Domestic payment (WeChat/Alipay) polling state.
  const outTradeNo = useMemo(() => {
    const v = sessionStorage.getItem(outTradeNoKey);
    // Keep key until confirmed, removed once confirmed or timed out.
    return v;
  }, []);

  const [pollStatus, setPollStatus] = useState<'polling' | 'confirmed' | 'timeout'>(() =>
    outTradeNo ? 'polling' : 'confirmed',
  );
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const attemptRef = useRef(0);
  const MAX_ATTEMPTS = 30; // 30 × 2 s = 60 s

  useEffect(() => {
    if (!outTradeNo) return;

    const poll = async () => {
      attemptRef.current += 1;
      try {
        const res = await billingApi.getPaymentStatus(outTradeNo);
        if (res.status === 'completed') {
          sessionStorage.removeItem(outTradeNoKey);
          setPollStatus('confirmed');
          if (pollRef.current) clearInterval(pollRef.current);
        } else if (res.status === 'failed' || attemptRef.current >= MAX_ATTEMPTS) {
          sessionStorage.removeItem(outTradeNoKey);
          setPollStatus('timeout');
          if (pollRef.current) clearInterval(pollRef.current);
        }
      } catch {
        if (attemptRef.current >= MAX_ATTEMPTS) {
          sessionStorage.removeItem(outTradeNoKey);
          setPollStatus('timeout');
          if (pollRef.current) clearInterval(pollRef.current);
        }
      }
    };

    // Poll immediately, then every 2 s.
    poll();
    pollRef.current = setInterval(poll, 2000);
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [outTradeNo]);

  // Auto-redirect for Stripe success (no polling needed).
  useEffect(() => {
    if (pollStatus !== 'confirmed' || safeReturnTo) return;
    const timer = setTimeout(() => navigate('/plan'), 5000);
    return () => clearTimeout(timer);
  }, [navigate, safeReturnTo, pollStatus]);

  // Still waiting for payment confirmation.
  if (pollStatus === 'polling') {
    return (
      <div className="mx-auto flex max-w-2xl flex-col items-center justify-center py-20 text-center">
        <div className="flex h-20 w-20 items-center justify-center rounded-3xl border border-yellow-500/20 bg-yellow-500/10 text-yellow-400 mx-auto">
          <Clock className="h-10 w-10 animate-pulse" />
        </div>
        <div className="mt-6 inline-flex items-center gap-2 rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-primary-200">
          <Sparkles className="h-3.5 w-3.5" />
          Payment processing
        </div>
        <h1 className="mt-4 text-3xl font-bold text-white">等待支付确认中…</h1>
        <p className="mt-3 max-w-lg text-sm leading-6 text-dark-400">
          请完成支付后稍等片刻，积分将在确认后立即到账。
        </p>
        <div className="mt-8 flex gap-2">
          <span className="inline-block h-2 w-2 animate-bounce rounded-full bg-primary-400 [animation-delay:-0.3s]" />
          <span className="inline-block h-2 w-2 animate-bounce rounded-full bg-primary-400 [animation-delay:-0.15s]" />
          <span className="inline-block h-2 w-2 animate-bounce rounded-full bg-primary-400" />
        </div>
      </div>
    );
  }

  // Payment timed out or failed.
  if (pollStatus === 'timeout') {
    return (
      <div className="mx-auto flex max-w-2xl flex-col items-center justify-center py-20 text-center">
        <div className="flex h-20 w-20 items-center justify-center rounded-3xl border border-red-500/20 bg-red-500/10 text-red-400 mx-auto">
          <Clock className="h-10 w-10" />
        </div>
        <h1 className="mt-6 text-3xl font-bold text-white">未能确认支付状态</h1>
        <p className="mt-3 max-w-lg text-sm leading-6 text-dark-400">
          如果已完成付款，积分通常会在几分钟内自动到账。如有疑问请联系客服。
        </p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <Link
            to="/buy-credits"
            className="inline-flex items-center justify-center gap-2 rounded-xl border border-dark-700 bg-dark-900 px-5 py-3 text-sm font-semibold text-dark-100 transition-colors hover:bg-dark-800"
          >
            重新购买
          </Link>
          <Link
            to="/dashboard"
            className="inline-flex items-center justify-center gap-2 rounded-xl border border-dark-700 bg-dark-900 px-5 py-3 text-sm font-semibold text-dark-100 transition-colors hover:bg-dark-800"
          >
            <MessageCircle className="h-4 w-4" />
            Browse Agents
          </Link>
        </div>
      </div>
    );
  }

  // Payment confirmed (or Stripe success).
  return (
    <div className="mx-auto flex max-w-2xl flex-col items-center justify-center py-20 text-center">
      <div role="status" aria-live="polite">
        <div className="flex h-20 w-20 items-center justify-center rounded-3xl border border-accent-emerald/20 bg-accent-emerald/10 text-accent-emerald mx-auto">
          <CheckCircle className="h-10 w-10" />
        </div>
        <div className="mt-6 inline-flex items-center gap-2 rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-primary-200">
          <Sparkles className="h-3.5 w-3.5" />
          Payment complete
        </div>
        <h1 className="mt-4 text-3xl font-bold text-white">Credits added successfully.</h1>
        <p className="mt-3 max-w-lg text-sm leading-6 text-dark-400">
          Your credits are available now. Continue your previous conversation or browse Agents to start a new task.
        </p>
      </div>

      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        {safeReturnTo ? (
          <Link
            to={safeReturnTo}
            className="inline-flex items-center justify-center gap-2 rounded-xl bg-primary-500 px-5 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-600"
          >
            Continue conversation
            <ArrowRight className="h-4 w-4" />
          </Link>
        ) : null}
        <Link
          to="/dashboard"
          className="inline-flex items-center justify-center gap-2 rounded-xl border border-dark-700 bg-dark-900 px-5 py-3 text-sm font-semibold text-dark-100 transition-colors hover:bg-dark-800"
        >
          <MessageCircle className="h-4 w-4" />
          Browse Agents
        </Link>
        <Link
          to="/plan"
          className="inline-flex items-center justify-center gap-2 rounded-xl border border-dark-700 bg-dark-900 px-5 py-3 text-sm font-semibold text-dark-100 transition-colors hover:bg-dark-800"
        >
          <Receipt className="h-4 w-4" />
          View billing
        </Link>
      </div>

      {!safeReturnTo && pollStatus === 'confirmed' && !outTradeNo ? (
        <p className="mt-5 text-xs text-dark-500">Redirecting to billing in a few seconds...</p>
      ) : null}
    </div>
  );
}
