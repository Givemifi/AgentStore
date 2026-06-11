import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

export function PublicFooter() {
  const { t } = useTranslation('common');
  const year = 2026;

  return (
    <footer className="border-t border-white/8 bg-dark-950 mt-16">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-sm text-dark-500">
            © {year} AgentStore. All rights reserved.
          </p>
          <nav className="flex items-center gap-6 text-sm text-dark-400">
            <Link to="/agents" className="hover:text-white transition-colors">Agents</Link>
            <Link to="/p/privacy-policy" className="hover:text-white transition-colors">Privacy</Link>
            <Link to="/p/terms-of-service" className="hover:text-white transition-colors">Terms</Link>
            <a href="mailto:hello@agentstore.ai" className="hover:text-white transition-colors">Contact</a>
          </nav>
        </div>
      </div>
    </footer>
  );
}
