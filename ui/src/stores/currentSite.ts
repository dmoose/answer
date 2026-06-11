/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

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
