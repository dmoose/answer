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

import { FC, FormEvent, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Button, Card } from 'react-bootstrap';

import { useToast } from '@/hooks';
import { getAppSwitcher, saveAppSwitcher, AppSwitcherLink } from '@/services';
import { Icon } from '@/components';

// React needs a stable key while we reorder/add/remove rows; the link
// payload itself doesn't have one, so we attach a client-side id that gets
// stripped before saving.
type EditLink = AppSwitcherLink & { clientId: string };

let nextClientId = 1;
const newClientId = () => {
  const id = nextClientId;
  nextClientId += 1;
  return `link-${id}`;
};
const withClientId = (l: AppSwitcherLink): EditLink => ({
  ...l,
  clientId: newClientId(),
});
const emptyLink = (): EditLink =>
  withClientId({ name: '', description: '', url: '', icon: '' });

const AppSwitcherPage: FC = () => {
  const toast = useToast();
  const { t } = useTranslation('translation', {
    keyPrefix: 'admin.app_switcher',
  });
  const [enabled, setEnabled] = useState(false);
  const [links, setLinks] = useState<EditLink[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getAppSwitcher()
      .then((resp) => {
        setEnabled(!!resp.enabled);
        setLinks(resp.links?.length ? resp.links.map(withClientId) : []);
      })
      .catch(() => {
        // No row yet; start blank.
      })
      .finally(() => setLoading(false));
  }, []);

  const updateLink = (idx: number, patch: Partial<EditLink>) => {
    setLinks((prev) =>
      prev.map((l, i) => (i === idx ? { ...l, ...patch } : l)),
    );
  };
  const removeLink = (idx: number) => {
    setLinks((prev) => prev.filter((_, i) => i !== idx));
  };
  const addLink = () => {
    setLinks((prev) => [...prev, emptyLink()]);
  };
  const moveLink = (idx: number, dir: -1 | 1) => {
    setLinks((prev) => {
      const next = [...prev];
      const target = idx + dir;
      if (target < 0 || target >= next.length) return prev;
      [next[idx], next[target]] = [next[target], next[idx]];
      return next;
    });
  };

  const onSubmit = (evt: FormEvent) => {
    evt.preventDefault();
    const cleaned = links
      .map((l) => ({
        name: l.name.trim(),
        description: l.description?.trim() ?? '',
        url: l.url.trim(),
        icon: l.icon?.trim() ?? '',
      }))
      .filter((l) => l.name && l.url);
    saveAppSwitcher({ enabled, links: cleaned }).then(() => {
      setLinks(cleaned.map(withClientId));
      toast.onShow({
        msg: t('update', { keyPrefix: 'toast' }),
        variant: 'success',
      });
    });
  };

  if (loading) return null;

  return (
    <>
      <h3 className="mb-4">{t('app_switcher', { keyPrefix: 'nav_menus' })}</h3>
      <p className="text-secondary">{t('description')}</p>
      <Form onSubmit={onSubmit} className="max-w-748">
        <Form.Group className="mb-4">
          <Form.Check
            type="switch"
            id="app-switcher-enabled"
            label={t('enabled_label')}
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
          />
          <Form.Text className="text-muted">{t('enabled_text')}</Form.Text>
        </Form.Group>

        {links.map((link, idx) => (
          <Card key={link.clientId} className="mb-3">
            <Card.Body>
              <div className="d-flex justify-content-between align-items-center mb-3">
                <strong>
                  {t('app_title')} {idx + 1}
                </strong>
                <div className="d-flex gap-1">
                  <Button
                    size="sm"
                    variant="outline-secondary"
                    onClick={() => moveLink(idx, -1)}
                    disabled={idx === 0}
                    title={t('app_move_up')}>
                    <Icon name="arrow-up" />
                  </Button>
                  <Button
                    size="sm"
                    variant="outline-secondary"
                    onClick={() => moveLink(idx, 1)}
                    disabled={idx === links.length - 1}
                    title={t('app_move_down')}>
                    <Icon name="arrow-down" />
                  </Button>
                  <Button
                    size="sm"
                    variant="outline-danger"
                    onClick={() => removeLink(idx)}
                    title={t('app_remove')}>
                    <Icon name="trash" />
                  </Button>
                </div>
              </div>
              <Form.Group className="mb-3">
                <Form.Label>{t('app_name_label')}</Form.Label>
                <Form.Control
                  required
                  value={link.name}
                  maxLength={64}
                  onChange={(e) => updateLink(idx, { name: e.target.value })}
                />
              </Form.Group>
              <Form.Group className="mb-3">
                <Form.Label>{t('app_url_label')}</Form.Label>
                <Form.Control
                  required
                  type="url"
                  placeholder="https://"
                  value={link.url}
                  onChange={(e) => updateLink(idx, { url: e.target.value })}
                />
              </Form.Group>
              <Form.Group className="mb-3">
                <Form.Label>{t('app_description_label')}</Form.Label>
                <Form.Control
                  value={link.description ?? ''}
                  maxLength={200}
                  onChange={(e) =>
                    updateLink(idx, { description: e.target.value })
                  }
                />
                <Form.Text className="text-muted">
                  {t('app_description_text')}
                </Form.Text>
              </Form.Group>
              <Form.Group>
                <Form.Label>{t('app_icon_label')}</Form.Label>
                <Form.Control
                  type="url"
                  placeholder="https://"
                  value={link.icon ?? ''}
                  onChange={(e) => updateLink(idx, { icon: e.target.value })}
                />
                <Form.Text className="text-muted">
                  {t('app_icon_text')}
                </Form.Text>
              </Form.Group>
            </Card.Body>
          </Card>
        ))}

        <Button
          variant="outline-primary"
          type="button"
          className="mb-4"
          onClick={addLink}>
          <Icon name="plus-lg" className="me-1" />
          {t('add_link')}
        </Button>

        <div>
          <Button variant="primary" type="submit">
            {t('save', { keyPrefix: 'btns' })}
          </Button>
        </div>
      </Form>
    </>
  );
};

export default AppSwitcherPage;
