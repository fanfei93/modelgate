import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { getSiteSettings } from '../api/admin';
import type { SiteSetting } from '../types/api';

interface SiteConfig {
  siteName: string;
  menuArenaVisible: boolean;
  menuDocsVisible: boolean;
  menuDocsUrl: string;
  loading: boolean;
}

const defaultConfig: SiteConfig = {
  siteName: 'ModelGate',
  menuArenaVisible: true,
  menuDocsVisible: true,
  menuDocsUrl: '/docs',
  loading: true,
};

const SiteConfigContext = createContext<SiteConfig>(defaultConfig);

export function SiteConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<SiteConfig>(defaultConfig);

  useEffect(() => {
    getSiteSettings()
      .then((res) => {
        const settings = res.data.data || [];
        const map = new Map<string, string>();
        for (const s of settings) {
          map.set(s.key, s.value);
        }
        setConfig({
          siteName: map.get('site_name') || 'ModelGate',
          menuArenaVisible: map.get('menu_arena_visible') !== 'false',
          menuDocsVisible: map.get('menu_docs_visible') !== 'false',
          menuDocsUrl: map.get('menu_docs_url') || '/docs',
          loading: false,
        });
      })
      .catch(() => {
        setConfig((c) => ({ ...c, loading: false }));
      });
  }, []);

  return (
    <SiteConfigContext.Provider value={config}>
      {children}
    </SiteConfigContext.Provider>
  );
}

export function useSiteConfig() {
  return useContext(SiteConfigContext);
}
