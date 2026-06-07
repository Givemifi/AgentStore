import { useEffect, useMemo } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowRight, CheckCircle, MessageCircle, Receipt, Sparkles } from 'lucide-react';

const checkoutReturnKey = 'agentstore.checkoutReturnTo';

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

export default function BillingSuccessPage() {
  const navigate = useNavigate();

  // Read initial value into state/memo and clear key on mount to prevent stale paths
  const returnTo = useMemo(() => {
    const value = sessionStorage.getItem(checkoutReturnKey);
    sessionStorage.removeItem(checkoutReturnKey);
    return value;
  }, []);
  const safeReturnTo = returnTo && isSafeCheckoutReturnPath(returnTo) ? returnTo : '';

  useEffect(() => {
    if (safeReturnTo) {
      return undefined;
    }
    const timer = setTimeout(() => navigate('/plan'), 5000);
    return () => clearTimeout(timer);
  }, [navigate, safeReturnTo]);

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

      {!safeReturnTo ? (
        <p className="mt-5 text-xs text-dark-500">Redirecting to billing in a few seconds...</p>
      ) : null}
    </div>
  );
}
