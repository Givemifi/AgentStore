import { Link, useLocation } from 'react-router-dom';
import { Shield } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export interface MobileBottomNavItem {
  path: string;
  label: string;
  icon: LucideIcon;
}

interface MobileBottomNavProps {
  items: MobileBottomNavItem[];
  hasAdminAccess: boolean;
}

interface RenderedMobileNavItem extends MobileBottomNavItem {
  isActive: (pathname: string, search: string) => boolean;
}

function isItemActive(item: MobileBottomNavItem, pathname: string, search: string) {
  if (item.path === '/dashboard') {
    return pathname === '/dashboard' || pathname.startsWith('/chat');
  }
  if (item.path === '/buy-credits') {
    return pathname === '/buy-credits' || pathname === '/plan' || pathname.startsWith('/billing');
  }
  if (item.path === '/activity') {
    return pathname === '/activity';
  }
  if (item.path === '/settings') {
    return pathname.startsWith('/settings');
  }
  if (item.path.includes('?')) {
    return `${pathname}${search}` === item.path;
  }
  return pathname === item.path;
}

export default function MobileBottomNav({ items, hasAdminAccess }: MobileBottomNavProps) {
  const location = useLocation();
  const renderedItems: RenderedMobileNavItem[] = items.map((item) => ({
    ...item,
    isActive: (pathname, search) => isItemActive(item, pathname, search),
  }));

  if (hasAdminAccess) {
    renderedItems.push({
      path: '/admin',
      label: 'Admin',
      icon: Shield,
      isActive: (pathname) => pathname.startsWith('/admin'),
    });
  }

  return (
    <nav
      className="fixed inset-x-0 bottom-0 z-50 border-t border-dark-800 bg-dark-900/95 px-2 pt-2 shadow-[0_-12px_40px_-24px_rgba(0,0,0,0.95)] backdrop-blur-xl md:hidden"
      style={{ paddingBottom: 'max(0.5rem, env(safe-area-inset-bottom))' }}
      aria-label="Primary mobile navigation"
    >
      <div
        className="mx-auto grid max-w-md gap-1"
        style={{ gridTemplateColumns: `repeat(${Math.max(renderedItems.length, 1)}, minmax(0, 1fr))` }}
      >
        {renderedItems.map((item) => {
          const active = item.isActive(location.pathname, location.search);
          return (
            <Link
              key={item.path}
              to={item.path}
              aria-current={active ? 'page' : undefined}
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
