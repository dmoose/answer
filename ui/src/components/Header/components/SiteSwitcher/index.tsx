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

import currentSiteStore from '@/stores/currentSite';

const SiteSwitcher: FC = () => {
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
    <Dropdown className="ms-auto me-3">
      <Dropdown.Toggle
        variant="link"
        className="nav-link text-capitalize text-nowrap p-0"
        id="site-switcher">
        {currentSite?.name || 'Select Site'}
      </Dropdown.Toggle>
      <Dropdown.Menu align="end">
        {sites.map((site) => (
          <Dropdown.Item
            key={site.id}
            active={site.id === currentSite?.id}
            onClick={() => handleSelect(site.slug)}>
            {site.name}
            {site.description ? (
              <small className="d-block text-muted">{site.description}</small>
            ) : null}
          </Dropdown.Item>
        ))}
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default SiteSwitcher;
