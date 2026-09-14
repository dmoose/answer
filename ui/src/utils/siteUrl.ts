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

import { REACT_BASE_PATH } from '@/router/alias';
import type { Site } from '@/stores/currentSite';

/**
 * Absolute URL of a route on a given site. The one place that knows how a
 * site is addressed: its admin-set base_url when present (a sub-site on its
 * own host), otherwise this origin plus the deployment base path plus the
 * /s/<slug> prefix (the default site has no prefix). Use it for every hard
 * navigation or link that leaves the current router basename.
 */
export const siteURL = (
  site: Pick<Site, 'slug' | 'base_url'>,
  route = '',
): string => {
  const path = route && !route.startsWith('/') ? `/${route}` : route;
  const base = site.base_url?.trim().replace(/\/+$/, '');
  if (base) {
    return `${base}${path}`;
  }
  const prefix = site.slug === 'default' ? '' : `/s/${site.slug}`;
  return `${window.location.origin}${REACT_BASE_PATH}${prefix}${path || '/'}`;
};
