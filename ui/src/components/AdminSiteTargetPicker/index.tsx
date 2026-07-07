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

import { FC, useEffect, useState } from 'react';
import { Form } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import { getSiteList } from '@/services';
import type { Site } from '@/stores/currentSite';

interface Props {
  value: string;
  onChange: (siteId: string) => void;
}

// AdminSiteTargetPicker selects which site a per-site presentation setting
// (general, branding, custom css/html) applies to. The empty value edits the
// global default; a site id edits that site's override.
const AdminSiteTargetPicker: FC<Props> = ({ value, onChange }) => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'admin.site_target',
  });
  const [sites, setSites] = useState<Site[]>([]);

  useEffect(() => {
    getSiteList()
      .then((resp) => {
        if (Array.isArray(resp)) {
          setSites(resp);
        }
      })
      .catch(() => {
        setSites([]);
      });
  }, []);

  return (
    <Form.Group className="mb-4" controlId="admin_site_target">
      <Form.Label>{t('label')}</Form.Label>
      <Form.Select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">{t('global_option')}</option>
        {sites.map((site) => (
          <option key={site.id} value={site.id}>
            {site.name}
          </option>
        ))}
      </Form.Select>
      <Form.Text className="text-muted">{t('help')}</Form.Text>
    </Form.Group>
  );
};

export default AdminSiteTargetPicker;
