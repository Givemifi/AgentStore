import { useEffect, useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Users,
  Building2,
  Mail,
  FileText,
  CreditCard,
  DollarSign,
  Activity,
  Settings,
  Info,
  ArrowLeft,
  Shield,
  Code2,
  Paintbrush,
  Tag,
  Megaphone,
  UserPlus,
  BarChart3,
  Cpu,
  Sparkles,
  Menu,
  X,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import { messagesApi } from '../api/client';
import LoadingSpinner from './LoadingSpinner';

import { Navigate } from 'react-router-dom';

interface NavItem {
  path: string;
  icon: LucideIcon;
  label: string;
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

export default function AdminLayout() {
  const location = useLocation();
  const { isRootTenant, role, isTenantReady } = useTenant();
  const [unreadCount, setUnreadCount] = useState(0);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  useEffect(() => {
    if (!isRootTenant) {
      return;
    }

    messagesApi.unreadCount()
      .then((data) => setUnreadCount(data.count))
      .catch(() => { /* non-critical: badge just won't show */ });
  }, [isRootTenant]);

  // Close mobile drawer on route change
  useEffect(() => {
    setMobileNavOpen(false);
  }, [location.pathname]);

  if (!isTenantReady) {
    return (
      <div className="min-h-screen bg-dark-950 flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (!isRootTenant) {
    return <Navigate to="/dashboard" replace />;
  }

  const isActive = (path: string) => location.pathname === path;

  const navGroups: NavGroup[] = [
    {
      title: 'Overview',
      items: [
        { path: '/admin', icon: LayoutDashboard, label: 'Dashboard' },
        { path: '/admin/messages', icon: Mail, label: 'Messages' },
      ],
    },
    {
      title: 'People',
      items: [
        { path: '/admin/users', icon: Users, label: 'Users' },
        { path: '/admin/tenants', icon: Building2, label: 'Tenants' },
        { path: '/admin/members', icon: UserPlus, label: 'Root Members' },
      ],
    },
    {
      title: 'Revenue',
      items: [
        { path: '/admin/plans', icon: CreditCard, label: 'Plans' },
        { path: '/admin/financial', icon: DollarSign, label: 'Financial' },
        { path: '/admin/promotions', icon: Tag, label: 'Promotions' },
      ],
    },
    {
      title: 'Growth',
      items: [
        { path: '/admin/pm', icon: BarChart3, label: 'Product' },
        { path: '/admin/annotations', icon: Sparkles, label: 'Annotations' },
        { path: '/admin/announcements', icon: Megaphone, label: 'Announcements' },
      ],
    },
    {
      title: 'System',
      items: [
        { path: '/admin/health', icon: Activity, label: 'System Health' },
        { path: '/admin/logs', icon: FileText, label: 'Logs' },
        { path: '/admin/config', icon: Settings, label: 'Configuration' },
        { path: '/admin/branding', icon: Paintbrush, label: 'Branding' },
        { path: '/admin/api', icon: Code2, label: 'API' },
        ...(role === 'owner' ? [
          { path: '/admin/llm-config', icon: Cpu, label: 'LLM Config' },
          { path: '/admin/payment-config', icon: CreditCard, label: 'Payment Config' },
        ] : []),
        { path: '/admin/about', icon: Info, label: 'About' },
      ],
    },
  ];

  const sidebarNav = (
    <nav className="space-y-5">
      {navGroups.map((group) => (
        <div key={group.title}>
          <p className="px-2 mb-1 text-[11px] font-semibold uppercase tracking-wider text-dark-500">
            {group.title}
          </p>
          <div className="space-y-px">
            {group.items.map((item) => {
              const active = isActive(item.path);
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`relative flex items-center gap-2.5 px-2 py-1.5 rounded-md text-[13px] transition-colors ${
                    active
                      ? 'bg-white/5 text-white'
                      : 'text-dark-400 hover:text-white hover:bg-white/5'
                  }`}
                >
                  {active && <span className="absolute left-0 top-1/2 -translate-y-1/2 h-4 w-0.5 rounded-full bg-primary-500" />}
                  <item.icon className="w-3.5 h-3.5 shrink-0" />
                  <span>{item.label}</span>
                  {item.label === 'Messages' && unreadCount > 0 && (
                    <span className="ml-auto text-[10px] bg-primary-500 text-white rounded-full px-1.5 py-px min-w-[16px] text-center">
                      {unreadCount}
                    </span>
                  )}
                </Link>
              );
            })}
          </div>
        </div>
      ))}
    </nav>
  );

  return (
    <div className="min-h-screen bg-dark-950">
      {/* Admin Header */}
      <header className="sticky top-0 z-50 bg-dark-950/95 border-b border-white/8">
        <div className="px-4 sm:px-6">
          <div className="flex items-center justify-between h-12">
            <div className="flex items-center gap-3">
              {/* Mobile nav toggle */}
              <button
                type="button"
                onClick={() => setMobileNavOpen(!mobileNavOpen)}
                className="lg:hidden text-dark-400 hover:text-white transition-colors"
                aria-label="Toggle navigation"
              >
                {mobileNavOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
              </button>
              <Link to="/dashboard" className="flex items-center gap-1.5 text-dark-400 hover:text-white transition-colors">
                <ArrowLeft className="w-3.5 h-3.5" />
                <span className="text-[13px]">Back to App</span>
              </Link>
              <div className="h-4 w-px bg-white/8" />
              <div className="flex items-center gap-2">
                <div className="w-5 h-5 rounded-md bg-primary-500 flex items-center justify-center">
                  <Shield className="w-3 h-3 text-white" />
                </div>
                <span className="font-semibold text-white text-[13px]">Admin</span>
              </div>
            </div>
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Desktop sidebar */}
        <aside className="hidden lg:block w-60 shrink-0 min-h-[calc(100vh-3rem)] border-r border-white/8 p-3">
          {sidebarNav}
        </aside>

        {/* Mobile drawer */}
        {mobileNavOpen && (
          <div className="fixed inset-0 top-12 z-40 lg:hidden">
            <div className="absolute inset-0 bg-black/60" onClick={() => setMobileNavOpen(false)} />
            <aside className="relative z-10 w-60 h-full bg-dark-950 border-r border-white/8 p-3 overflow-y-auto">
              {sidebarNav}
            </aside>
          </div>
        )}

        {/* Main Content */}
        <main className="flex-1 min-w-0 p-4 sm:p-6">
          <div className="max-w-5xl mx-auto">
            <Outlet context={{ setUnreadCount }} />
          </div>
        </main>
      </div>
    </div>
  );
}
