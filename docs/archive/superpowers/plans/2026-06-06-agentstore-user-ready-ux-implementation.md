# AgentStore User-Ready UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the approved AgentStore user-ready UX design into a P0 product polish pass where ordinary users can discover Agents, understand credits, chat successfully, buy credits, and operators can see launch readiness.

**Architecture:** Keep the existing Go + React architecture and improve the user experience through focused page/component changes rather than a rewrite. Add a small shared frontend UX layer, then update marketplace, chat, credits, onboarding, admin launch readiness, and capability-honesty surfaces. Backend work is limited to making provider testing honest for OpenAI-compatible providers while preserving existing enum/schema compatibility.

**Tech Stack:** Go 1.25, Gorilla Mux, MongoDB Go driver, React 19, Vite 7, TypeScript, TanStack Query, React Router 7, Tailwind CSS 4, Vitest, Testing Library, Playwright.

---

## Scope and execution notes

- Do not add creator marketplace, revenue sharing, public Agent pages, ratings/reviews, true multimodal generation, or full Anthropic/Gemini execution in this P0 pass.
- Do not remove backend enum/schema support for Anthropic/Gemini, image/video models, or image/video Agent capabilities. Existing data and API clients must keep working.
- P0 product promise: ordinary users see only usable text-chat capabilities; operator/admin surfaces explain unsupported capabilities as coming soon or OpenAI-compatible-only.
- Do not commit unless the user explicitly asks for commits. Each task has a checkpoint command section that can become a commit step only after explicit user approval.
- Because the user authorized multi-agent work, execute with `superpowers:subagent-driven-development`: one fresh subagent per task, review between tasks. Run tasks in parallel only where the file map below says they do not conflict.
- Existing local Bash output failed once because the Claude temp task filesystem was full. If Bash verification fails with `ENOSPC`, stop and ask the user to clear or relocate Claude Code temp output before claiming verification.

## File structure map

### Shared frontend UX components

- Create `frontend/src/components/app/EmptyState.tsx` — role-aware empty-state card with icon, title, description, and optional CTA.
- Create `frontend/src/components/app/ErrorState.tsx` — retryable error card/banner with consistent copy and actions.
- Create `frontend/src/components/app/CreditExplainer.tsx` — compact reusable credit balance/cost/safety explanation.
- Create `frontend/src/components/app/MobileBottomNav.tsx` — ordinary-user mobile bottom navigation for Agents, Credits, History, Settings, optional Admin under More/Settings later.
- Create `frontend/src/components/app/MarkdownMessage.tsx` — safe Markdown-ish renderer for assistant messages without adding a dependency.
- Create `frontend/src/components/app/index.ts` — exports shared app UX components.
- Create tests in `frontend/src/components/app/*.test.tsx` for these components.

### Ordinary user app surfaces

- Modify `frontend/src/components/Layout.tsx` — add mobile bottom nav and bottom padding, keep desktop nav unchanged.
- Create `frontend/src/components/Layout.test.tsx` — test default nav, Admin visibility, credit routing, announcement dismissal, tenant switching.
- Modify `frontend/src/pages/app/DashboardPage.tsx` — marketplace hero, search, category chips, credit safety, stronger Agent cards, role-aware empty state, hide ordinary-user image/video labels.
- Modify `frontend/src/pages/app/DashboardPage.test.tsx` — update copy assertions and add search/filter/navigation/capability-honesty tests.
- Modify `frontend/src/pages/app/ChatPage.tsx` — Agent terminology, Markdown rendering, prompt query support, cost notice, retry states, mobile conversation drawer.
- Modify `frontend/src/pages/app/ChatPage.test.tsx` — update terminology and add Markdown, prompt-query, stream-error, retry, and mobile/sidebar tests.
- Modify `frontend/src/pages/app/BuyCreditsPage.tsx` — educate credits, show estimated messages, set checkout return path, improve empty/error/loading states.
- Create `frontend/src/pages/app/BuyCreditsPage.test.tsx` — test balance, bundles, empty state, checkout redirect/return path.
- Modify `frontend/src/pages/app/BillingSuccessPage.tsx` — show success actions and continue-chat path instead of only auto-redirecting.
- Create `frontend/src/pages/app/BillingSuccessPage.test.tsx` — test success actions and fallback redirect.
- Modify `frontend/src/pages/app/OnboardingPage.tsx` — first-success onboarding: goal, recommended Agent, first prompt, credits explanation, then chat.
- Create `frontend/src/pages/app/OnboardingPage.test.tsx` — test onboarding happy path, no-Agent fallback, completion failure fallback.

### Operator/admin surfaces

- Create `frontend/src/pages/admin/components/LaunchChecklist.tsx` — admin launch checklist with statuses and links.
- Create `frontend/src/pages/admin/components/LaunchChecklist.test.tsx` — test checklist statuses and links.
- Modify `frontend/src/pages/admin/DashboardPage.tsx` — fetch checklist data and render Launch Checklist above metrics.
- Create or extend `frontend/src/pages/admin/DashboardPage.test.tsx` — test dashboard metrics and checklist visibility.

### Capability honesty surfaces

- Modify `frontend/src/pages/app/settings/ModelSettingsTab.tsx` — OpenAI-compatible-only P0 copy, disabled/coming-soon non-P0 provider labels for existing data, text-only new model creation, honest provider test result language.
- Modify `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx` — test provider labels, text-only modality creation, existing non-P0 providers render as coming soon.
- Modify `frontend/src/pages/app/settings/AgentsTab.tsx` — text-chat-only creation controls, image/video coming-soon labels for existing agents, hide image/video credit inputs from P0 creation.
- Modify `frontend/src/pages/app/settings/AgentsTab.test.tsx` — test image/video cannot be newly selected and existing image/video capabilities are labeled coming soon.
- Modify `backend/internal/llm/openai.go` — add OpenAI-compatible provider connectivity test helper.
- Modify `backend/internal/api/handlers/model_settings.go` — make `TestProvider` call the connectivity helper for OpenAI-compatible providers and return honest unsupported status for Anthropic/Gemini.
- Modify `backend/internal/api/handlers/model_settings_test.go` — add provider connectivity test success/failure/unsupported cases.

### Final verification and docs

- Modify `README.md` only if the implementation changes visible product positioning text there in this pass. Otherwise leave README for a later docs-only pass.
- Modify `docs/smoke-test-real-mongo-llm-billing.md` only if checkout return behavior or provider-test behavior changes the smoke path.
- Run backend and frontend verification listed in Task 10.

## Parallelization map

1. **Task 1 is serial foundation.** Do this first because later tasks import shared components.
2. After Task 1, these can run in parallel because they touch separate files:
   - Task 2 Layout mobile nav
   - Task 4 Chat UX
   - Task 5 Credits/Billing UX
   - Task 6 Onboarding
   - Task 7 Admin Launch Checklist
   - Task 8 Backend provider test
3. **Task 3 Dashboard Marketplace** should not run in parallel with any other task that modifies `DashboardPage.tsx`. In this plan, Task 3 owns that file.
4. **Task 9 Model/Agent capability honesty** should run after Task 8 if it displays provider-test language, but it can run in parallel with Dashboard/Chat/Credits/Onboarding/Admin if those agents do not modify settings files.
5. **Task 10 Final verification** is serial and runs after all implementation tasks are merged.

---

## Task 1: Shared UX Components Foundation

**Files:**
- Create: `frontend/src/components/app/EmptyState.tsx`
- Create: `frontend/src/components/app/ErrorState.tsx`
- Create: `frontend/src/components/app/CreditExplainer.tsx`
- Create: `frontend/src/components/app/MobileBottomNav.tsx`
- Create: `frontend/src/components/app/MarkdownMessage.tsx`
- Create: `frontend/src/components/app/index.ts`
- Test: `frontend/src/components/app/EmptyState.test.tsx`
- Test: `frontend/src/components/app/ErrorState.test.tsx`
- Test: `frontend/src/components/app/CreditExplainer.test.tsx`
- Test: `frontend/src/components/app/MobileBottomNav.test.tsx`
- Test: `frontend/src/components/app/MarkdownMessage.test.tsx`

- [ ] **Step 1: Write failing tests for shared components**

Create `frontend/src/components/app/EmptyState.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import { MessageCircle } from 'lucide-react';
import { describe, expect, it, vi } from 'vitest';
import EmptyState from './EmptyState';

vi.mock('react-router-dom', () => ({
  Link: ({ to, children, className }: { to: string; children: React.ReactNode; className?: string }) => (
    <a href={to} className={className}>{children}</a>
  ),
}));

describe('EmptyState', () => {
  it('renders title, description, icon, and link action', () => {
    render(
      <EmptyState
        icon={MessageCircle}
        title="No agents yet"
        description="Ask an administrator to publish your first Agent."
        action={{ label: 'Browse plans', to: '/plan' }}
      />
    );

    expect(screen.getByText('No agents yet')).toBeInTheDocument();
    expect(screen.getByText('Ask an administrator to publish your first Agent.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Browse plans' })).toHaveAttribute('href', '/plan');
  });

  it('renders button action when onClick is provided', () => {
    const onClick = vi.fn();
    render(
      <EmptyState
        icon={MessageCircle}
        title="No conversations"
        description="Start a new chat to create your first thread."
        action={{ label: 'Start chat', onClick }}
      />
    );

    screen.getByRole('button', { name: 'Start chat' }).click();
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
```

Create `frontend/src/components/app/ErrorState.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import ErrorState from './ErrorState';

describe('ErrorState', () => {
  it('renders a retry action when provided', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();

    render(
      <ErrorState
        title="Unable to load Agents"
        message="Network unavailable"
        retryLabel="Try again"
        onRetry={onRetry}
      />
    );

    expect(screen.getByText('Unable to load Agents')).toBeInTheDocument();
    expect(screen.getByText('Network unavailable')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('disables retry while retrying', () => {
    render(
      <ErrorState
        title="Unable to load Agents"
        message="Retrying"
        retryLabel="Try again"
        onRetry={() => undefined}
        isRetrying
      />
    );

    expect(screen.getByRole('button', { name: 'Retrying...' })).toBeDisabled();
  });
});
```

Create `frontend/src/components/app/CreditExplainer.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import CreditExplainer from './CreditExplainer';

describe('CreditExplainer', () => {
  it('renders balance, per-message cost, and safety copy', () => {
    render(<CreditExplainer balance={1200} perMessageCost={3} />);

    expect(screen.getByText('1,200 credits available')).toBeInTheDocument();
    expect(screen.getByText('Most selected Agents cost 3 credits/message.')).toBeInTheDocument();
    expect(screen.getByText('Credits are charged only after a successful response.')).toBeInTheDocument();
  });

  it('renders loading state when balance is unknown', () => {
    render(<CreditExplainer balance={null} perMessageCost={1} />);

    expect(screen.getByText('Credits will appear when usage data is available.')).toBeInTheDocument();
  });
});
```

Create `frontend/src/components/app/MobileBottomNav.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import MobileBottomNav from './MobileBottomNav';

function renderNav(path = '/dashboard') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <MobileBottomNav hasAdminAccess={false} />
    </MemoryRouter>
  );
}

describe('MobileBottomNav', () => {
  it('renders ordinary user mobile nav links', () => {
    renderNav('/dashboard');

    expect(screen.getByRole('link', { name: 'Agents' })).toHaveAttribute('href', '/dashboard');
    expect(screen.getByRole('link', { name: 'Credits' })).toHaveAttribute('href', '/buy-credits');
    expect(screen.getByRole('link', { name: 'History' })).toHaveAttribute('href', '/activity');
    expect(screen.getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings');
  });

  it('shows admin link for platform admins', () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <MobileBottomNav hasAdminAccess />
      </MemoryRouter>
    );

    expect(screen.getByRole('link', { name: 'Admin' })).toHaveAttribute('href', '/admin');
  });
});
```

