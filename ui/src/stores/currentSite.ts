import { create } from 'zustand';

export interface Site {
  id: string;
  name: string;
  slug: string;
  description?: string;
  icon_url?: string;
  base_url?: string;
}

interface CurrentSiteState {
  currentSite: Site | null;
  sites: Site[];
  setSites: (sites: Site[]) => void;
}

function siteSlugFromURL(): string {
  const path = window.location.pathname;
  const match = path.match(/^\/s\/([^/]+)/);
  return match ? match[1] : '';
}

function siteFromHostname(sites: Site[]): Site | null {
  const host = window.location.hostname;
  const sub = host.split('.')[0];
  if (sub && sub !== 'www' && sub !== 'localhost') {
    const match = sites.find((s) => s.slug === sub);
    if (match) return match;
  }
  return null;
}

function resolveCurrentSite(sites: Site[]): Site | null {
  if (sites.length === 0) return null;

  const slug = siteSlugFromURL();
  if (slug) {
    const match = sites.find((s) => s.slug === slug);
    if (match) return match;
  }

  const hostMatch = siteFromHostname(sites);
  if (hostMatch) return hostMatch;

  const defaultSite = sites.find((s) => s.slug === 'default');
  return defaultSite || sites[0];
}

const currentSiteStore = create<CurrentSiteState>((set) => ({
  currentSite: null,
  sites: [],
  setSites: (sites) => {
    const current = resolveCurrentSite(sites);
    set({ sites, currentSite: current });
  },
}));

export default currentSiteStore;
