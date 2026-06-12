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

import { FC } from 'react';
import { Dropdown } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import { Icon } from '@/components';
import { appSwitcherStore } from '@/stores';
import { isDarkTheme } from '@/utils';

// AppSwitcher renders the 3x3 grid in the header far-right. Each card
// links same-window by default (browser modifiers handle new-tab); a small
// ↗ icon in the corner forces a new tab explicitly. data-bs-theme on the
// Dropdown forces Bootstrap to render light/dark based on Answer's chosen
// theme — without it the Popper-rendered menu can fall back to the OS
// color scheme.
const AppSwitcher: FC = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'header' });
  const { enabled, links } = appSwitcherStore();
  if (!enabled || !links.length) {
    return null;
  }

  return (
    <Dropdown
      align="end"
      className="ms-2 app-switcher"
      data-bs-theme={isDarkTheme() ? 'dark' : 'light'}>
      <Dropdown.Toggle
        variant="link"
        className="icon-link nav-link no-toggle d-flex align-items-center justify-content-center p-0"
        aria-label={t('app_switcher.label', { defaultValue: 'Apps' })}>
        <Icon name="grid-3x3-gap-fill" className="lh-1 fs-4" />
      </Dropdown.Toggle>
      <Dropdown.Menu className="app-switcher-menu p-3">
        <div className="text-uppercase small text-secondary mb-2 ms-1">
          {t('app_switcher.header', { defaultValue: 'Apps' })}
        </div>
        <div className="d-flex flex-wrap gap-1">
          {links.map((link) => (
            <div
              key={link.url}
              className="app-switcher-card position-relative d-flex flex-column">
              <a
                href={link.url}
                className="text-decoration-none text-reset d-flex flex-column align-items-center text-center p-2 rounded h-100 app-switcher-primary"
                title={link.description || link.name}>
                {link.icon ? (
                  <img
                    src={link.icon}
                    alt=""
                    width={32}
                    height={32}
                    className="mb-1 rounded"
                  />
                ) : (
                  <Icon
                    name="box-arrow-up-right"
                    className="mb-1 text-secondary fs-4"
                  />
                )}
                <span className="small fw-semibold text-truncate w-100">
                  {link.name}
                </span>
                {link.description && (
                  <span className="text-secondary text-truncate w-100">
                    <small>{link.description}</small>
                  </span>
                )}
              </a>
              <a
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                className="position-absolute top-0 end-0 m-1 small text-secondary text-decoration-none app-switcher-newtab"
                title={t('app_switcher.open_new', {
                  defaultValue: 'Open in new tab',
                })}
                onClick={(e) => e.stopPropagation()}>
                <Icon name="box-arrow-up-right" />
              </a>
            </div>
          ))}
        </div>
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default AppSwitcher;
