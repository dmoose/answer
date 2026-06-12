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

import { FC, useState, useEffect } from 'react';
import { Form, FormControl } from 'react-bootstrap';
import { useSearchParams, useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { Icon } from '@/components';
import currentSiteStore from '@/stores/currentSite';

const SearchInput: FC<{ className?: string }> = ({ className }) => {
  const { t } = useTranslation('translation', { keyPrefix: 'header' });
  const navigate = useNavigate();
  const location = useLocation();
  const [urlSearch] = useSearchParams();
  const q = urlSearch.get('q');
  const [searchStr, setSearch] = useState('');
  const currentSite = currentSiteStore((s) => s.currentSite);
  // Search is site-scoped — reflect that in the placeholder so the user
  // knows what they're searching, especially after a site switch.
  const placeholder = currentSite?.name
    ? `Search ${currentSite.name}`
    : t('search.placeholder');
  const handleInput = (val) => {
    setSearch(val);
  };
  const handleSearch = (evt) => {
    evt.preventDefault();
    if (!searchStr) {
      return;
    }
    const searchUrl = `/search?q=${encodeURIComponent(searchStr)}`;
    navigate(searchUrl);
  };

  useEffect(() => {
    if (q && location.pathname === '/search') {
      handleInput(q);
    }
  }, [q]);

  useEffect(() => {
    // clear search input when navigate to other page
    if (location.pathname !== '/search' && searchStr) {
      setSearch('');
    }
  }, [location.pathname]);
  return (
    <Form
      action="/search"
      className={`w-100 position-relative mx-auto ${className}`}
      onSubmit={handleSearch}>
      <div className="search-wrap" onClick={handleSearch}>
        <Icon name="search" className="search-icon" />
      </div>
      <FormControl
        type="search"
        placeholder={placeholder}
        className="placeholder-search"
        value={searchStr}
        name="q"
        onChange={(e) => handleInput(e.target.value)}
      />
    </Form>
  );
};

export default SearchInput;