Create `frontend/src/components/app/MarkdownMessage.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import MarkdownMessage from './MarkdownMessage';

describe('MarkdownMessage', () => {
  it('renders headings, bullets, numbered items, links, inline code, and fenced code', () => {
    render(
      <MarkdownMessage
        content={`## Plan\n- First step\n1. Numbered step\nUse \`credits\` safely.\n[Open docs](https://example.com)\n\`\`\`ts\nconst ok = true;\n\`\`\``}
      />
    );

    expect(screen.getByRole('heading', { name: 'Plan' })).toBeInTheDocument();
    expect(screen.getByText('First step')).toBeInTheDocument();
    expect(screen.getByText('Numbered step')).toBeInTheDocument();
    expect(screen.getByText('credits')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open docs' })).toHaveAttribute('href', 'https://example.com');
    expect(screen.getByText('const ok = true;')).toBeInTheDocument();
  });

  it('does not render raw html as html', () => {
    render(<MarkdownMessage content={'<img src=x onerror=alert(1)> plain text'} />);

    expect(screen.getByText('<img src=x onerror=alert(1)> plain text')).toBeInTheDocument();
    expect(document.querySelector('img')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run tests and verify they fail before implementation**

Run:

```bash
cd frontend && npm test -- --run \
  src/components/app/EmptyState.test.tsx \
  src/components/app/ErrorState.test.tsx \
  src/components/app/CreditExplainer.test.tsx \
  src/components/app/MobileBottomNav.test.tsx \
  src/components/app/MarkdownMessage.test.tsx
```

Expected: FAIL because the new component files do not exist.

- [ ] **Step 3: Implement shared components**

Create `frontend/src/components/app/EmptyState.tsx`:

```tsx
import { Link } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';

type EmptyStateAction =
  | { label: string; to: string; onClick?: never }
  | { label: string; onClick: () => void; to?: never };

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: EmptyStateAction;
  className?: string;
}

export default function EmptyState({ icon: Icon, title, description, action, className = '' }: EmptyStateProps) {
  const actionClassName = 'mt-6 inline-flex items-center justify-center rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600';

  return (
    <div className={`rounded-[26px] border border-dashed border-white/10 bg-dark-950/70 px-6 py-14 text-center ${className}`}>
      <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-white/8 bg-dark-900/80 text-primary-300">
        <Icon className="h-6 w-6" />
      </div>
      <h2 className="mt-5 text-xl font-semibold text-white">{title}</h2>
      <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-dark-400">{description}</p>
      {action && 'to' in action ? (
        <Link to={action.to} className={actionClassName}>{action.label}</Link>
      ) : action ? (
        <button type="button" onClick={action.onClick} className={actionClassName}>{action.label}</button>
      ) : null}
    </div>
  );
}
```

Create `frontend/src/components/app/ErrorState.tsx`:

```tsx
import { AlertCircle } from 'lucide-react';

interface ErrorStateProps {
  title: string;
  message: string;
  retryLabel?: string;
  onRetry?: () => void;
  isRetrying?: boolean;
  className?: string;
}

export default function ErrorState({
  title,
  message,
  retryLabel = 'Retry',
  onRetry,
  isRetrying = false,
  className = '',
}: ErrorStateProps) {
  return (
    <div className={`flex flex-col gap-4 rounded-2xl border border-red-500/25 bg-red-500/10 p-4 text-sm text-red-200 sm:flex-row sm:items-center sm:justify-between ${className}`}>
      <div className="flex gap-3">
        <AlertCircle className="mt-0.5 h-5 w-5 flex-shrink-0 text-red-300" />
        <div>
          <p className="font-semibold text-red-100">{title}</p>
          <p className="mt-1 text-red-200/80">{message}</p>
        </div>
      </div>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          disabled={isRetrying}
          className="inline-flex items-center justify-center rounded-xl border border-red-400/30 bg-red-500/10 px-4 py-2 font-semibold text-red-100 transition-colors hover:bg-red-500/20 disabled:cursor-not-allowed disabled:opacity-70"
        >
          {isRetrying ? 'Retrying...' : retryLabel}
        </button>
      ) : null}
    </div>
  );
}
```

Create `frontend/src/components/app/CreditExplainer.tsx`:

```tsx
import { ShieldCheck, Zap } from 'lucide-react';

interface CreditExplainerProps {
  balance: number | null;
  perMessageCost?: number;
  className?: string;
}

export default function CreditExplainer({ balance, perMessageCost = 1, className = '' }: CreditExplainerProps) {
  return (
    <div className={`rounded-2xl border border-white/8 bg-dark-900/70 p-5 backdrop-blur-sm ${className}`}>
      <div className="flex items-start gap-4">
        <div className="rounded-2xl border border-primary-400/15 bg-primary-500/10 p-3 text-primary-200">
          <Zap className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-dark-400">Credits</p>
          {balance === null ? (
            <p className="mt-3 text-sm text-dark-400">Credits will appear when usage data is available.</p>
          ) : (
            <p className="mt-3 text-2xl font-semibold text-white">{balance.toLocaleString()} credits available</p>
          )}
          <p className="mt-2 text-sm text-dark-300">Most selected Agents cost {perMessageCost} credits/message.</p>
          <p className="mt-2 inline-flex items-center gap-1.5 text-xs text-accent-emerald">
            <ShieldCheck className="h-3.5 w-3.5" />
            Credits are charged only after a successful response.
          </p>
        </div>
      </div>
    </div>
  );
}
```

Create `frontend/src/components/app/MobileBottomNav.tsx`:

```tsx
import { Link, useLocation } from 'react-router-dom';
import { Activity, Bot, CreditCard, Settings, Shield } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

interface MobileBottomNavProps {
  hasAdminAccess: boolean;
}

interface MobileNavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  match: (pathname: string) => boolean;
}

export default function MobileBottomNav({ hasAdminAccess }: MobileBottomNavProps) {
  const location = useLocation();
  const items: MobileNavItem[] = [
    { to: '/dashboard', label: 'Agents', icon: Bot, match: (path) => path === '/dashboard' || path.startsWith('/chat') },
    { to: '/buy-credits', label: 'Credits', icon: CreditCard, match: (path) => path === '/buy-credits' || path === '/plan' || path.startsWith('/billing') },
    { to: '/activity', label: 'History', icon: Activity, match: (path) => path === '/activity' },
    { to: '/settings', label: 'Settings', icon: Settings, match: (path) => path.startsWith('/settings') },
  ];

  if (hasAdminAccess) {
    items.push({ to: '/admin', label: 'Admin', icon: Shield, match: (path) => path.startsWith('/admin') });
  }

  return (
    <nav className="fixed inset-x-0 bottom-0 z-50 border-t border-dark-800 bg-dark-900/95 px-2 py-2 shadow-[0_-12px_40px_-24px_rgba(0,0,0,0.95)] backdrop-blur-xl md:hidden" aria-label="Primary mobile navigation">
      <div className="mx-auto grid max-w-md grid-cols-4 gap-1">
        {items.slice(0, 4).map((item) => {
          const active = item.match(location.pathname);
          return (
            <Link
              key={item.to}
              to={item.to}
              className={`flex flex-col items-center justify-center gap-1 rounded-2xl px-2 py-2 text-xs font-medium transition-colors ${
                active ? 'bg-primary-500/15 text-primary-300' : 'text-dark-400 hover:bg-dark-800 hover:text-white'
              }`}
            >
              <item.icon className="h-5 w-5" />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
```

Create `frontend/src/components/app/MarkdownMessage.tsx`:

```tsx
import type { ReactNode } from 'react';

interface MarkdownMessageProps {
  content: string;
}

function renderInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  const pattern = /(\[([^\]]+)\]\((https?:\/\/[^\s)]+)\))|(`([^`]+)`)/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > lastIndex) {
      nodes.push(text.slice(lastIndex, match.index));
    }

    if (match[2] && match[3]) {
      nodes.push(
        <a
          key={`link-${match.index}`}
          href={match[3]}
          target="_blank"
          rel="noreferrer"
          className="text-primary-300 underline decoration-primary-300/40 underline-offset-2 hover:text-primary-200"
        >
          {match[2]}
        </a>
      );
    } else if (match[5]) {
      nodes.push(
        <code key={`code-${match.index}`} className="rounded bg-dark-950 px-1.5 py-0.5 font-mono text-[0.9em] text-primary-200">
          {match[5]}
        </code>
      );
    }

    lastIndex = pattern.lastIndex;
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex));
  }

  return nodes;
}

export default function MarkdownMessage({ content }: MarkdownMessageProps) {
  const lines = content.split('\n');
  const blocks: ReactNode[] = [];
  let index = 0;

  while (index < lines.length) {
    const line = lines[index];

    if (line.trim().startsWith('```')) {
      const language = line.trim().slice(3).trim();
      const codeLines: string[] = [];
      index += 1;
      while (index < lines.length && !lines[index].trim().startsWith('```')) {
        codeLines.push(lines[index]);
        index += 1;
      }
      blocks.push(
        <pre key={`code-${index}`} className="my-3 overflow-x-auto rounded-2xl border border-dark-700 bg-dark-950 p-4 text-sm text-dark-100">
          {language ? <div className="mb-2 text-xs uppercase tracking-[0.18em] text-dark-500">{language}</div> : null}
          <code>{codeLines.join('\n')}</code>
        </pre>
      );
      index += 1;
      continue;
    }

    if (line.startsWith('### ')) {
      blocks.push(<h3 key={index} className="mt-4 text-base font-semibold text-white">{renderInline(line.slice(4))}</h3>);
    } else if (line.startsWith('## ')) {
      blocks.push(<h2 key={index} className="mt-4 text-lg font-semibold text-white">{renderInline(line.slice(3))}</h2>);
    } else if (line.startsWith('# ')) {
      blocks.push(<h1 key={index} className="mt-4 text-xl font-semibold text-white">{renderInline(line.slice(2))}</h1>);
    } else if (/^[-*]\s+/.test(line)) {
      const items: string[] = [];
      while (index < lines.length && /^[-*]\s+/.test(lines[index])) {
        items.push(lines[index].replace(/^[-*]\s+/, ''));
        index += 1;
      }
      blocks.push(
        <ul key={`ul-${index}`} className="my-3 list-disc space-y-1 pl-5 text-sm leading-6">
          {items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item)}</li>)}
        </ul>
      );
      continue;
    } else if (/^\d+\.\s+/.test(line)) {
      const items: string[] = [];
      while (index < lines.length && /^\d+\.\s+/.test(lines[index])) {
        items.push(lines[index].replace(/^\d+\.\s+/, ''));
        index += 1;
      }
      blocks.push(
        <ol key={`ol-${index}`} className="my-3 list-decimal space-y-1 pl-5 text-sm leading-6">
          {items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item)}</li>)}
        </ol>
      );
      continue;
    } else if (line.startsWith('> ')) {
      blocks.push(<blockquote key={index} className="my-3 border-l-2 border-primary-400/60 pl-3 text-sm italic text-dark-300">{renderInline(line.slice(2))}</blockquote>);
    } else if (line.trim() === '') {
      blocks.push(<div key={index} className="h-2" />);
    } else {
      blocks.push(<p key={index} className="text-sm leading-6">{renderInline(line)}</p>);
    }

    index += 1;
  }

  return <div className="space-y-1 break-words">{blocks}</div>;
}
```

Create `frontend/src/components/app/index.ts`:

```ts
export { default as CreditExplainer } from './CreditExplainer';
export { default as EmptyState } from './EmptyState';
export { default as ErrorState } from './ErrorState';
export { default as MarkdownMessage } from './MarkdownMessage';
export { default as MobileBottomNav } from './MobileBottomNav';
```

- [ ] **Step 4: Run shared component tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run \
  src/components/app/EmptyState.test.tsx \
  src/components/app/ErrorState.test.tsx \
  src/components/app/CreditExplainer.test.tsx \
  src/components/app/MobileBottomNav.test.tsx \
  src/components/app/MarkdownMessage.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Type-check shared components**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS, or only pre-existing unrelated errors. If errors are from new files, fix them before moving on.

- [ ] **Step 6: Checkpoint**

Do not commit unless the user explicitly asks. If asked to commit this checkpoint, run:

```bash
git add frontend/src/components/app
git commit -m "feat: add shared user-ready UX components"
```

---

## Task 2: Main Layout Mobile Navigation

**Files:**
- Modify: `frontend/src/components/Layout.tsx`
- Create: `frontend/src/components/Layout.test.tsx`
- Depends on: Task 1 (`MobileBottomNav`)

- [ ] **Step 1: Write failing Layout tests**

Create `frontend/src/components/Layout.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Layout from './Layout';

const apiMocks = vi.hoisted(() => ({
  unreadCount: vi.fn(),
  listPlans: vi.fn(),
  listBundles: vi.fn(),
  listAnnouncements: vi.fn(),
  logout: vi.fn(),
  setActiveTenant: vi.fn(),
}));

const authState = vi.hoisted(() => ({
  memberships: [
    { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false },
  ],
}));

