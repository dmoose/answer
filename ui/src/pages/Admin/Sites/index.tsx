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
import { Table, Button, Form, Modal, Badge } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import { useToast } from '@/hooks';
import {
  getAdminSiteList,
  addSite,
  updateSite,
  setUserSiteRole,
} from '@/services';

interface Site {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: number;
}

const ROLES = [
  { id: 1, labelKey: 'role_user' },
  { id: 2, labelKey: 'role_admin' },
  { id: 3, labelKey: 'role_moderator' },
];

const Sites: FC = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'admin.sites' });
  const Toast = useToast();
  const [sites, setSites] = useState<Site[]>([]);
  const [showSiteModal, setShowSiteModal] = useState(false);
  const [editSite, setEditSite] = useState<Site | null>(null);
  const [siteForm, setSiteForm] = useState({
    name: '',
    slug: '',
    description: '',
  });

  const [showRoleModal, setShowRoleModal] = useState(false);
  const [roleSite, setRoleSite] = useState<Site | null>(null);
  const [roleForm, setRoleForm] = useState({ user_id: '', role_id: 2 });

  const loadSites = async () => {
    try {
      const resp = await getAdminSiteList();
      if (Array.isArray(resp)) {
        setSites(resp);
      }
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    loadSites();
  }, []);

  const handleOpenSite = (site?: Site) => {
    if (site) {
      setEditSite(site);
      setSiteForm({
        name: site.name,
        slug: site.slug,
        description: site.description,
      });
    } else {
      setEditSite(null);
      setSiteForm({ name: '', slug: '', description: '' });
    }
    setShowSiteModal(true);
  };

  const handleSaveSite = async () => {
    try {
      if (editSite) {
        await updateSite({ id: editSite.id, ...siteForm });
        Toast.onShow({ msg: t('update_success'), variant: 'success' });
      } else {
        await addSite(siteForm);
        Toast.onShow({ msg: t('create_success'), variant: 'success' });
      }
      setShowSiteModal(false);
      loadSites();
    } catch (e: any) {
      Toast.onShow({
        msg: e?.msg || t('save_failed'),
        variant: 'danger',
      });
    }
  };

  const handleOpenRole = (site: Site) => {
    setRoleSite(site);
    setRoleForm({ user_id: '', role_id: 2 });
    setShowRoleModal(true);
  };

  const handleSaveRole = async () => {
    if (!roleSite) return;
    try {
      await setUserSiteRole({
        user_id: roleForm.user_id,
        site_id: roleSite.id,
        role_id: roleForm.role_id,
      });
      Toast.onShow({ msg: t('role_success'), variant: 'success' });
      setShowRoleModal(false);
    } catch (e: any) {
      Toast.onShow({
        msg: e?.msg || t('role_failed'),
        variant: 'danger',
      });
    }
  };

  return (
    <>
      <h3 className="mb-4">{t('page_title')}</h3>
      <p className="text-secondary mb-3">{t('page_desc')}</p>
      <div className="mb-3">
        <Button variant="primary" size="sm" onClick={() => handleOpenSite()}>
          {t('add_site')}
        </Button>
      </div>
      <Table striped bordered hover size="sm">
        <thead>
          <tr>
            <th>{t('name_label')}</th>
            <th>{t('slug_label')}</th>
            <th>{t('description_label')}</th>
            <th>{t('status_label')}</th>
            <th>{t('actions_label')}</th>
          </tr>
        </thead>
        <tbody>
          {sites.map((site) => (
            <tr key={site.id}>
              <td>{site.name}</td>
              <td>
                <code>/s/{site.slug}</code>
              </td>
              <td>{site.description || '—'}</td>
              <td>
                <Badge bg={site.status === 1 ? 'success' : 'secondary'}>
                  {site.status === 1
                    ? t('status_active')
                    : t('status_suspended')}
                </Badge>
              </td>
              <td>
                <Button
                  variant="outline-secondary"
                  size="sm"
                  className="me-1"
                  onClick={() => handleOpenSite(site)}>
                  {t('edit', { keyPrefix: 'btns' })}
                </Button>
                <Button
                  variant="outline-primary"
                  size="sm"
                  onClick={() => handleOpenRole(site)}>
                  {t('assign_role')}
                </Button>
              </td>
            </tr>
          ))}
          {sites.length === 0 && (
            <tr>
              <td colSpan={5} className="text-center text-muted">
                {t('empty')}
              </td>
            </tr>
          )}
        </tbody>
      </Table>

      <Modal show={showSiteModal} onHide={() => setShowSiteModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>
            {editSite ? t('edit_title') : t('add_title')}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>{t('name_label')}</Form.Label>
            <Form.Control
              type="text"
              value={siteForm.name}
              onChange={(e) =>
                setSiteForm({ ...siteForm, name: e.target.value })
              }
              placeholder={t('name_placeholder')}
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('slug_label')}</Form.Label>
            <Form.Control
              type="text"
              value={siteForm.slug}
              onChange={(e) =>
                setSiteForm({ ...siteForm, slug: e.target.value })
              }
              placeholder={t('slug_placeholder')}
              disabled={!!editSite}
            />
            <Form.Text className="text-muted">{t('slug_help')}</Form.Text>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('description_label')}</Form.Label>
            <Form.Control
              as="textarea"
              rows={2}
              value={siteForm.description}
              onChange={(e) =>
                setSiteForm({ ...siteForm, description: e.target.value })
              }
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowSiteModal(false)}>
            {t('cancel', { keyPrefix: 'btns' })}
          </Button>
          <Button variant="primary" onClick={handleSaveSite}>
            {editSite
              ? t('save', { keyPrefix: 'btns' })
              : t('create', { keyPrefix: 'btns' })}
          </Button>
        </Modal.Footer>
      </Modal>

      <Modal show={showRoleModal} onHide={() => setShowRoleModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>{t('role_title', { name: roleSite?.name })}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>{t('user_id_label')}</Form.Label>
            <Form.Control
              type="text"
              value={roleForm.user_id}
              onChange={(e) =>
                setRoleForm({ ...roleForm, user_id: e.target.value })
              }
              placeholder={t('user_id_label')}
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('role_label')}</Form.Label>
            <Form.Select
              value={roleForm.role_id}
              onChange={(e) =>
                setRoleForm({ ...roleForm, role_id: Number(e.target.value) })
              }>
              {ROLES.map((r) => (
                <option key={r.id} value={r.id}>
                  {t(r.labelKey)}
                </option>
              ))}
            </Form.Select>
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowRoleModal(false)}>
            {t('cancel', { keyPrefix: 'btns' })}
          </Button>
          <Button variant="primary" onClick={handleSaveRole}>
            {t('assign')}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default Sites;
