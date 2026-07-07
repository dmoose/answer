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
import { Form, Button, Card, Badge, Modal } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { Navigate } from 'react-router-dom';

import { useToast } from '@/hooks';
import { Modal as ConfirmModal } from '@/components';
import {
  getNetworkProfile,
  createProject,
  updateProject,
  deleteProject,
  type ProfileProject,
} from '@/services';
import { loggedUserInfoStore, featuresControlStore } from '@/stores';

const STATUS_OPTS = [
  { value: 1, labelKey: 'status_active' },
  { value: 2, labelKey: 'status_paused' },
  { value: 9, labelKey: 'status_archived' },
];

interface FormState {
  title: string;
  description: string;
  repo_url: string;
  status: number;
  seeking_help: boolean;
}

const empty: FormState = {
  title: '',
  description: '',
  repo_url: '',
  status: 1,
  seeking_help: false,
};

const ProjectsSettings: FC = () => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'settings.projects',
  });
  const Toast = useToast();
  const user = loggedUserInfoStore((s) => s.user);
  const directoryEnabled = featuresControlStore((s) => s.directory_enabled);
  const [projects, setProjects] = useState<ProfileProject[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<ProfileProject | null>(null);
  const [form, setForm] = useState<FormState>(empty);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  function reload() {
    if (!user?.id) return;
    setLoading(true);
    getNetworkProfile(user.id)
      .then((p) => setProjects(p?.projects ?? []))
      .catch(() => setProjects([]))
      .finally(() => setLoading(false));
  }

  useEffect(reload, [user?.id]);

  function openAdd() {
    setEditing(null);
    setForm(empty);
    setShowModal(true);
  }

  function openEdit(p: ProfileProject) {
    setEditing(p);
    setForm({
      title: p.title,
      description: p.description,
      repo_url: p.repo_url,
      status: p.status,
      seeking_help: p.seeking_help,
    });
    setShowModal(true);
  }

  async function save() {
    setSaving(true);
    try {
      if (editing) {
        await updateProject(editing.id, form);
        Toast.onShow({ msg: t('update_success'), variant: 'success' });
      } else {
        await createProject(form);
        Toast.onShow({ msg: t('add_success'), variant: 'success' });
      }
      setShowModal(false);
      reload();
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

  function remove(p: ProfileProject) {
    ConfirmModal.confirm({
      title: t('delete_title'),
      content: t('delete_confirm', { title: p.title }),
      cancelBtnVariant: 'link',
      confirmBtnVariant: 'danger',
      confirmText: t('delete', { keyPrefix: 'btns' }),
      onConfirm: async () => {
        try {
          await deleteProject(p.id);
          Toast.onShow({ msg: t('delete_success'), variant: 'success' });
          reload();
        } catch (e: unknown) {
          const msg =
            typeof e === 'object' && e && 'msg' in e
              ? String((e as { msg: unknown }).msg)
              : t('delete_failed');
          Toast.onShow({ msg, variant: 'danger' });
        }
      },
    });
  }

  if (!directoryEnabled) {
    return <Navigate to="/users/settings/profile" replace />;
  }

  return (
    <>
      <div className="d-flex justify-content-between align-items-center mb-4">
        <h3 className="mb-0">{t('page_title')}</h3>
        <Button variant="primary" onClick={() => openAdd()}>
          {t('add_project')}
        </Button>
      </div>
      <p className="text-secondary mb-4">{t('page_desc')}</p>

      {loading ? (
        <div className="text-secondary">{t('loading')}</div>
      ) : projects.length === 0 ? (
        <div className="text-secondary">{t('empty')}</div>
      ) : (
        projects.map((p) => (
          <Card key={p.id} className="mb-2">
            <Card.Body>
              <div className="d-flex justify-content-between align-items-start gap-2">
                <div className="flex-grow-1">
                  <div className="d-flex flex-wrap gap-2 align-items-center mb-1">
                    <strong>{p.title}</strong>
                    <Badge bg={p.status === 1 ? 'success' : 'secondary'}>
                      {t(
                        STATUS_OPTS.find((s) => s.value === p.status)
                          ?.labelKey ?? 'status_unknown',
                      )}
                    </Badge>
                    {p.seeking_help && (
                      <Badge bg="warning">{t('seeking_help')}</Badge>
                    )}
                  </div>
                  {p.description && (
                    <div className="small text-secondary mb-1">
                      {p.description}
                    </div>
                  )}
                  {p.repo_url && (
                    <a
                      href={p.repo_url}
                      target="_blank"
                      rel="noreferrer"
                      className="small">
                      {p.repo_url}
                    </a>
                  )}
                </div>
                <div className="d-flex gap-1">
                  <Button
                    size="sm"
                    variant="outline-secondary"
                    onClick={() => openEdit(p)}>
                    {t('edit', { keyPrefix: 'btns' })}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline-danger"
                    onClick={() => remove(p)}>
                    {t('delete', { keyPrefix: 'btns' })}
                  </Button>
                </div>
              </div>
            </Card.Body>
          </Card>
        ))
      )}

      <Modal show={showModal} onHide={() => setShowModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>
            {editing ? t('edit_title') : t('add_title')}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>{t('form.title')}</Form.Label>
            <Form.Control
              type="text"
              value={form.title}
              maxLength={200}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('form.description')}</Form.Label>
            <Form.Control
              as="textarea"
              rows={3}
              value={form.description}
              maxLength={4000}
              onChange={(e) =>
                setForm({ ...form, description: e.target.value })
              }
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('form.repo_url')}</Form.Label>
            <Form.Control
              type="url"
              value={form.repo_url}
              maxLength={512}
              onChange={(e) => setForm({ ...form, repo_url: e.target.value })}
              placeholder="https://…"
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>{t('form.status')}</Form.Label>
            <Form.Select
              value={form.status}
              onChange={(e) =>
                setForm({ ...form, status: Number(e.target.value) })
              }>
              {STATUS_OPTS.map((s) => (
                <option key={s.value} value={s.value}>
                  {t(s.labelKey)}
                </option>
              ))}
            </Form.Select>
          </Form.Group>
          <Form.Check
            type="checkbox"
            id="seeking-help"
            label={t('form.seeking_help')}
            checked={form.seeking_help}
            onChange={(e) =>
              setForm({ ...form, seeking_help: e.target.checked })
            }
          />
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
            disabled={saving || !form.title.trim()}>
            {saving
              ? t('saving')
              : editing
                ? t('save', { keyPrefix: 'btns' })
                : t('add')}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default ProjectsSettings;
