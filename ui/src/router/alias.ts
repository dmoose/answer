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

export const REACT_BASE_PATH = process.env.REACT_APP_BASE_URL || '';
export const BASE_ORIGIN = `${window.location.origin}${REACT_BASE_PATH}`;

/**
 * The base path the app is actually running under at runtime: the build-time
 * base plus the multisite prefix (/s/<slug>) when present. REACT_BASE_PATH is
 * baked at build time, so hard navigations built from it alone would drop the
 * site prefix and land on the default site. The site prefix cannot change
 * within a page's lifetime, so deriving it from location on each call is safe.
 */
export const getRuntimeBasePath = (): string => {
  const match = window.location.pathname
    .replace(REACT_BASE_PATH, '')
    .match(/^\/s\/[^/]+/);
  return `${REACT_BASE_PATH}${match ? match[0] : ''}`;
};

/**
 * Origin + runtime base path — for building absolute URLs (share links,
 * logout redirects) that must stay on the current sub-site.
 */
export const getRuntimeBaseOrigin = (): string =>
  `${window.location.origin}${getRuntimeBasePath()}`;

export const RouteAlias = {
  home: '/',
  login: '/users/login',
  signUp: '/users/register',
  inactive: '/users/login?status=inactive',
  accountRecovery: '/users/account-recovery',
  changeEmail: '/users/change-email',
  passwordReset: '/users/password-reset',
  accountActivation: '/users/account-activation',
  activationSuccess: '/users/account-activation/success',
  activationFailed: '/users/account-activation/failed',
  suspended: '/users/account-suspended',
  confirmNewEmail: '/users/confirm-new-email',
  confirmEmail: '/users/confirm-email',
  authLanding: '/users/auth-landing',
};
