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
import { Table, Button, Form, Modal, Badge } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { Navigate } from 'react-router-dom';

import { useToast } from '@/hooks';
import {
  useAdminProfileTags,
  createAdminProfileTag,
  updateAdminProfileTag,
  type AdminProfileTag,
  type ProfileTagUpsertParams,
} from '@/services';
import { featuresControlStore } from '@/stores';

const KIND_KEY: Record<number, string> = {
  1: 'kind_skill',
  2: 'kind_interest',
  3: 'kind_both',
};

const STATUS_KEY: Record<number, string> = {
  1: 'status_active',
  9: 'status_inactive',
};

const empty: ProfileTagUpsertParams = {
  slug: '',
  name: '',
  kind: 1,
  description: '',
  status: 1,
};

// slugify converts a free-form name to a URL-friendly slug. Admins may still
// hand-edit the slug; this is just a "first guess" when creating a new tag.
function slugify(s: string) {
  return s
    .toLowerCase()
    .replace(/['"]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 64);
}

const NetworkTags: FC = () => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'admin.network_tags',
  });
  const Toast = useToast();
  const directoryEnabled = featuresControlStore((s) => s.directory_enabled);
  const { data: tags, mutate } = useAdminProfileTags();
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<AdminProfileTag | null>(null);
  const [form, setForm] = useState<ProfileTagUpsertParams>(empty);
  const [slugTouched, setSlugTouched] = useState(false);
  const [saving, setSaving] = useState(false);

  // Auto-fill slug from name while it's untouched, so creating a tag feels
  // one-handed; once the user types in the slug field we stop syncing.
  useEffect(() => {
    if (!editing && !slugTouched) {
      setForm((f) => ({ ...f, slug: slugify(f.name) }));
    }
  }, [form.name, editing, slugTouched]);

  if (!directoryEnabled) {
    return <Navigate to="/admin" replace />;
  }

  function openAdd() {
    setEditing(null);
    setForm(empty);
    setSlugTouched(false);
    setShowModal(true);
  }

  function openEdit(tag: AdminProfileTag) {
    setEditing(tag);
    setForm({
      slug: tag.slug,
      name: tag.name,
      kind: tag.kind,
      description: tag.description || '',
      status: tag.status,
    });
    setSlugTouched(true);
    setShowModal(true);
  }

  async function save() {
    setSaving(true);
    try {
      if (editing) {
        await updateAdminProfileTag(editing.id, form);
        Toast.onShow({ msg: t('update_success'), variant: 'success' });
      } else {
        await createAdminProfileTag(form);
        Toast.onShow({ msg: t('create_success'), variant: 'success' });
      }
      setShowModal(false);
      mutate();
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e
          ? String((e as { msg: unknown }).msg)
          : t('save_failed');
      Toast.onShow({ msg, variant: 'danger' });
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <h3 className="mb-4">{t('page_title')}</h3>
      <p className="text-secondary mb-3">{t('page_desc')}</p>
      <div className="mb-3">
        <Button variant="primary" size="sm" onClick={() => openAdd()}>
          {t('add_tag')}
        </Button>
      </div>
      <Table striped bordered hover size="sm">
        <thead>
          <tr>
            <th>{t('name_label')}</th>
            <th>{t('slug_label')}</th>
            <th>{t('kind_label')}</th>
            <th>{t('description_label')}</th>
            <th>{t('status_label')}</th>
            <th>{t('actions_label')}</th>
          </tr>
        </thead>
        <tbody>
          {tags?.map((tag) => (
            <tr key={tag.id}>
              <td>{tag.name}</td>
              <td>
                <code>{tag.slug}</code>
              </td>
              <td>{t(KIND_KEY[tag.kind] ?? 'unknown')}</td>
              <td className="text-secondary">{tag.description || '—'}</td>
              <td>
                <Badge bg={tag.status === 1 ? 'success' : 'secondary'}>
                  {t(STATUS_KEY[tag.status] ?? 'unknown')}
                </Badge>
              </td>
              <td>
                <Button
                  variant="outline-secondary"
                  size="sm"
                  onClick={() => openEdit(tag)}>
                  {t('edit', { keyPrefix: 'btns' })}
                </Button>
              </td>
            </tr>
          ))}
          {tags && tags.length === 0 && (
            <tr>
              <td colSpan={6} className="text-center text-muted">
                {t('empty')}
              </td>
            </tr>
          )}
        </tbody>
      </Table>

      <Modal show={showModal} onHide={() => setShowModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>
            {editing ? t('edit_title') : t('add_title')}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>{t('name_label')}</Form.Label>
            <Form.Control
              type="text"
              value={form.name}
              maxLength={128}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder={t('name_placeholder')}
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('slug_label')}</Form.Label>
            <Form.Control
              type="text"
              value={form.slug}
              maxLength={64}
              disabled={!!editing}
              onChange={(e) => {
                setForm({ ...form, slug: e.target.value });
                setSlugTouched(true);
              }}
              placeholder={t('slug_placeholder')}
            />
            <Form.Text className="text-muted">{t('slug_help')}</Form.Text>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('kind_label')}</Form.Label>
            <Form.Select
              value={form.kind}
              onChange={(e) =>
                setForm({ ...form, kind: Number(e.target.value) })
              }>
              <option value={1}>{t('kind_skill')}</option>
              <option value={2}>{t('kind_interest')}</option>
              <option value={3}>{t('kind_both')}</option>
            </Form.Select>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('description_label')}</Form.Label>
            <Form.Control
              as="textarea"
              rows={2}
              value={form.description}
              maxLength={512}
              onChange={(e) =>
                setForm({ ...form, description: e.target.value })
              }
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('status_label')}</Form.Label>
            <Form.Select
              value={form.status}
              onChange={(e) =>
                setForm({ ...form, status: Number(e.target.value) })
              }>
              <option value={1}>{t('status_active')}</option>
              <option value={9}>{t('status_inactive')}</option>
            </Form.Select>
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="secondary"
            onClick={() => setShowModal(false)}
            disabled={saving}>
            {t('cancel', { keyPrefix: 'btns' })}
          </Button>
          <Button
            variant="primary"
            onClick={() => save()}
            disabled={saving || !form.name.trim() || !form.slug.trim()}>
            {saving
              ? t('saving')
              : editing
                ? t('save', { keyPrefix: 'btns' })
                : t('create', { keyPrefix: 'btns' })}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default NetworkTags;
