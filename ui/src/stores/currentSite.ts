import { create } from 'zustand';

import Storage from '@/utils/storage';

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
  setCurrentSite: (site: Site) => void;
  setSites: (sites: Site[]) => void;
}

const currentSiteStore = create<CurrentSiteState>((set) => ({
  currentSite: null,
  sites: [],
  setCurrentSite: (site) => {
    Storage.set('CURRENT_SITE_ID', site.id);
    set({ currentSite: site });
  },
  setSites: (sites) => {
    set({ sites });
    if (sites.length > 0) {
      const storedId = Storage.get('CURRENT_SITE_ID');
      const match = sites.find((s) => s.id === storedId);
      if (match) {
        set({ currentSite: match });
      } else {
        Storage.set('CURRENT_SITE_ID', sites[0].id);
        set({ currentSite: sites[0] });
      }
    }
  },
}));

export default currentSiteStore;
