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
  { id: 1, label: 'User' },
  { id: 2, label: 'Admin' },
  { id: 3, label: 'Moderator' },
];

const Sites: FC = () => {
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
        Toast.onShow({ msg: 'Site updated', variant: 'success' });
      } else {
        await addSite(siteForm);
        Toast.onShow({ msg: 'Site created', variant: 'success' });
      }
      setShowSiteModal(false);
      loadSites();
    } catch (e: any) {
      Toast.onShow({
        msg: e?.msg || 'Error saving site',
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
      Toast.onShow({ msg: 'Role assigned', variant: 'success' });
      setShowRoleModal(false);
    } catch (e: any) {
      Toast.onShow({
        msg: e?.msg || 'Error assigning role',
        variant: 'danger',
      });
    }
  };

  return (
    <>
      <h3 className="mb-4">Sites</h3>
      <p className="text-secondary mb-3">
        Manage the sites in your network. Each site is an independent Q&amp;A
        community with its own content, tags, and reputation.
      </p>
      <div className="mb-3">
        <Button variant="primary" size="sm" onClick={() => handleOpenSite()}>
          Add Site
        </Button>
      </div>
      <Table striped bordered hover size="sm">
        <thead>
          <tr>
            <th>Name</th>
            <th>Slug</th>
            <th>Description</th>
            <th>Status</th>
            <th>Actions</th>
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
                  {site.status === 1 ? 'Active' : 'Suspended'}
                </Badge>
              </td>
              <td>
                <Button
                  variant="outline-secondary"
                  size="sm"
                  className="me-1"
                  onClick={() => handleOpenSite(site)}>
                  Edit
                </Button>
                <Button
                  variant="outline-primary"
                  size="sm"
                  onClick={() => handleOpenRole(site)}>
                  Assign Role
                </Button>
              </td>
            </tr>
          ))}
          {sites.length === 0 && (
            <tr>
              <td colSpan={5} className="text-center text-muted">
                No sites found
              </td>
            </tr>
          )}
        </tbody>
      </Table>

      <Modal show={showSiteModal} onHide={() => setShowSiteModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>{editSite ? 'Edit Site' : 'Add Site'}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>Name</Form.Label>
            <Form.Control
              type="text"
              value={siteForm.name}
              onChange={(e) =>
                setSiteForm({ ...siteForm, name: e.target.value })
              }
              placeholder="Go Community"
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Slug</Form.Label>
            <Form.Control
              type="text"
              value={siteForm.slug}
              onChange={(e) =>
                setSiteForm({ ...siteForm, slug: e.target.value })
              }
              placeholder="golang"
              disabled={!!editSite}
            />
            <Form.Text className="text-muted">
              Used in URLs: /s/&#123;slug&#125;
            </Form.Text>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Description</Form.Label>
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
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSaveSite}>
            {editSite ? 'Save' : 'Create'}
          </Button>
        </Modal.Footer>
      </Modal>

      <Modal show={showRoleModal} onHide={() => setShowRoleModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Assign Role &mdash; {roleSite?.name}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label>User ID</Form.Label>
            <Form.Control
              type="text"
              value={roleForm.user_id}
              onChange={(e) =>
                setRoleForm({ ...roleForm, user_id: e.target.value })
              }
              placeholder="User ID"
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Role</Form.Label>
            <Form.Select
              value={roleForm.role_id}
              onChange={(e) =>
                setRoleForm({ ...roleForm, role_id: Number(e.target.value) })
              }>
              {ROLES.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.label}
                </option>
              ))}
            </Form.Select>
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowRoleModal(false)}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSaveRole}>
            Assign
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default Sites;
