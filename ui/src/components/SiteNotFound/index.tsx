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

import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { REACT_BASE_PATH } from '@/router/alias';

// Rendered in place of the router when the URL names an unknown site. It
// stands alone: no site settings or translations loaded for that slug, so
// every string carries its own fallback and the way out is a hard
// navigation to the network root, outside the dead /s/<slug> basename.
function SiteNotFound() {
  const { t } = useTranslation('translation', { keyPrefix: 'page_error' });
  useEffect(() => {
    document.title = '404';
  }, []);

  return (
    <div className="d-flex flex-column min-vh-100 justify-content-center align-items-center">
      <div
        className="mb-4 text-secondary"
        style={{ fontSize: '120px', lineHeight: 1.2 }}>
        (=‘x‘=)
      </div>
      <h4 className="text-center">
        {t('http_error', { code: '404', defaultValue: 'HTTP Error 404' })}
      </h4>
      <div className="text-center mb-3 fs-5">
        {t('desc_site_missing', {
          defaultValue: "Unfortunately, this site doesn't exist.",
        })}
      </div>
      <div className="text-center">
        <a href={`${REACT_BASE_PATH}/`} className="btn btn-link">
          {t('back_home', { defaultValue: 'Back to homepage' })}
        </a>
      </div>
    </div>
  );
}

export default SiteNotFound;