const tenantState = vi.hoisted(() => ({
  activeTenant: { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false },
}));

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { displayName: 'Ada Lovelace' },
    isAuthenticated: true,
    logout: apiMocks.logout,
    memberships: authState.memberships,
  }),
}));

vi.mock('../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: tenantState.activeTenant,
    setActiveTenant: apiMocks.setActiveTenant,
  }),
}));

vi.mock('../contexts/BrandingContext', () => ({
  useBranding: () => ({
    branding: {
      appName: 'AgentStore',
      logoMode: 'text',
      logoUrl: '',
      navItems: [],
    },
  }),
}));

vi.mock('../contexts/ThemeContext', () => ({
  useTheme: () => ({
    resolvedTheme: 'dark',
    setTheme: vi.fn(),
  }),
}));

vi.mock('../api/client', () => ({
  messagesApi: { unreadCount: apiMocks.unreadCount },
  plansApi: { list: apiMocks.listPlans },
  bundlesApi: { list: apiMocks.listBundles },
  announcementsApi: { list: apiMocks.listAnnouncements },
}));

vi.mock('./ImpersonationBanner', () => ({ default: () => null }));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderLayout(initialEntry = '/dashboard') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route path="dashboard" element={<><div>Dashboard content</div><LocationDisplay /></>} />
            <Route path="buy-credits" element={<><div>Buy credits content</div><LocationDisplay /></>} />
            <Route path="plan" element={<><div>Plan content</div><LocationDisplay /></>} />
            <Route path="activity" element={<><div>Activity content</div><LocationDisplay /></>} />
            <Route path="settings" element={<><div>Settings content</div><LocationDisplay /></>} />
            <Route path="admin" element={<><div>Admin content</div><LocationDisplay /></>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('Layout', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    authState.memberships = [{ tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false }];
    tenantState.activeTenant = authState.memberships[0];
    apiMocks.unreadCount.mockResolvedValue({ count: 0 });
    apiMocks.listPlans.mockResolvedValue({ plans: [], tenantSubscriptionCredits: 10, tenantPurchasedCredits: 5, maxPlanUserLimit: 5 });
    apiMocks.listBundles.mockResolvedValue({ bundles: [{ id: 'bundle-1' }] });
    apiMocks.listAnnouncements.mockResolvedValue({ announcements: [] });
  });

  it('renders desktop and mobile ordinary-user navigation', async () => {
    renderLayout('/dashboard');

    expect(await screen.findByText('Dashboard content')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: 'Agents' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: 'Credits' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: 'History' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: 'Settings' }).length).toBeGreaterThan(0);
  });

  it('shows admin navigation only for root memberships', async () => {
    const { rerender } = renderLayout('/dashboard');
    await screen.findByText('Dashboard content');
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();

    authState.memberships = [{ tenantId: 'root-tenant', tenantName: 'Root', tenantSlug: 'root', role: 'owner', isRoot: true }];
    tenantState.activeTenant = authState.memberships[0];

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter initialEntries={['/dashboard']}>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route path="dashboard" element={<><div>Dashboard content</div><LocationDisplay /></>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );

    expect(await screen.findByRole('link', { name: 'Admin' })).toHaveAttribute('href', '/admin');
  });

  it('routes the credits indicator to buy credits when bundles exist', async () => {
    const user = userEvent.setup();
    renderLayout('/dashboard');

    const creditButton = await screen.findByRole('button', { name: /15/i });
    await user.click(creditButton);

    expect(screen.getByTestId('location-display')).toHaveTextContent('/buy-credits');
  });

  it('routes the credits indicator to plan when bundles are unavailable', async () => {
    const user = userEvent.setup();
    apiMocks.listBundles.mockResolvedValue({ bundles: [] });
    renderLayout('/dashboard');

    const creditButton = await screen.findByRole('button', { name: /15/i });
    await user.click(creditButton);

    expect(screen.getByTestId('location-display')).toHaveTextContent('/plan');
  });

  it('dismisses latest announcement and stores dismissal', async () => {
    const user = userEvent.setup();
    apiMocks.listAnnouncements.mockResolvedValue({ announcements: [{ id: 'ann-1', title: 'New Agents available' }] });

    renderLayout('/dashboard');

    expect(await screen.findByText('New Agents available')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Dismiss' }));

    expect(screen.queryByText('New Agents available')).not.toBeInTheDocument();
    expect(localStorage.getItem('dismissed_announcement')).toBe('ann-1');
  });
});
```

- [ ] **Step 2: Run Layout test and verify it fails**

Run:

```bash
cd frontend && npm test -- --run src/components/Layout.test.tsx
```

Expected: FAIL before implementation because `MobileBottomNav` is not rendered and the main content lacks mobile-safe bottom padding.

- [ ] **Step 3: Modify `Layout.tsx` to render mobile bottom nav**

In `frontend/src/components/Layout.tsx`, add import:

```tsx
import { MobileBottomNav } from './app';
```

Keep existing `isPlatformAdmin` calculation. Replace the main content block near the end with:

```tsx
      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 pb-28 pt-8 sm:px-6 lg:px-8 md:pb-8">
        <Outlet context={{ setUnreadCount, showTeam }} />
      </main>

      {isAuthenticated && (
        <MobileBottomNav hasAdminAccess={isPlatformAdmin} />
      )}
```

- [ ] **Step 4: Run Layout test and verify it passes**

Run:

```bash
cd frontend && npm test -- --run src/components/Layout.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Run affected existing tests**

Run:

```bash
cd frontend && npm test -- --run src/components/AdminLayout.test.tsx src/components/Layout.test.tsx
```

Expected: PASS.

