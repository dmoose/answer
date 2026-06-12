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

import { Icon } from '@/components';
import currentSiteStore from '@/stores/currentSite';

import './index.scss';

// Site picker block at the top of the left nav. Hidden when only one site.
const SiteContextPicker: FC = () => {
  const { currentSite, sites } = currentSiteStore();

  if (sites.length <= 1) {
    return null;
  }

  const handleSelect = (slug: string) => {
    if (slug === 'default') {
      window.location.href = '/';
    } else {
      window.location.href = `/s/${slug}`;
    }
  };

  return (
    <Dropdown className="mb-3 site-context-picker">
      <Dropdown.Toggle
        variant="light"
        className="w-100 d-flex align-items-center justify-content-between site-context-toggle">
        <span className="d-flex align-items-center flex-grow-1 text-truncate">
          {currentSite?.icon_url ? (
            <img
              src={currentSite.icon_url}
              alt=""
              width={24}
              height={24}
              className="me-2 rounded site-context-icon"
            />
          ) : (
            <Icon
              name="grid-fill"
              className="me-2 site-context-icon-fallback"
            />
          )}
          <span className="fw-semibold text-truncate">
            {currentSite?.name ?? 'Select site'}
          </span>
        </span>
        <Icon name="chevron-expand" className="ms-2 small text-secondary" />
      </Dropdown.Toggle>
      <Dropdown.Menu className="site-context-menu">
        <Dropdown.Header className="text-uppercase small text-secondary">
          Sites
        </Dropdown.Header>
        {sites.map((site) => (
          <Dropdown.Item
            key={site.id}
            active={site.id === currentSite?.id}
            onClick={() => handleSelect(site.slug)}
            className="py-2">
            <div className="d-flex align-items-start">
              <span className="fw-semibold flex-grow-1">{site.name}</span>
              {site.id === currentSite?.id && (
                <Icon name="check2" className="ms-2 text-primary" />
              )}
            </div>
            {site.description ? (
              <div className="small text-secondary text-wrap">
                {site.description}
              </div>
            ) : null}
          </Dropdown.Item>
        ))}
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default SiteContextPicker;
