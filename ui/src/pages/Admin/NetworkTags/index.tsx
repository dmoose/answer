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

const KIND_LABEL: Record<number, string> = {
  1: 'Skill',
  2: 'Interest',
  3: 'Both',
};

const STATUS_LABEL: Record<number, string> = {
  1: 'Active',
  9: 'Inactive',
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

  function openEdit(t: AdminProfileTag) {
    setEditing(t);
    setForm({
      slug: t.slug,
      name: t.name,
      kind: t.kind,
      description: t.description || '',
      status: t.status,
    });
    setSlugTouched(true);
    setShowModal(true);
  }

  async function save() {
    setSaving(true);
    try {
      if (editing) {
        await updateAdminProfileTag(editing.id, form);
        Toast.onShow({ msg: 'Tag updated', variant: 'success' });
      } else {
        await createAdminProfileTag(form);
        Toast.onShow({ msg: 'Tag created', variant: 'success' });
      }
      setShowModal(false);
      mutate();
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e
          ? String((e as { msg: unknown }).msg)
          : 'Failed to save';
      Toast.onShow({ msg, variant: 'danger' });
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <h3 className="mb-4">Profile tags</h3>
      <p className="text-secondary mb-3">
        Curate the skill and interest tags members can attach to their directory
        profile. &ldquo;Both&rdquo; tags appear in either picker. Inactive tags
        stay attached to members who already have them but won&apos;t appear in
        pickers or facets.
      </p>
      <div className="mb-3">
        <Button variant="primary" size="sm" onClick={() => openAdd()}>
          Add tag
        </Button>
      </div>
      <Table striped bordered hover size="sm">
        <thead>
          <tr>
            <th>Name</th>
            <th>Slug</th>
            <th>Kind</th>
            <th>Description</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {tags?.map((t) => (
            <tr key={t.id}>
              <td>{t.name}</td>
              <td>
                <code>{t.slug}</code>
              </td>
              <td>{KIND_LABEL[t.kind] ?? '?'}</td>
              <td className="text-secondary">{t.description || '—'}</td>
              <td>
                <Badge bg={t.status === 1 ? 'success' : 'secondary'}>
                  {STATUS_LABEL[t.status] ?? '?'}
                </Badge>
              </td>
              <td>
                <Button
                  variant="outline-secondary"
                  size="sm"
                  onClick={() => openEdit(t)}>
                  Edit
                </Button>
              </td>
            </tr>
          ))}
          {tags && tags.length === 0 && (
            <tr>
              <td colSpan={6} className="text-center text-muted">
                No tags yet. Add one to start populating the directory facets.
              </td>
            </tr>
          )}
        </tbody>
      </Table>

      <Modal show={showModal} onHide={() => setShowModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>{editing ? 'Edit tag' : 'Add tag'}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>Name</Form.Label>
            <Form.Control
              type="text"
              value={form.name}
              maxLength={128}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Rust"
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Slug</Form.Label>
            <Form.Control
              type="text"
              value={form.slug}
              maxLength={64}
              disabled={!!editing}
              onChange={(e) => {
                setForm({ ...form, slug: e.target.value });
                setSlugTouched(true);
              }}
              placeholder="rust"
            />
            <Form.Text className="text-muted">
              URL-safe identifier — used in member-profile URLs and the
              directory facets.
            </Form.Text>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Kind</Form.Label>
            <Form.Select
              value={form.kind}
              onChange={(e) =>
                setForm({ ...form, kind: Number(e.target.value) })
              }>
              <option value={1}>Skill</option>
              <option value={2}>Interest</option>
              <option value={3}>Both</option>
            </Form.Select>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Description</Form.Label>
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
            <Form.Label>Status</Form.Label>
            <Form.Select
              value={form.status}
              onChange={(e) =>
                setForm({ ...form, status: Number(e.target.value) })
              }>
              <option value={1}>Active</option>
              <option value={9}>Inactive</option>
            </Form.Select>
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="secondary"
            onClick={() => setShowModal(false)}
            disabled={saving}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={() => save()}
            disabled={saving || !form.name.trim() || !form.slug.trim()}>
            {saving ? 'Saving…' : editing ? 'Save' : 'Create'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default NetworkTags;