- [ ] **Step 6: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/components/Layout.tsx frontend/src/components/Layout.test.tsx
git commit -m "feat: add mobile app navigation"
```

---

## Task 3: Dashboard Marketplace Productization

**Files:**
- Modify: `frontend/src/pages/app/DashboardPage.tsx`
- Modify: `frontend/src/pages/app/DashboardPage.test.tsx`
- Depends on: Task 1 (`CreditExplainer`, `EmptyState`, `ErrorState`)
- Owns: all changes to `DashboardPage.tsx`

- [ ] **Step 1: Update Dashboard tests for marketplace behavior**

In `frontend/src/pages/app/DashboardPage.test.tsx`, update the first test and add tests below the existing empty-state test. Use this code:

```tsx
  it('renders the agent marketplace hero, credit safety copy, search, categories, and agent cards', async () => {
    renderDashboardPage();

    expect(await screen.findByText('Find the right AI agent for your next task.')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText('36 credits available')).toBeInTheDocument();
    });
    expect(screen.getByText('Credits are charged only after a successful response.')).toBeInTheDocument();
    expect(screen.getByRole('searchbox', { name: 'Search Agents' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Marketing' })).toBeInTheDocument();
    expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.getByText('Best for Marketing teams')).toBeInTheDocument();
    expect(screen.getByText('3 credits/message')).toBeInTheDocument();
    expect(screen.getByText('Text chat')).toBeInTheDocument();
  });

  it('filters agents by search text and category chip', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([
      ...mockAgents,
      {
        ...mockAgents[0],
        id: 'agent-2',
        name: 'Legal Reviewer',
        slug: 'legal-reviewer',
        category: 'Legal',
        description: 'Reviews contracts and clauses',
        suggestedPrompts: ['Review this contract'],
      },
    ]);

    renderDashboardPage();

    expect(await screen.findByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.getByText('Legal Reviewer')).toBeInTheDocument();

    await user.type(screen.getByRole('searchbox', { name: 'Search Agents' }), 'legal');
    expect(screen.queryByText('Growth Strategist')).not.toBeInTheDocument();
    expect(screen.getByText('Legal Reviewer')).toBeInTheDocument();

    await user.clear(screen.getByRole('searchbox', { name: 'Search Agents' }));
    await user.click(screen.getByRole('button', { name: 'Marketing' }));
    expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.queryByText('Legal Reviewer')).not.toBeInTheDocument();
  });

  it('starts chat using the agent slug', async () => {
    const user = userEvent.setup();
    renderDashboardPage();

    await user.click(await screen.findByRole('button', { name: 'Start with Growth Strategist' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/growth-strategist');
  });

  it('starts chat with a selected suggested prompt in the query string', async () => {
    const user = userEvent.setup();
    renderDashboardPage();

    await user.click(await screen.findByRole('button', { name: 'Try prompt: Draft a launch plan' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/growth-strategist?prompt=Draft+a+launch+plan');
  });

  it('hides ordinary-user image and video labels even when backend capabilities include them', async () => {
    apiMocks.listAgents.mockResolvedValue([
      {
        ...mockAgents[0],
        capabilities: ['text_chat', 'image_generation', 'video_generation'],
      },
    ]);

    renderDashboardPage();

    expect(await screen.findByText('Text chat')).toBeInTheDocument();
    expect(screen.queryByText('Image')).not.toBeInTheDocument();
    expect(screen.queryByText('Video')).not.toBeInTheDocument();
  });
```

Also update the retry test's final negative assertion from `Unable to load experts right now.` to `Unable to load agents right now.`.

- [ ] **Step 2: Run Dashboard tests and verify they fail before implementation**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/DashboardPage.test.tsx
```

Expected: FAIL on new hero/search/category/CTA/query/capability expectations.

- [ ] **Step 3: Implement Dashboard marketplace behavior**

In `frontend/src/pages/app/DashboardPage.tsx`, update imports:

```tsx
import { useMemo, useState } from 'react';
import DOMPurify from 'dompurify';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Landmark, MessageCircle, PenLine, Scale, Search, Sparkles, TowerControl, Wallet, Zap, Headphones, Globe2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { agentsApi, brandingApi, usageApi } from '../../api/client';
import { useTenant } from '../../contexts/TenantContext';
import { CreditExplainer, EmptyState, ErrorState } from '../../components/app';
import type { Agent } from '../../types';
import { getErrorMessage } from '../../utils/errors';
```

Inside `DashboardPage`, add state and derived data after credit calculations:

```tsx
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('Recommended');

  const categories = useMemo(() => {
    const uniqueCategories = Array.from(new Set((agents ?? []).map((agent) => agent.category).filter(Boolean))).sort();
    return ['Recommended', ...uniqueCategories];
  }, [agents]);

  const filteredAgents = useMemo(() => {
    const normalizedSearch = searchTerm.trim().toLowerCase();
    return (agents ?? []).filter((agent) => {
      const matchesSearch = normalizedSearch === '' || [
        agent.name,
        agent.category,
        agent.description,
        ...(agent.suggestedPrompts ?? []),
      ].some((value) => value.toLowerCase().includes(normalizedSearch));
      const matchesCategory = selectedCategory === 'Recommended' || agent.category === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  }, [agents, searchTerm, selectedCategory]);

  const commonMessageCost = filteredAgents.length > 0
    ? Math.max(1, Math.min(...filteredAgents.map((agent) => agent.creditCost?.textMessageCredits ?? 1)))
    : 1;

  const startWithPrompt = (agent: Agent, prompt: string) => {
    const params = new URLSearchParams({ prompt });
    navigate(`/chat/${agent.slug}?${params.toString()}`);
  };
```

Replace the existing hero section with:

```tsx
      <section className="rounded-[28px] border border-white/8 bg-dark-950/70 p-1 shadow-[0_30px_80px_-40px_rgba(0,0,0,0.9)]">
        <div className="rounded-[24px] border border-primary-500/10 bg-[radial-gradient(circle_at_top_left,rgba(139,92,246,0.20),transparent_36%),linear-gradient(180deg,rgba(17,24,39,0.98),rgba(2,6,23,0.94))] px-6 py-7 sm:px-8 sm:py-9 lg:px-10">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl">
              <div className="inline-flex items-center gap-2 rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-primary-200">
                <Sparkles className="h-3.5 w-3.5" />
                Agent Marketplace
              </div>
              <h1 className="mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl lg:text-5xl">
                Find the right AI agent for your next task.
              </h1>
              <p className="mt-4 max-w-2xl text-sm leading-7 text-dark-300 sm:text-base">
                Search practical business Agents, try a suggested prompt, and only pay credits after a successful response.
              </p>
              <label className="sr-only" htmlFor="agent-search">Search Agents</label>
              <div className="mt-6 flex max-w-2xl items-center gap-3 rounded-2xl border border-white/10 bg-white px-4 py-3 text-dark-950 shadow-xl shadow-black/20">
                <Search className="h-5 w-5 text-dark-400" />
                <input
                  id="agent-search"
                  type="search"
                  role="searchbox"
                  aria-label="Search Agents"
                  value={searchTerm}
                  onChange={(event) => setSearchTerm(event.target.value)}
                  placeholder="Search legal, tax, marketing, support..."
                  className="min-w-0 flex-1 bg-transparent text-sm text-dark-950 placeholder:text-dark-400 focus:outline-none"
                />
              </div>
            </div>

            <div className="min-w-full lg:min-w-[340px] lg:max-w-sm">
              <CreditExplainer
                balance={usageSummary ? availableCreditsValue : null}
                perMessageCost={commonMessageCost}
              />
            </div>
          </div>
        </div>
      </section>
```

After custom dashboard HTML/error banner and before the grid, render category chips:

```tsx
      {categories.length > 1 ? (
        <div className="flex flex-wrap gap-2">
          {categories.map((category) => (
            <button
              key={category}
              type="button"
              onClick={() => setSelectedCategory(category)}
              className={`rounded-full px-4 py-2 text-sm font-semibold transition-colors ${
                selectedCategory === category
                  ? 'bg-primary-500 text-white'
                  : 'border border-white/8 bg-dark-900/70 text-dark-300 hover:border-primary-400/30 hover:text-white'
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      ) : null}
```

Replace grid decision with:

```tsx
      {agentsLoading || !tenantReady ? (
        <AgentGridSkeleton />
      ) : agents && agents.length > 0 && filteredAgents.length > 0 ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
          {filteredAgents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onStartChat={() => navigate(`/chat/${agent.slug}`)}
              onPromptClick={(prompt) => startWithPrompt(agent, prompt)}
            />
          ))}
        </div>
      ) : agents && agents.length > 0 ? (
        <EmptyState
          icon={Search}
          title="No Agents match your search"
          description="Try a different keyword or category to find the right Agent for your task."
          action={{ label: 'Clear filters', onClick: () => { setSearchTerm(''); setSelectedCategory('Recommended'); } }}
        />
      ) : agentsError ? null : (
        <EmptyExpertsState />
      )}
```

Change `AgentCardProps` and `AgentCard` signature:

```tsx
interface AgentCardProps {
  agent: Agent;
  onStartChat: () => void;
  onPromptClick: (prompt: string) => void;
}

function AgentCard({ agent, onStartChat, onPromptClick }: AgentCardProps) {
```

Inside `AgentCard`, replace suggested prompt `<li>` with buttons:

```tsx
            {agent.suggestedPrompts?.slice(0, 3).map((example, index) => (
              <li key={index}>
                <button
                  type="button"
                  onClick={() => onPromptClick(example)}
                  className="flex w-full items-start gap-2 rounded-xl px-2 py-1.5 text-left text-sm text-dark-200 transition-colors hover:bg-dark-800 hover:text-white"
                  aria-label={`Try prompt: ${example}`}
                >
                  <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-primary-400" />
                  <span>{example}</span>
                </button>
              </li>
            ))}
```

Below the description, add best-for copy:

```tsx
        <p className="mt-3 text-xs font-semibold uppercase tracking-[0.18em] text-primary-300">
          Best for {agent.category} teams
        </p>
```

Replace capability display so ordinary users only see usable text chat:

```tsx
            {agent.capabilities?.includes('text_chat') && (
              <span className="text-xs text-dark-400">Text chat</span>
            )}
```

Replace button accessible name:

```tsx
          <button
            onClick={onStartChat}
            className="inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600"
            aria-label={`Start with ${agent.name}`}
          >
            Start
            <ArrowRight className="h-4 w-4" />
          </button>
```

Replace `InlineErrorBanner` body with shared `ErrorState`:

```tsx
function InlineErrorBanner({ message, onRetry, isRetrying }: { message: string; onRetry: () => void; isRetrying: boolean }) {
  return (
    <ErrorState
      title="Unable to load agents right now."
      message={message}
      retryLabel="Retry"
      onRetry={onRetry}
      isRetrying={isRetrying}
    />
  );
}
```

Replace `EmptyExpertsState` implementation:

```tsx
function EmptyExpertsState() {
  return (
    <EmptyState
      icon={MessageCircle}
      title="No agents yet"
      description="This workspace has not published any Agents yet. Check back soon or contact your administrator to publish the first Agent."
    />
  );
}
```

- [ ] **Step 4: Run Dashboard tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/DashboardPage.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Run focused frontend type-check**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS, or only pre-existing unrelated errors. Fix any errors from `DashboardPage.tsx`.

- [ ] **Step 6: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/pages/app/DashboardPage.tsx frontend/src/pages/app/DashboardPage.test.tsx
git commit -m "feat: productize agent marketplace dashboard"
```

---

## Task 4: Chat UX Productization

**Files:**
- Modify: `frontend/src/pages/app/ChatPage.tsx`
- Modify: `frontend/src/pages/app/ChatPage.test.tsx`
- Depends on: Task 1 (`MarkdownMessage`, `ErrorState`)

- [ ] **Step 1: Update Chat tests for P0 UX**

In `frontend/src/pages/app/ChatPage.test.tsx`, update terminology assertions:

- Replace `Expert not found` with `Agent not found`.
- Replace `Unable to load this expert` with `Unable to load this Agent`.
- Replace `Back to Experts` accessible names with `Back to Agents`.
- Replace `Experts page` route sentinel with `Agents page`.

Add these tests near the existing stream tests:

```tsx
  it('prefills the composer from the prompt query string and removes the prompt from the URL', async () => {
    renderChatPage('/chat/agent-1?prompt=Draft+a+launch+plan');

    const textbox = await screen.findByRole('textbox');
    expect(textbox).toHaveValue('Draft a launch plan');
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1');
  });

  it('renders assistant markdown instead of plain preformatted text', async () => {
    apiMocks.messages.mockResolvedValue([
      {
        ...mockMessages[1],
        content: '## Launch Plan\n- Pick a segment\nUse `credits` carefully.\n[Docs](https://example.com)',
      },
    ]);

    renderChatPage('/chat/agent-1?conversationId=conv-1');

    expect(await screen.findByRole('heading', { name: 'Launch Plan' })).toBeInTheDocument();
    expect(screen.getByText('Pick a segment')).toBeInTheDocument();
    expect(screen.getByText('credits')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Docs' })).toHaveAttribute('href', 'https://example.com');
  });

  it('shows explicit composer credit safety copy', async () => {
    renderChatPage('/chat/agent-1');

    expect(await screen.findByText('This message costs 7 credits. Failed responses are not charged.')).toBeInTheDocument();
  });

  it('shows server-sent stream errors inline and restores the draft', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.stream.mockImplementation(async (_request, onEvent) => {
      onEvent({ event: 'message_start', data: { conversationId: 'conv-new', messageId: 'msg-new-assistant' } });
      onEvent({ event: 'delta', data: { text: 'Partial answer' } });
      onEvent({ event: 'error', data: { message: 'AI service is temporarily unavailable. Please try again.' } });
    });

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');
    await user.keyboard('{Enter}');

    expect(await screen.findByText('AI service is temporarily unavailable. Please try again.')).toBeInTheDocument();
    expect(textbox).toHaveValue('Help me plan a launch');
    expect(screen.queryByText('Partial answer')).not.toBeInTheDocument();
  });

  it('shows a retry button for message load errors', async () => {
    const user = userEvent.setup();
    apiMocks.messages
      .mockRejectedValueOnce(new Error('Messages unavailable'))
      .mockResolvedValueOnce(mockMessages);

    renderChatPage('/chat/agent-1?conversationId=conv-1');

    expect(await screen.findByText('Unable to load messages.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => {
      expect(apiMocks.messages).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('How should we launch?')).toBeInTheDocument();
  });

  it('navigates back to dashboard from agent error states', async () => {
    const user = userEvent.setup();
    apiMocks.agentsGet.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 404,
        statusText: 'Not Found',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Agent not found' },
      } as never)
    );

    renderChatPage('/chat/missing-agent');

    await user.click(await screen.findByRole('button', { name: /Back to Agents/i }));
    expect(await screen.findByText('Agents page')).toBeInTheDocument();
  });
```

- [ ] **Step 2: Run Chat tests and verify they fail before implementation**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx
```

Expected: FAIL on prompt query, Markdown, new copy, stream error draft preservation, and retry expectations.

- [ ] **Step 3: Implement Chat UX changes**

In `frontend/src/pages/app/ChatPage.tsx`, add imports:

```tsx
import { ErrorState, MarkdownMessage } from '../../components/app';
```

Use existing `useSearchParams`; after `conversationId` definition, add:

```tsx
  const promptParam = searchParams.get('prompt')?.trim() || '';
```

Add effect after the `conversationId` cleanup effect:

```tsx
  useEffect(() => {
    if (!promptParam) {
      return;
    }
    setDraft(promptParam);
    const nextParams = new URLSearchParams(searchParams);
    nextParams.delete('prompt');
    setSearchParams(nextParams, { replace: true });
    focusComposer();
  }, [promptParam]);
```

In the stream `event.event === 'error'` branch, preserve the draft:

```tsx
          } else if (event.event === 'error') {
            setIsStreaming(false);
            setComposerNotice({ tone: 'error', message: event.data.message });
            setDraft(message);
            setPendingUserMessage(null);
            setStreamingContent('');
            streamingContentRef.current = '';
          }
```

Update agent error copy:

```tsx
        <h1 className="mt-5 text-2xl font-semibold text-white">Agent not found</h1>
        <p className="mt-2 max-w-lg text-sm text-dark-400">
          The Agent you tried to open is unavailable or may have been removed.
        </p>
```

Use button label:

```tsx
          Back to Agents
```

For generic agent load error:

```tsx
        <h1 className="mt-5 text-2xl font-semibold text-white">Unable to load this Agent</h1>
```

Update sidebar labels:

```tsx
<p className="text-xs font-medium uppercase tracking-[0.16em] text-dark-500">Current Agent</p>
```

Update all `aria-label="Back to Experts"` to `aria-label="Back to Agents"`.

For messages query, destructure `refetch`:

```tsx
  const {
    data: messages,
    isLoading: messagesLoading,
    error: messagesError,
    refetch: refetchMessages,
    isFetching: messagesFetching,
  } = useQuery({
```

Replace messages error block with:

```tsx
          ) : messagesError ? (
            <ErrorState
              title="Unable to load messages."
              message={getErrorMessage(messagesError)}
              retryLabel="Retry"
              onRetry={() => void refetchMessages()}
              isRetrying={messagesFetching}
            />
```

Replace assistant message content rendering:

```tsx
                    {message.role === 'assistant' ? (
                      <MarkdownMessage content={message.content} />
                    ) : (
                      <p className="whitespace-pre-wrap break-words text-sm leading-6">{message.content}</p>
                    )}
```

Replace streaming content rendering:

```tsx
                          <div className="text-sm leading-6">
                            <MarkdownMessage content={streamingContent} />
                            <span className="animate-pulse">|</span>
                          </div>
```

In the composer footer, add cost safety copy above the textarea:

```tsx
          <p className="mb-3 text-xs text-dark-400">
            This message costs {requiredCredits} credits. Failed responses are not charged.
          </p>
```

For mobile sidebar, start with a low-risk responsive improvement: change root layout class from:

```tsx
<div className="flex h-[calc(100vh-8rem)] min-h-[40rem] flex-col gap-4 lg:flex-row">
```

to:

```tsx
<div className="flex h-[calc(100vh-10rem)] min-h-[34rem] flex-col gap-4 lg:h-[calc(100vh-8rem)] lg:min-h-[40rem] lg:flex-row">
```

This keeps current sidebar visible on mobile but reduces the fixed-height pressure. A true drawer can be handled in P1 if this P0 pass needs to stay smaller.

- [ ] **Step 4: Run Chat tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Run Dashboard-to-Chat integration tests**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/DashboardPage.test.tsx src/pages/app/ChatPage.test.tsx
```

Expected: PASS.

- [ ] **Step 6: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/pages/app/ChatPage.tsx frontend/src/pages/app/ChatPage.test.tsx
git commit -m "feat: improve agent chat UX"
```

---

## Task 5: Credits Purchase and Billing Success UX

**Files:**
- Modify: `frontend/src/pages/app/BuyCreditsPage.tsx`
- Create: `frontend/src/pages/app/BuyCreditsPage.test.tsx`
- Modify: `frontend/src/pages/app/BillingSuccessPage.tsx`
- Create: `frontend/src/pages/app/BillingSuccessPage.test.tsx`
- Optionally modify: `frontend/src/pages/app/ChatPage.tsx` only if Task 4 did not add `returnTo` to Buy Credits link.

- [ ] **Step 1: Write Buy Credits tests**

Create `frontend/src/pages/app/BuyCreditsPage.test.tsx`:

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import BuyCreditsPage from './BuyCreditsPage';

const apiMocks = vi.hoisted(() => ({
  listBundles: vi.fn(),
  listPlans: vi.fn(),
  checkout: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  bundlesApi: { list: apiMocks.listBundles },
  plansApi: { list: apiMocks.listPlans },
  billingApi: { checkout: apiMocks.checkout },
}));

vi.mock('sonner', () => ({ toast: { error: apiMocks.toastError } }));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderBuyCredits(initialEntry = '/buy-credits') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/buy-credits" element={<><BuyCreditsPage /><LocationDisplay /></>} />
        <Route path="/plan" element={<div>Plan page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('BuyCreditsPage', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    apiMocks.listBundles.mockResolvedValue({
      bundles: [
        { id: 'bundle-1', name: 'Starter', credits: 100, priceCents: 900, isActive: true, sortOrder: 1, createdAt: '', updatedAt: '' },
        { id: 'bundle-2', name: 'Team', credits: 1000, priceCents: 4900, isActive: true, sortOrder: 2, createdAt: '', updatedAt: '' },
      ],
    });
    apiMocks.listPlans.mockResolvedValue({ tenantSubscriptionCredits: 24, tenantPurchasedCredits: 12, plans: [] });
  });

  it('renders current balance, credit education, and bundles with message estimates', async () => {
    renderBuyCredits();

    expect(await screen.findByText('Buy Credits')).toBeInTheDocument();
    expect(screen.getByText(/36/)).toBeInTheDocument();
    expect(screen.getByText('Credits are available immediately after checkout.')).toBeInTheDocument();
    expect(screen.getByText('Starter')).toBeInTheDocument();
    expect(screen.getByText('About 100 typical text messages')).toBeInTheDocument();
    expect(screen.getByText('$9.00')).toBeInTheDocument();
    expect(screen.getByText('Team')).toBeInTheDocument();
    expect(screen.getByText('Best for teams')).toBeInTheDocument();
  });

  it('shows an actionable empty state when bundles are unavailable', async () => {
    apiMocks.listBundles.mockResolvedValue({ bundles: [] });
    renderBuyCredits();

    expect(await screen.findByText('No credit bundles are available right now')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View plans' })).toHaveAttribute('href', '/plan');
  });

  it('stores return path and redirects to checkout URL after buying', async () => {
    const user = userEvent.setup();
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/chat/agent-1%3FconversationId%3Dconv-1');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(apiMocks.checkout).toHaveBeenCalledWith({ bundleId: 'bundle-1' });
    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBe('/chat/agent-1?conversationId=conv-1');
    expect(window.location.assign).toHaveBeenCalledWith('https://checkout.example/session');

    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation });
  });

  it('disables buy buttons while checkout is loading', async () => {
    const user = userEvent.setup();
    let resolveCheckout: (value: { checkoutUrl: string }) => void = () => undefined;
    apiMocks.checkout.mockReturnValue(new Promise((resolve) => { resolveCheckout = resolve; }));

    renderBuyCredits();

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));
    expect(screen.getByRole('button', { name: 'Buy Starter' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Buy Team' })).toBeDisabled();

    resolveCheckout({ checkoutUrl: 'https://checkout.example/session' });
  });
});
```

- [ ] **Step 2: Write Billing Success tests**

Create `frontend/src/pages/app/BillingSuccessPage.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import BillingSuccessPage from './BillingSuccessPage';

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderSuccess() {
  return render(
    <MemoryRouter initialEntries={['/billing/success?session_id=test']}>
      <Routes>
        <Route path="/billing/success" element={<><BillingSuccessPage /><LocationDisplay /></>} />
        <Route path="/dashboard" element={<><div>Dashboard page</div><LocationDisplay /></>} />
        <Route path="/plan" element={<><div>Plan page</div><LocationDisplay /></>} />
        <Route path="/chat/:agentId" element={<><div>Chat page</div><LocationDisplay /></>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('BillingSuccessPage', () => {
  afterEach(() => {
    vi.useRealTimers();
    sessionStorage.clear();
  });

  it('shows success actions and continues previous conversation when return path exists', async () => {
    const user = userEvent.setup();
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/agent-1?conversationId=conv-1');

    renderSuccess();

    expect(screen.getByText('Credits added successfully.')).toBeInTheDocument();
    await user.click(screen.getByRole('link', { name: 'Continue conversation' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1?conversationId=conv-1');
    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('falls back to plan after delay when no return path exists', () => {
    vi.useFakeTimers();
    renderSuccess();

    expect(screen.getByRole('link', { name: 'Browse Agents' })).toHaveAttribute('href', '/dashboard');
    vi.advanceTimersByTime(5000);

    expect(screen.getByTestId('location-display')).toHaveTextContent('/plan');
  });
});
```

- [ ] **Step 3: Run new billing tests and verify they fail before implementation**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/BuyCreditsPage.test.tsx src/pages/app/BillingSuccessPage.test.tsx
```

Expected: FAIL because new copy, return path storage, `window.location.assign`, and success actions are not implemented.

- [ ] **Step 4: Implement Buy Credits UX**

In `frontend/src/pages/app/BuyCreditsPage.tsx`, update imports:

```tsx
import { Link, useSearchParams } from 'react-router-dom';
import { CheckCircle, ShoppingCart, Sparkles, Zap } from 'lucide-react';
import { toast } from 'sonner';
import { bundlesApi, plansApi, billingApi } from '../../api/client';
import type { CreditBundle } from '../../types';
import LoadingSpinner from '../../components/LoadingSpinner';
import { EmptyState } from '../../components/app';
import { getErrorMessage } from '../../utils/errors';
```

Add helper functions:

```tsx
function getBundleLabel(bundle: CreditBundle, index: number): string {
  if (bundle.credits >= 1000) return 'Best for teams';
  if (index === 1 || bundle.credits >= 500) return 'Best value';
  return 'Occasional use';
}

function getEstimatedMessages(bundle: CreditBundle): string {
  return `About ${bundle.credits.toLocaleString()} typical text messages`;
}
```

Inside component, add:

```tsx
  const [searchParams] = useSearchParams();
```

Update `handleBuy`:

```tsx
  const handleBuy = async (bundleId: string) => {
    setCheckoutLoading(bundleId);
    const returnTo = searchParams.get('returnTo');
    if (returnTo && returnTo.startsWith('/')) {
      sessionStorage.setItem('agentstore.checkoutReturnTo', returnTo);
    }

    try {
      const result = await billingApi.checkout({ bundleId });
      if (result.checkoutUrl) {
        window.location.assign(result.checkoutUrl);
      }
    } catch (err) {
      toast.error(getErrorMessage(err));
    } finally {
      setCheckoutLoading(null);
    }
  };
```

Replace page JSX with:

```tsx
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
                <span className="text-dark-500 text-sm ml-1">({formatPrice(Math.round(bundle.priceCents / bundle.credits * 100))}/100 credits)</span>
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
```

- [ ] **Step 5: Implement Billing Success UX**

Replace `frontend/src/pages/app/BillingSuccessPage.tsx` with:

```tsx
import { useEffect, useMemo } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowRight, CheckCircle, MessageCircle, Receipt, Sparkles } from 'lucide-react';

const checkoutReturnKey = 'agentstore.checkoutReturnTo';

export default function BillingSuccessPage() {
  const navigate = useNavigate();
  const returnTo = useMemo(() => sessionStorage.getItem(checkoutReturnKey), []);
  const safeReturnTo = returnTo && returnTo.startsWith('/') ? returnTo : '';

  useEffect(() => {
    if (safeReturnTo) {
      return undefined;
    }
    const timer = setTimeout(() => navigate('/plan'), 5000);
    return () => clearTimeout(timer);
  }, [navigate, safeReturnTo]);

  const clearReturnPath = () => {
    sessionStorage.removeItem(checkoutReturnKey);
  };

  return (
    <div className="mx-auto flex max-w-2xl flex-col items-center justify-center py-20 text-center">
      <div className="flex h-20 w-20 items-center justify-center rounded-3xl border border-accent-emerald/20 bg-accent-emerald/10 text-accent-emerald">
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

      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        {safeReturnTo ? (
          <Link
            to={safeReturnTo}
            onClick={clearReturnPath}
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
```

- [ ] **Step 6: Add return path to Chat Buy Credits links if not already done**

In `frontend/src/pages/app/ChatPage.tsx`, find the Buy Credits link in the composer notice. Replace its `to` prop with:

```tsx
const buyCreditsReturnTo = `${location.pathname}${location.search}`;
```

This requires importing `useLocation` from `react-router-dom` and adding:

```tsx
const location = useLocation();
```

Then use:

```tsx
<Link to={`/buy-credits?returnTo=${encodeURIComponent(buyCreditsReturnTo)}`}>Buy Credits</Link>
```

If Task 4 already made this change, do not duplicate it.

- [ ] **Step 7: Run billing tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/BuyCreditsPage.test.tsx src/pages/app/BillingSuccessPage.test.tsx src/pages/app/ChatPage.test.tsx
```

Expected: PASS.

- [ ] **Step 8: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/pages/app/BuyCreditsPage.tsx frontend/src/pages/app/BuyCreditsPage.test.tsx frontend/src/pages/app/BillingSuccessPage.tsx frontend/src/pages/app/BillingSuccessPage.test.tsx frontend/src/pages/app/ChatPage.tsx frontend/src/pages/app/ChatPage.test.tsx
git commit -m "feat: clarify credits purchase flow"
```

---

## Task 6: First-Success Onboarding

**Files:**
- Modify: `frontend/src/pages/app/OnboardingPage.tsx`
- Create: `frontend/src/pages/app/OnboardingPage.test.tsx`

- [ ] **Step 1: Write Onboarding tests**

Create `frontend/src/pages/app/OnboardingPage.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import OnboardingPage from './OnboardingPage';

const apiMocks = vi.hoisted(() => ({
  completeOnboarding: vi.fn(),
  refreshUser: vi.fn(),
  listAgents: vi.fn(),
  usageSummary: vi.fn(),
}));

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { displayName: 'Ada Lovelace' },
    refreshUser: apiMocks.refreshUser,
  }),
}));

vi.mock('../../api/client', () => ({
  authApi: { completeOnboarding: apiMocks.completeOnboarding },
  agentsApi: { list: apiMocks.listAgents },
  usageApi: { summary: apiMocks.usageSummary },
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderOnboarding() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/onboarding']}>
        <Routes>
          <Route path="/onboarding" element={<><OnboardingPage /><LocationDisplay /></>} />
          <Route path="/dashboard" element={<><div>Dashboard page</div><LocationDisplay /></>} />
          <Route path="/chat/:agentId" element={<><div>Chat page</div><LocationDisplay /></>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('OnboardingPage', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    apiMocks.listAgents.mockResolvedValue([
      {
        id: 'agent-1',
        name: 'Marketing Writer',
        slug: 'marketing-writer',
        category: 'Marketing',
        description: 'Writes launch copy',
        suggestedPrompts: ['Write a launch email', 'Create ad copy'],
        visibility: 'private',
        capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '',
        updatedAt: '',
      },
      {
        id: 'agent-2',
        name: 'Legal Reviewer',
        slug: 'legal-reviewer',
        category: 'Legal',
        description: 'Reviews clauses',
        suggestedPrompts: ['Review this clause'],
        visibility: 'private',
        capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 2, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '',
        updatedAt: '',
      },
    ]);
    apiMocks.usageSummary.mockResolvedValue({ subscriptionCredits: 10, purchasedCredits: 5, usage: [], totalCreditsUsed: 0, periodStart: '' });
    apiMocks.completeOnboarding.mockResolvedValue({});
    apiMocks.refreshUser.mockResolvedValue(undefined);
  });

  it('guides the user from goal to recommended Agent to first prompt', async () => {
    const user = userEvent.setup();
    renderOnboarding();

    expect(await screen.findByText('What do you want to accomplish first?')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Marketing' }));

    expect(await screen.findByText('Pick your first Agent')).toBeInTheDocument();
    expect(screen.getByText('Marketing Writer')).toBeInTheDocument();
    expect(screen.queryByText('Legal Reviewer')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Choose Marketing Writer' }));

    expect(await screen.findByText('Choose your first prompt')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Write a launch email' }));

    expect(await screen.findByText('You start with 15 credits.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Start chatting' }));

    await waitFor(() => expect(apiMocks.completeOnboarding).toHaveBeenCalledTimes(1));
    expect(apiMocks.refreshUser).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/marketing-writer?prompt=Write+a+launch+email');
  });

  it('falls back to marketplace when no Agents are available', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([]);
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Legal' }));
    expect(await screen.findByText('No Agents are published yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Go to marketplace' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/dashboard');
  });

  it('still navigates to chat if completing onboarding fails', async () => {
    const user = userEvent.setup();
    apiMocks.completeOnboarding.mockRejectedValue(new Error('failed'));
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    await user.click(await screen.findByRole('button', { name: 'Write a launch email' }));
    await user.click(await screen.findByRole('button', { name: 'Start chatting' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/marketing-writer?prompt=Write+a+launch+email');
  });
});
```

- [ ] **Step 2: Run Onboarding test and verify it fails before implementation**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/OnboardingPage.test.tsx
```

Expected: FAIL because the current onboarding is profile/team/done.

- [ ] **Step 3: Replace Onboarding with first-success flow**

Replace `frontend/src/pages/app/OnboardingPage.tsx` with:

```tsx
import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CheckCircle, ChevronRight, MessageCircle, Sparkles, Target, Zap } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../contexts/AuthContext';
import { authApi, agentsApi, usageApi } from '../../api/client';
import LoadingSpinner from '../../components/LoadingSpinner';
import type { Agent } from '../../types';

type Step = 'goal' | 'agent' | 'prompt' | 'credits';

const goals = ['Marketing', 'Legal', 'Tax', 'Customer support', 'E-commerce', 'Strategy', 'Other'];

export default function OnboardingPage() {
  const navigate = useNavigate();
  const { refreshUser } = useAuth();
  const [step, setStep] = useState<Step>('goal');
  const [selectedGoal, setSelectedGoal] = useState('');
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
  const [selectedPrompt, setSelectedPrompt] = useState('');
  const [loading, setLoading] = useState(false);

  const { data: agents = [], isLoading: agentsLoading } = useQuery({ queryKey: ['onboarding-agents'], queryFn: agentsApi.list, retry: false });
  const { data: usageSummary } = useQuery({ queryKey: ['onboarding-usage-summary'], queryFn: usageApi.summary, retry: false });

  const recommendedAgents = useMemo(() => {
    if (!selectedGoal || selectedGoal === 'Other') return agents.slice(0, 3);
    const normalizedGoal = selectedGoal.toLowerCase();
    const matches = agents.filter((agent) => [agent.category, agent.name, agent.description].some((value) => value.toLowerCase().includes(normalizedGoal)));
    return (matches.length > 0 ? matches : agents).slice(0, 3);
  }, [agents, selectedGoal]);

  const totalCredits = (usageSummary?.subscriptionCredits ?? 0) + (usageSummary?.purchasedCredits ?? 0);

  const goToDestination = async () => {
    setLoading(true);
    const destination = selectedAgent
      ? `/chat/${selectedAgent.slug}${selectedPrompt ? `?${new URLSearchParams({ prompt: selectedPrompt }).toString()}` : ''}`
      : '/dashboard';

    try {
      await authApi.completeOnboarding();
      await refreshUser();
    } catch {
      // The user should still reach the product even if marking onboarding complete fails.
    } finally {
      navigate(destination);
    }
  };

  const chooseGoal = (goal: string) => {
    setSelectedGoal(goal);
    setStep('agent');
  };

  const chooseAgent = (agent: Agent) => {
    setSelectedAgent(agent);
    setSelectedPrompt(agent.suggestedPrompts?.[0] ?? '');
    setStep('prompt');
  };

  return (
    <div className="min-h-screen bg-dark-950 px-4 py-10">
      <div className="mx-auto w-full max-w-3xl">
        <div className="mb-8 text-center">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-500/15 text-primary-300">
            <Sparkles className="h-7 w-7" />
          </div>
          <h1 className="mt-5 text-3xl font-bold text-white">Welcome to AgentStore</h1>
          <p className="mt-2 text-sm text-dark-400">Choose a goal, pick an Agent, and start with a useful prompt.</p>
        </div>

        <div className="mb-8 flex items-center justify-center gap-2">
          {(['goal', 'agent', 'prompt', 'credits'] as Step[]).map((item, index) => {
            const labels: Record<Step, string> = { goal: 'Goal', agent: 'Agent', prompt: 'Prompt', credits: 'Credits' };
            const activeIndex = ['goal', 'agent', 'prompt', 'credits'].indexOf(step);
            return (
              <div key={item} className="flex items-center gap-2">
                <div className={`rounded-full px-3 py-1.5 text-sm font-medium ${index <= activeIndex ? 'bg-primary-500/20 text-primary-300' : 'bg-dark-800 text-dark-500'}`}>
                  {labels[item]}
                </div>
                {index < 3 ? <ChevronRight className="h-4 w-4 text-dark-600" /> : null}
              </div>
            );
          })}
        </div>

        {step === 'goal' && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="flex items-center gap-2 text-xl font-bold text-white"><Target className="h-5 w-5 text-primary-300" />What do you want to accomplish first?</h2>
            <p className="mt-2 text-sm text-dark-400">We will recommend a practical Agent and a first prompt.</p>
            <div className="mt-6 grid gap-3 sm:grid-cols-2">
              {goals.map((goal) => (
                <button key={goal} type="button" onClick={() => chooseGoal(goal)} className="rounded-2xl border border-dark-800 bg-dark-950/60 px-4 py-4 text-left font-semibold text-white transition-colors hover:border-primary-500/40 hover:bg-dark-900">
                  {goal}
                </button>
              ))}
            </div>
          </section>
        )}

        {step === 'agent' && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="text-xl font-bold text-white">Pick your first Agent</h2>
            <p className="mt-2 text-sm text-dark-400">Recommended for {selectedGoal}.</p>
            {agentsLoading ? <LoadingSpinner size="lg" className="py-10" /> : recommendedAgents.length === 0 ? (
              <div className="mt-6 rounded-2xl border border-dashed border-dark-800 bg-dark-950/60 p-8 text-center">
                <MessageCircle className="mx-auto h-8 w-8 text-dark-500" />
                <h3 className="mt-4 text-lg font-semibold text-white">No Agents are published yet</h3>
                <p className="mt-2 text-sm text-dark-400">You can still enter the marketplace and check back after an administrator publishes Agents.</p>
                <button type="button" onClick={() => navigate('/dashboard')} className="mt-5 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-primary-600">Go to marketplace</button>
              </div>
            ) : (
              <div className="mt-6 grid gap-4">
                {recommendedAgents.map((agent) => (
                  <button key={agent.id} type="button" onClick={() => chooseAgent(agent)} className="rounded-2xl border border-dark-800 bg-dark-950/60 p-4 text-left transition-colors hover:border-primary-500/40 hover:bg-dark-900" aria-label={`Choose ${agent.name}`}>
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <h3 className="font-semibold text-white">{agent.name}</h3>
                        <p className="mt-1 text-sm text-dark-400">{agent.category}</p>
                        <p className="mt-2 text-sm text-dark-300">{agent.description}</p>
                      </div>
                      <span className="rounded-full bg-primary-500/10 px-3 py-1 text-xs font-semibold text-primary-300">{agent.creditCost?.textMessageCredits ?? 1} credit/message</span>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </section>
        )}

        {step === 'prompt' && selectedAgent && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="text-xl font-bold text-white">Choose your first prompt</h2>
            <p className="mt-2 text-sm text-dark-400">Start with a prompt for {selectedAgent.name}. You can edit it before sending.</p>
            <div className="mt-6 grid gap-3">
              {(selectedAgent.suggestedPrompts && selectedAgent.suggestedPrompts.length > 0 ? selectedAgent.suggestedPrompts : ['Help me get started.']).slice(0, 4).map((prompt) => (
                <button key={prompt} type="button" onClick={() => { setSelectedPrompt(prompt); setStep('credits'); }} className="rounded-2xl border border-dark-800 bg-dark-950/60 px-4 py-4 text-left text-sm text-dark-200 transition-colors hover:border-primary-500/40 hover:bg-dark-900 hover:text-white">
                  {prompt}
                </button>
              ))}
            </div>
          </section>
        )}

        {step === 'credits' && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6 text-center">
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-accent-emerald/10 text-accent-emerald">
              <CheckCircle className="h-8 w-8" />
            </div>
            <h2 className="mt-5 text-xl font-bold text-white">You are ready to start.</h2>
            <p className="mt-3 text-sm text-dark-400">You start with {totalCredits.toLocaleString()} credits.</p>
            <p className="mt-1 text-sm text-dark-400">Most Agents cost 1 credit/message. Failed responses are not charged.</p>
            <button type="button" onClick={goToDestination} disabled={loading} className="mt-6 w-full rounded-xl bg-primary-500 px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-600 disabled:opacity-60">
              {loading ? 'Starting...' : 'Start chatting'}
            </button>
          </section>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run Onboarding tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/OnboardingPage.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Run type-check for onboarding changes**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS, or only pre-existing unrelated errors. Fix any errors from `OnboardingPage.tsx`.

- [ ] **Step 6: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/pages/app/OnboardingPage.tsx frontend/src/pages/app/OnboardingPage.test.tsx
git commit -m "feat: guide users to first agent chat"
```

---

## Task 7: Admin Launch Checklist

**Files:**
- Create: `frontend/src/pages/admin/components/LaunchChecklist.tsx`
- Create: `frontend/src/pages/admin/components/LaunchChecklist.test.tsx`
- Modify: `frontend/src/pages/admin/DashboardPage.tsx`
- Create: `frontend/src/pages/admin/DashboardPage.test.tsx`

- [ ] **Step 1: Write LaunchChecklist component test**

Create `frontend/src/pages/admin/components/LaunchChecklist.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import LaunchChecklist, { type LaunchChecklistItem } from './LaunchChecklist';

const items: LaunchChecklistItem[] = [
  { id: 'brand', label: 'Brand configured', status: 'complete', description: 'App name and landing copy are ready.', to: '/admin/branding' },
  { id: 'model', label: 'Model provider connected', status: 'warning', description: 'Connect an OpenAI-compatible provider.', to: '/settings/models' },
  { id: 'chat', label: 'Test chat passed', status: 'pending', description: 'Run a smoke chat before launch.', to: '/dashboard' },
];

describe('LaunchChecklist', () => {
  it('renders launch readiness items with links and statuses', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={items} />
      </MemoryRouter>
    );

    expect(screen.getByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Brand configured/i })).toHaveAttribute('href', '/admin/branding');
    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getByText('Needs attention')).toBeInTheDocument();
    expect(screen.getByText('Review')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Write Admin Dashboard checklist test**

Create `frontend/src/pages/admin/DashboardPage.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AdminDashboardPage from './DashboardPage';

const apiMocks = vi.hoisted(() => ({
  getDashboard: vi.fn(),
  getHealthIntegrations: vi.fn(),
  getFinancialMetrics: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  adminApi: {
    getDashboard: apiMocks.getDashboard,
    getHealthIntegrations: apiMocks.getHealthIntegrations,
    getFinancialMetrics: apiMocks.getFinancialMetrics,
  },
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Area: () => <div />,
  XAxis: () => <div />,
  YAxis: () => <div />,
  Tooltip: () => <div />,
}));

function renderDashboard() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AdminDashboardPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('AdminDashboardPage', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    apiMocks.getDashboard.mockResolvedValue({ users: 42, tenants: 7, health: { healthy: true, issues: [] } });
    apiMocks.getHealthIntegrations.mockResolvedValue({ integrations: [{ name: 'stripe', status: 'not_configured', message: 'Missing key', lastCheck: '', responseMs: 0, calls24h: 0 }] });
    apiMocks.getFinancialMetrics.mockResolvedValue({ data: [{ date: '2026-06-06', value: 12345 }] });
  });

  it('renders summary metrics and launch checklist', async () => {
    renderDashboard();

    expect(await screen.findByText('Admin Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.getByText('Total Users')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('Tenants')).toBeInTheDocument();
    expect(screen.getByText('7')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Stripe webhook healthy/i })).toHaveAttribute('href', '/admin/health#integrations');
  });

  it('changes chart range and refetches financial metrics', async () => {
    const user = userEvent.setup();
    renderDashboard();

    await screen.findByText('Business Metrics');
    await user.click(screen.getByRole('button', { name: '7d' }));

    expect(apiMocks.getFinancialMetrics).toHaveBeenCalledWith({ range: '7d', metric: 'revenue' });
    expect(apiMocks.getFinancialMetrics).toHaveBeenCalledWith({ range: '7d', metric: 'arr' });
    expect(apiMocks.getFinancialMetrics).toHaveBeenCalledWith({ range: '7d', metric: 'dau' });
  });
});
```

- [ ] **Step 3: Run admin tests and verify they fail before implementation**

Run:

```bash
cd frontend && npm test -- --run src/pages/admin/components/LaunchChecklist.test.tsx src/pages/admin/DashboardPage.test.tsx
```

Expected: FAIL because `LaunchChecklist` does not exist.

- [ ] **Step 4: Implement LaunchChecklist component**

Create `frontend/src/pages/admin/components/LaunchChecklist.tsx`:

```tsx
import { Link } from 'react-router-dom';
import { AlertTriangle, CheckCircle, ChevronRight, CircleDot } from 'lucide-react';

export type LaunchChecklistStatus = 'complete' | 'warning' | 'pending';

export interface LaunchChecklistItem {
  id: string;
  label: string;
  status: LaunchChecklistStatus;
  description: string;
  to: string;
}

const statusConfig: Record<LaunchChecklistStatus, { label: string; className: string; icon: typeof CheckCircle }> = {
  complete: { label: 'Complete', className: 'border-accent-emerald/20 bg-accent-emerald/10 text-accent-emerald', icon: CheckCircle },
  warning: { label: 'Needs attention', className: 'border-yellow-500/20 bg-yellow-500/10 text-yellow-400', icon: AlertTriangle },
  pending: { label: 'Review', className: 'border-dark-700 bg-dark-800 text-dark-300', icon: CircleDot },
};

interface LaunchChecklistProps {
  items: LaunchChecklistItem[];
}

export default function LaunchChecklist({ items }: LaunchChecklistProps) {
  const completedCount = items.filter((item) => item.status === 'complete').length;

  return (
    <section className="mb-8 rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-xl font-bold text-white">Launch Checklist</h2>
          <p className="mt-1 text-sm text-dark-400">Know exactly what remains before users can rely on AgentStore.</p>
        </div>
        <p className="text-sm font-semibold text-primary-300">{completedCount}/{items.length} ready</p>
      </div>

      <div className="mt-5 grid gap-3 lg:grid-cols-2">
        {items.map((item) => {
          const config = statusConfig[item.status];
          const Icon = config.icon;
          return (
            <Link key={item.id} to={item.to} className="group rounded-2xl border border-dark-800 bg-dark-950/50 p-4 transition-colors hover:border-primary-500/30 hover:bg-dark-900">
              <div className="flex items-start gap-3">
                <div className={`rounded-xl border p-2 ${config.className}`}>
                  <Icon className="h-4 w-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="font-semibold text-white">{item.label}</h3>
                    <ChevronRight className="h-4 w-4 text-dark-600 transition-colors group-hover:text-primary-300" />
                  </div>
                  <p className="mt-1 text-xs font-semibold uppercase tracking-[0.16em] text-dark-500">{config.label}</p>
                  <p className="mt-2 text-sm leading-5 text-dark-400">{item.description}</p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </section>
  );
}
```

- [ ] **Step 5: Modify Admin Dashboard to render checklist**

In `frontend/src/pages/admin/DashboardPage.tsx`, add import:

```tsx
import LaunchChecklist, { type LaunchChecklistItem } from './components/LaunchChecklist';
```

Before `return`, derive checklist:

```tsx
  const stripeUnconfigured = unconfiguredIntegrations.includes('stripe');
  const resendUnconfigured = unconfiguredIntegrations.includes('resend');
  const launchChecklist: LaunchChecklistItem[] = [
    {
      id: 'brand',
      label: 'Brand configured',
      status: 'pending',
      description: 'Review app name, logo, landing copy, dashboard copy, and auth page text before inviting users.',
      to: '/admin/branding',
    },
    {
      id: 'model',
      label: 'Model provider connected',
      status: 'pending',
      description: 'Connect an OpenAI-compatible provider and set a default text model for chat.',
      to: '/settings/models',
    },
    {
      id: 'agent',
      label: 'First Agent published',
      status: 'pending',
      description: 'Create and publish at least one text-chat Agent so users have something to try.',
      to: '/settings/agents',
    },
    {
      id: 'credits',
      label: 'Credit bundle or plan active',
      status: stripeUnconfigured ? 'warning' : 'pending',
      description: stripeUnconfigured ? 'Stripe is not configured, so paid checkout cannot work yet.' : 'Review plans and credit bundles for the first user cohort.',
      to: '/admin/plans',
    },
    {
      id: 'stripe',
      label: 'Stripe webhook healthy',
      status: stripeUnconfigured ? 'warning' : 'complete',
      description: stripeUnconfigured ? 'Configure Stripe keys and webhook handling before selling credits.' : 'Stripe integration is configured according to the health check.',
      to: '/admin/health#integrations',
    },
    {
      id: 'email',
      label: 'Email provider ready',
      status: resendUnconfigured ? 'warning' : 'complete',
      description: resendUnconfigured ? 'Configure Resend before relying on invites, verification, and password resets.' : 'Email integration is configured according to the health check.',
      to: '/admin/health#integrations',
    },
    {
      id: 'test-chat',
      label: 'Test chat passed',
      status: 'pending',
      description: 'Run a real smoke chat with credits before announcing the product.',
      to: '/dashboard',
    },
  ];
```

Render checklist immediately after the header block and before unconfigured warning:

```tsx
      <LaunchChecklist items={launchChecklist} />
```

- [ ] **Step 6: Run admin tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/admin/components/LaunchChecklist.test.tsx src/pages/admin/DashboardPage.test.tsx
```

Expected: PASS.

- [ ] **Step 7: Run AdminLayout tests for route compatibility**

Run:

```bash
cd frontend && npm test -- --run src/components/AdminLayout.test.tsx src/pages/admin/DashboardPage.test.tsx
```

Expected: PASS.

- [ ] **Step 8: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/pages/admin/DashboardPage.tsx frontend/src/pages/admin/DashboardPage.test.tsx frontend/src/pages/admin/components/LaunchChecklist.tsx frontend/src/pages/admin/components/LaunchChecklist.test.tsx
git commit -m "feat: add admin launch checklist"
```

---

## Task 8: Honest OpenAI-Compatible Provider Test

**Files:**
- Modify: `backend/internal/llm/openai.go`
- Modify: `backend/internal/api/handlers/model_settings.go`
- Modify: `backend/internal/api/handlers/model_settings_test.go`

- [ ] **Step 1: Write backend tests for provider testing**

Append these tests to `backend/internal/api/handlers/model_settings_test.go`:

```go
func TestModelProviderAPI_TestProvider_OpenAICompatibleSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected /v1/models, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("expected bearer key, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer upstream.Close()

	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      upstream.URL,
		APIKey:       "sk-test",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-providers/"+providerID.Hex()+"/test", nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, body)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected ok status, got %v", body)
	}
	if !strings.Contains(body["message"], "OpenAI-compatible provider responded") {
		t.Fatalf("expected connectivity message, got %v", body)
	}
}

func TestModelProviderAPI_TestProvider_OpenAICompatibleFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer upstream.Close()

	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      upstream.URL,
		APIKey:       "sk-test",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-providers/"+providerID.Hex()+"/test", nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 502, got %d body %s", resp.StatusCode, body)
	}
}

func TestModelProviderAPI_TestProvider_UnsupportedProviderIsHonest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Anthropic",
		ProviderType: models.ProviderTypeAnthropic,
		BaseURL:      "https://api.anthropic.com",
		APIKey:       "sk-test",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-providers/"+providerID.Hex()+"/test", nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, body)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "unsupported" {
		t.Fatalf("expected unsupported status, got %v", body)
	}
	if !strings.Contains(body["message"], "coming soon") {
		t.Fatalf("expected coming soon message, got %v", body)
	}
}
```

- [ ] **Step 2: Run provider tests and verify they fail before implementation**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'TestModelProviderAPI_TestProvider' -count=1
```

Expected: FAIL because current provider test returns only `{"status":"ok"}` and never calls upstream.

- [ ] **Step 3: Add OpenAI-compatible provider connectivity helper**

In `backend/internal/llm/openai.go`, add imports if missing:

```go
	"net/http"
	"time"
```

`net/http` is already imported in this file. Add this helper after `normalizeBaseURL`:

```go
// TestOpenAICompatibleProvider verifies that an OpenAI-compatible provider
// responds to an authenticated /models request. It intentionally does not list
// or store models; it only verifies base URL and API key reachability.
func TestOpenAICompatibleProvider(ctx context.Context, baseURL string, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key is not set")
	}

	baseURL = normalizeBaseURL(baseURL)
	if baseURL == "" {
		return fmt.Errorf("base URL is not set")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create provider test request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("provider request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("provider returned status %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 4: Update model provider test handler**

In `backend/internal/api/handlers/model_settings.go`, add import:

```go
	"agentstore/internal/llm"
```

Replace `TestProvider` response block with:

```go
	if provider.ProviderType != models.ProviderTypeOpenAICompatible {
		respondWithJSON(w, http.StatusOK, map[string]string{
			"status":  "unsupported",
			"message": "Provider connectivity testing for Anthropic and Gemini is coming soon. Use an OpenAI-compatible provider for P0 chat.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	if err := llm.TestOpenAICompatibleProvider(ctx, provider.BaseURL, provider.APIKey); err != nil {
		respondWithError(w, http.StatusBadGateway, fmt.Sprintf("provider connectivity test failed: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "OpenAI-compatible provider responded successfully.",
	})
```

- [ ] **Step 5: Run provider tests and verify they pass**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'TestModelProviderAPI_TestProvider' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run affected backend tests**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/llm -run 'TestModelProviderAPI|TestModelDefaults|TestRouter' -count=1
```

Expected: PASS.

- [ ] **Step 7: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add backend/internal/llm/openai.go backend/internal/api/handlers/model_settings.go backend/internal/api/handlers/model_settings_test.go
git commit -m "feat: verify openai-compatible provider connectivity"
```

---

## Task 9: Model and Agent Capability Honesty

**Files:**
- Modify: `frontend/src/pages/app/settings/ModelSettingsTab.tsx`
- Modify: `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx`
- Modify: `frontend/src/pages/app/settings/AgentsTab.tsx`
- Modify: `frontend/src/pages/app/settings/AgentsTab.test.tsx`
- Depends on: Task 8 if provider-test message shape is displayed.

- [ ] **Step 1: Update ModelSettings tests**

Append to `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx`:

```tsx
  it('presents OpenAI-compatible as the only new P0 provider option and labels existing unsupported providers as coming soon', async () => {
    const user = userEvent.setup();
    apiMocks.listProviders.mockResolvedValue([
      { id: 'provider-1', tenantId: 'tenant-1', name: 'OpenAI', providerType: 'openai_compatible', baseUrl: 'https://api.example.com/v1', apiKeyPreview: 'sk-***1234', enabled: true, createdAt: '', updatedAt: '' },
      { id: 'provider-2', tenantId: 'tenant-1', name: 'Anthropic', providerType: 'anthropic', baseUrl: 'https://api.anthropic.com', apiKeyPreview: 'sk-***2222', enabled: true, createdAt: '', updatedAt: '' },
    ]);

    renderModelSettingsTab();

    expect(await screen.findByText('OpenAI-compatible')).toBeInTheDocument();
    expect(screen.getByText('Anthropic (coming soon)')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'New Provider' }));
    const providerType = screen.getByLabelText('Type');
    expect(providerType).toHaveValue('openai_compatible');
    expect(screen.queryByRole('option', { name: 'Anthropic' })).not.toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Google Gemini' })).not.toBeInTheDocument();
  });

  it('allows only text models in the P0 new model form', async () => {
    const user = userEvent.setup();
    renderModelSettingsTab();

    await user.click(await screen.findByRole('button', { name: 'New Model' }));

    const modality = screen.getByLabelText('Modality');
    expect(modality).toHaveValue('text');
    expect(screen.getByRole('option', { name: 'Text chat' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Image' })).not.toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Video' })).not.toBeInTheDocument();
    expect(screen.getByText('Image and video models are coming soon and are hidden from P0 chat setup.')).toBeInTheDocument();
  });
```

Add missing import at top:

```tsx
import userEvent from '@testing-library/user-event';
```

- [ ] **Step 2: Update AgentsTab tests**

Append to `frontend/src/pages/app/settings/AgentsTab.test.tsx`:

```tsx
  it('limits new Agent capability controls to text chat for P0', async () => {
    const user = userEvent.setup();
    renderAgentsTab();

    await screen.findByText(/No agents yet/i);
    await user.click(screen.getByRole('button', { name: /new agent/i }));

    expect(screen.getByLabelText('Text chat')).toBeChecked();
    expect(screen.queryByLabelText('image generation')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('video generation')).not.toBeInTheDocument();
    expect(screen.queryByText('Image Credits')).not.toBeInTheDocument();
    expect(screen.queryByText('Video Credits')).not.toBeInTheDocument();
    expect(screen.getByText('Image and video generation are coming soon. P0 Agents use text chat only.')).toBeInTheDocument();
  });

  it('labels existing image and video capabilities as coming soon without advertising them as usable', async () => {
    apiMocks.list.mockResolvedValue([
      { id: 'agent-1', name: 'Visual Agent', slug: 'visual-agent', category: 'Creative', description: 'Has future capabilities', systemPrompt: 'Prompt', visibility: 'private', capabilities: ['text_chat', 'image_generation', 'video_generation'], creditCost: { textMessageCredits: 1, imageGenerationCredits: 5, videoGenerationCredits: 20 }, createdAt: '', updatedAt: '' },
    ]);

    renderAgentsTab();

    expect(await screen.findByText('Visual Agent')).toBeInTheDocument();
    expect(screen.getByText('text chat')).toBeInTheDocument();
    expect(screen.getByText('image generation (coming soon)')).toBeInTheDocument();
    expect(screen.getByText('video generation (coming soon)')).toBeInTheDocument();
  });
```

- [ ] **Step 3: Run capability UI tests and verify they fail before implementation**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/settings/ModelSettingsTab.test.tsx src/pages/app/settings/AgentsTab.test.tsx
```

Expected: FAIL on new labels and hidden options.

- [ ] **Step 4: Implement ModelSettings capability honesty**

In `frontend/src/pages/app/settings/ModelSettingsTab.tsx`, add helper constants before component:

```tsx
const providerTypeLabels: Record<ProviderType, string> = {
  openai_compatible: 'OpenAI-compatible',
  anthropic: 'Anthropic (coming soon)',
  gemini: 'Google Gemini (coming soon)',
};

const modelModalityLabels: Record<ModelModality, string> = {
  text: 'Text chat',
  image: 'Image (coming soon)',
  video: 'Video (coming soon)',
};
```

Replace `providerTypeOptions` with:

```tsx
  const providerTypeOptions: { value: ProviderType; label: string }[] = [
    { value: 'openai_compatible', label: 'OpenAI-compatible' },
  ];
```

Replace `modalityOptions` with:

```tsx
  const modalityOptions: { value: ModelModality; label: string }[] = [
    { value: 'text', label: 'Text chat' },
  ];
```

Add accessible labels to selects:

```tsx
<label htmlFor="providerType" className="block text-sm text-dark-400 mb-1">Type</label>
<select id="providerType" ...>
```

```tsx
<label htmlFor="modelModality" className="block text-sm text-dark-400 mb-1">Modality</label>
<select id="modelModality" ...>
```

Below the modality select, add:

```tsx
<p className="mt-2 text-xs text-dark-500">Image and video models are coming soon and are hidden from P0 chat setup.</p>
```

Replace provider list raw type render:

```tsx
{providerTypeLabels[provider.providerType] ?? provider.providerType} • {provider.baseUrl}
```

Replace model modality badge:

```tsx
{modelModalityLabels[model.modality] ?? model.modality}
```

Update test result success message display to prefer backend message if present. First adjust frontend API type in `frontend/src/api/client.ts` if needed:

```ts
testProvider: (id: string) =>
  api.post<{ status: string; message?: string }>(`/tenant/model-providers/${id}/test`).then(r => r.data),
```

Then in `ModelSettingsTab.tsx`:

```tsx
    onSuccess: (result) => {
      const success = result.status === 'ok';
      setTestResult({ success, message: result.message || result.status });
      setTimeout(() => setTestResult(null), 5000);
    },
```

- [ ] **Step 5: Implement AgentsTab capability honesty**

In `frontend/src/pages/app/settings/AgentsTab.tsx`, replace capability options with:

```tsx
  const capabilityOptions: { value: AgentCapability; label: string; disabled?: boolean }[] = [
    { value: 'text_chat', label: 'Text chat' },
  ];
```

Update the capabilities JSX:

```tsx
          <div>
            <label className="block text-sm text-dark-400 mb-1">Capabilities</label>
            <div className="flex gap-4">
              {capabilityOptions.map(cap => (
                <label key={cap.value} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={form.capabilities.includes(cap.value)}
                    onChange={e => {
                      if (e.target.checked) {
                        setForm({...form, capabilities: Array.from(new Set([...form.capabilities, cap.value]))});
                      } else {
                        setForm({...form, capabilities: form.capabilities.filter(c => c !== cap.value)});
                      }
                    }}
                    className="rounded"
                    aria-label={cap.label}
                  />
                  {cap.label}
                </label>
              ))}
            </div>
            <p className="mt-2 text-xs text-dark-500">Image and video generation are coming soon. P0 Agents use text chat only.</p>
          </div>
```

Replace credit grid with text-only input:

```tsx
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Text Credits</label>
              <input
                type="number"
                min="0"
                value={form.creditCost.textMessageCredits}
                onChange={e => setForm({...form, creditCost: {...form.creditCost, textMessageCredits: parseInt(e.target.value) || 0}})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              />
            </div>
          </div>
```

Add helper before return:

```tsx
  const renderCapabilityLabel = (capability: AgentCapability) => {
    if (capability === 'text_chat') return 'text chat';
    if (capability === 'image_generation') return 'image generation (coming soon)';
    if (capability === 'video_generation') return 'video generation (coming soon)';
    return capability;
  };
```

Replace list capability pills:

```tsx
                    {agent.capabilities?.map(cap => (
                      <span key={cap} className="text-xs px-2 py-0.5 bg-dark-800 rounded">{renderCapabilityLabel(cap)}</span>
                    ))}
```

When saving, force P0-created data to text-chat only while preserving text credits:

```tsx
    const data = {
      ...form,
      slug,
      capabilities: ['text_chat'] as AgentCapability[],
      creditCost: {
        textMessageCredits: form.creditCost.textMessageCredits,
        imageGenerationCredits: 0,
        videoGenerationCredits: 0,
      },
      suggestedPrompts: form.welcomeMessage ? [form.welcomeMessage] : [],
    };
```

- [ ] **Step 6: Run capability UI tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/settings/ModelSettingsTab.test.tsx src/pages/app/settings/AgentsTab.test.tsx
```

Expected: PASS.

- [ ] **Step 7: Run related Dashboard test for ordinary-user hidden image/video labels**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/DashboardPage.test.tsx src/pages/app/settings/AgentsTab.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx
```

Expected: PASS.

- [ ] **Step 8: Checkpoint**

Do not commit unless explicitly asked. If asked:

```bash
git add frontend/src/pages/app/settings/ModelSettingsTab.tsx frontend/src/pages/app/settings/ModelSettingsTab.test.tsx frontend/src/pages/app/settings/AgentsTab.tsx frontend/src/pages/app/settings/AgentsTab.test.tsx frontend/src/api/client.ts
git commit -m "feat: clarify p0 model and agent capabilities"
```

---

## Task 10: Final Verification and Smoke Review

**Files:**
- Modify only if failures expose real issues in touched files.
- Test suites across frontend/backend.

- [ ] **Step 1: Run all targeted frontend tests**

Run:

```bash
cd frontend && npm test -- --run \
  src/components/app/EmptyState.test.tsx \
  src/components/app/ErrorState.test.tsx \
  src/components/app/CreditExplainer.test.tsx \
  src/components/app/MobileBottomNav.test.tsx \
  src/components/app/MarkdownMessage.test.tsx \
  src/components/Layout.test.tsx \
  src/pages/app/DashboardPage.test.tsx \
  src/pages/app/ChatPage.test.tsx \
  src/pages/app/BuyCreditsPage.test.tsx \
  src/pages/app/BillingSuccessPage.test.tsx \
  src/pages/app/OnboardingPage.test.tsx \
  src/pages/app/settings/AgentsTab.test.tsx \
  src/pages/app/settings/ModelSettingsTab.test.tsx \
  src/pages/admin/components/LaunchChecklist.test.tsx \
  src/pages/admin/DashboardPage.test.tsx \
  src/components/AdminLayout.test.tsx
```

Expected: PASS.

- [ ] **Step 2: Run frontend type-check**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS.

- [ ] **Step 3: Run backend focused tests**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/llm ./internal/validation/... -count=1
```

Expected: PASS.

- [ ] **Step 4: Run required project build verification**

Run:

```bash
cd backend && go build ./...
```

Expected: PASS.

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS.

- [ ] **Step 5: Run Playwright smoke if environment is available**

Run:

```bash
cd frontend && npx playwright test e2e/admin.spec.ts e2e/navigation.spec.ts e2e/smoke.spec.ts
```

Expected: PASS if Playwright browsers and dev server setup are available. If the environment is not available, record the exact blocker and do not claim E2E verification passed.

- [ ] **Step 6: Manual smoke walkthrough**

Start the app using the project’s normal dev commands in two terminals if local dependencies are available:

```bash
cd backend && go run ./cmd/server
```

```bash
cd frontend && npm run dev
```

Then manually verify:

1. `/dashboard` shows marketplace hero, search, categories, Agent cards, and credit safety copy.
2. Mobile viewport shows bottom nav for Agents, Credits, History, Settings.
3. Search filters Agents.
4. Clicking Agent CTA opens `/chat/:slug`.
5. Clicking suggested prompt opens chat with the prompt prefilled.
6. Chat renders Markdown-style content.
7. Chat composer shows cost and failed-response safety copy.
8. Insufficient credits links to `/buy-credits?returnTo=...`.
9. Buy Credits explains bundles and stores checkout return path.
10. Billing success lets the user continue the conversation.
11. Onboarding goal → Agent → prompt → credits flow ends in chat.
12. Admin dashboard shows Launch Checklist.
13. Settings model provider UI only offers OpenAI-compatible for new providers and text chat for new models.
14. Agent settings shows image/video as coming soon, not as P0 usable controls.

- [ ] **Step 7: Report verification honestly**

In the final implementation report, include:

```markdown
## Verification

- Frontend targeted tests: PASS/FAIL with command output summary
- Frontend type-check: PASS/FAIL with command output summary
- Backend focused tests: PASS/FAIL with command output summary
- Backend build: PASS/FAIL with command output summary
- Playwright smoke: PASS/FAIL/SKIPPED with exact reason
- Manual smoke: PASS/FAIL/SKIPPED with exact reason
```

- [ ] **Step 8: Checkpoint**

Do not commit unless explicitly asked. If asked after all verification passes:

```bash
git add frontend backend docs
git commit -m "feat: polish agentstore user-ready experience"
```

---

## Self-review

### Spec coverage

- Current development assessment and scope are captured in this plan header, scope notes, and file map.
- Ordinary-user navigation and mobile bottom nav are covered by Task 2.
- Marketplace dashboard search, category chips, credit explanation, Agent card copy, prompt entry, and ordinary-user hidden image/video labels are covered by Task 3.
- Chat Markdown, prompt query, credit safety, retry, error handling, and terminology are covered by Task 4.
- Credits purchase explanation and billing success continuation are covered by Task 5.
- First-success onboarding is covered by Task 6.
- Admin Launch Checklist is covered by Task 7.
- Provider and multimodal capability honesty are covered by Tasks 8 and 9.
- Verification expectations are covered by Task 10.

### Placeholder scan

This plan does not contain TBD/TODO placeholders. Each task has exact files, test code, implementation code, commands, expected outcomes, and checkpoint instructions.

### Type and API consistency

- `Agent`, `CreditBundle`, `UsageSummary`, provider, model, and chat stream shapes match the inspected frontend types and backend responses.
- `tenantModelsApi.testProvider` is updated to accept an optional `message` while preserving existing `status` behavior.
- Backend provider test preserves existing provider enum support and returns `unsupported` for Anthropic/Gemini rather than rejecting stored data.
- No backend schema changes are required for this P0 pass.
