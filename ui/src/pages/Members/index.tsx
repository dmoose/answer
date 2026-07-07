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

import { FC, useMemo, useState, useEffect } from 'react';
import {
  Row,
  Col,
  Card,
  Form,
  InputGroup,
  Badge,
  Button,
} from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { Link, Navigate, useSearchParams } from 'react-router-dom';

import { Avatar, Pagination, Empty } from '@/components';
import {
  useDirectorySearch,
  useProfileTags,
  type DirectorySearchParams,
  type ProfileTag,
} from '@/services';
import { featuresControlStore } from '@/stores';

const SORT_OPTIONS: Array<{
  value: NonNullable<DirectorySearchParams['sort']>;
  labelKey: string;
}> = [
  { value: 'rep_desc', labelKey: 'sort_reputation' },
  { value: 'newest', labelKey: 'sort_newest' },
  { value: 'active', labelKey: 'sort_active' },
];

const PAGE_SIZE = 20;

// readArray returns the value of `key` as a string[] from URL params.
// React Router's useSearchParams returns one value per key call; the listing
// page joins tag IDs as a single comma-separated query value to keep the URL
// short, then expands here.
function readArrayParam(params: URLSearchParams, key: string): string[] {
  const raw = params.get(key);
  if (!raw) return [];
  return raw.split(',').filter(Boolean);
}

const Members: FC = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'members' });
  const directoryEnabled = featuresControlStore((s) => s.directory_enabled);
  const [searchParams, setSearchParams] = useSearchParams();

  const queryStr = searchParams.get('q') ?? '';
  const sort =
    (searchParams.get('sort') as DirectorySearchParams['sort']) ?? 'rep_desc';
  const page = Number(searchParams.get('page') ?? '1');
  const tagIds = useMemo(
    () => readArrayParam(searchParams, 'tags'),
    [searchParams],
  );
  const openToMentoring = searchParams.get('mentoring') === '1';
  const openToCollaboration = searchParams.get('collab') === '1';
  const openToHire = searchParams.get('hire') === '1';

  const [qInput, setQInput] = useState(queryStr);
  useEffect(() => setQInput(queryStr), [queryStr]);

  const { data: catalog } = useProfileTags();
  const tagById = useMemo(() => {
    const m: Record<string, ProfileTag> = {};
    catalog?.forEach((tag) => {
      m[tag.id] = tag;
    });
    return m;
  }, [catalog]);

  const { data, isLoading } = useDirectorySearch({
    q: queryStr,
    tag_ids: tagIds,
    open_to_mentoring: openToMentoring,
    open_to_collaboration: openToCollaboration,
    open_to_hire: openToHire,
    sort,
    page,
    page_size: PAGE_SIZE,
  });

  function patch(updates: Record<string, string | null>) {
    const next = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([k, v]) => {
      if (v === null || v === '') next.delete(k);
      else next.set(k, v);
    });
    // any filter change resets the page
    if (!('page' in updates)) next.delete('page');
    setSearchParams(next, { replace: false });
  }

  function toggleTag(id: string) {
    const set = new Set(tagIds);
    if (set.has(id)) set.delete(id);
    else set.add(id);
    patch({ tags: set.size ? Array.from(set).join(',') : null });
  }

  function clearAll() {
    setSearchParams({}, { replace: false });
    setQInput('');
  }

  const submitSearch = (e: React.FormEvent) => {
    e.preventDefault();
    patch({ q: qInput.trim() || null });
  };

  const totalCount = data?.count ?? 0;
  const members = data?.list ?? [];

  if (!directoryEnabled) {
    return <Navigate to="/" replace />;
  }

  return (
    <Row className="py-4 mb-4">
      <Col xxl={9} lg={8}>
        <h3 className="mb-3">{t('page_title')}</h3>
        <p className="text-secondary mb-3">{t('page_desc')}</p>

        <Form onSubmit={submitSearch} className="mb-3">
          <InputGroup>
            <Form.Control
              type="search"
              placeholder={t('search_placeholder')}
              value={qInput}
              onChange={(e) => setQInput(e.target.value)}
            />
            <Button type="submit" variant="outline-secondary">
              {t('search_btn')}
            </Button>
          </InputGroup>
        </Form>

        <div className="mb-3 d-flex flex-wrap gap-2 align-items-center">
          <span className="text-secondary small">{t('sort_label')}</span>
          {SORT_OPTIONS.map((opt) => (
            <Button
              key={opt.value}
              size="sm"
              variant={sort === opt.value ? 'primary' : 'outline-secondary'}
              onClick={() => patch({ sort: opt.value })}>
              {t(opt.labelKey)}
            </Button>
          ))}
        </div>

        {isLoading ? (
          <div className="text-secondary">{t('loading')}</div>
        ) : members.length === 0 ? (
          <Empty>{t('empty')}</Empty>
        ) : (
          <Row>
            {members.map((m) => (
              <Col key={m.user_id} md={6} className="mb-3">
                <Card className="h-100">
                  <Card.Body>
                    <div className="d-flex">
                      <Link to={`/users/${m.username}`} className="me-3">
                        <Avatar
                          size="56px"
                          avatar={m.avatar}
                          searchStr="s=96"
                          alt={m.display_name}
                        />
                      </Link>
                      <div className="flex-grow-1 overflow-hidden">
                        <Link
                          to={`/users/${m.username}`}
                          className="text-break fw-semibold">
                          {m.display_name}
                        </Link>
                        <div className="text-secondary small">
                          {t('rep', { count: m.reputation })}
                          {m.pronouns ? ` · ${m.pronouns}` : ''}
                          {m.timezone ? ` · ${m.timezone}` : ''}
                        </div>
                        {m.headline ? (
                          <div className="mt-1 small">{m.headline}</div>
                        ) : null}
                        {m.tags.length > 0 ? (
                          <div className="mt-2 d-flex flex-wrap gap-1">
                            {m.tags.slice(0, 6).map((tag) => (
                              <Badge
                                key={tag.id}
                                bg="body-tertiary"
                                text="body"
                                className="border">
                                {tag.name}
                              </Badge>
                            ))}
                            {m.tags.length > 6 ? (
                              <span className="small text-secondary">
                                +{m.tags.length - 6}
                              </span>
                            ) : null}
                          </div>
                        ) : null}
                        {(m.open_to_mentoring ||
                          m.open_to_collaboration ||
                          m.open_to_hire) && (
                          <div className="mt-2 small d-flex flex-wrap gap-2">
                            {m.open_to_mentoring && (
                              <Badge bg="success">{t('badge_mentoring')}</Badge>
                            )}
                            {m.open_to_collaboration && (
                              <Badge bg="info">{t('badge_collab')}</Badge>
                            )}
                            {m.open_to_hire && (
                              <Badge bg="warning">{t('badge_hire')}</Badge>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  </Card.Body>
                </Card>
              </Col>
            ))}
          </Row>
        )}

        {totalCount > PAGE_SIZE ? (
          <div className="mt-3">
            <Pagination
              currentPage={page}
              totalSize={totalCount}
              pageSize={PAGE_SIZE}
              pathname="/members"
            />
          </div>
        ) : null}
      </Col>

      <Col xxl={3} lg={4}>
        <Card>
          <Card.Body>
            <div className="d-flex justify-content-between align-items-center mb-2">
              <h6 className="mb-0">{t('open_to')}</h6>
              {(openToMentoring || openToCollaboration || openToHire) && (
                <button
                  type="button"
                  className="btn btn-link btn-sm p-0"
                  onClick={() =>
                    patch({ mentoring: null, collab: null, hire: null })
                  }>
                  {t('clear')}
                </button>
              )}
            </div>
            <Form.Check
              type="checkbox"
              id="filter-mentoring"
              label={t('filter_mentoring')}
              checked={openToMentoring}
              onChange={(e) =>
                patch({ mentoring: e.target.checked ? '1' : null })
              }
            />
            <Form.Check
              type="checkbox"
              id="filter-collab"
              label={t('filter_collaboration')}
              checked={openToCollaboration}
              onChange={(e) => patch({ collab: e.target.checked ? '1' : null })}
            />
            <Form.Check
              type="checkbox"
              id="filter-hire"
              label={t('filter_hire')}
              checked={openToHire}
              onChange={(e) => patch({ hire: e.target.checked ? '1' : null })}
            />
          </Card.Body>
        </Card>

        <Card className="mt-3">
          <Card.Body>
            <div className="d-flex justify-content-between align-items-center mb-2">
              <h6 className="mb-0">{t('tags')}</h6>
              {tagIds.length > 0 && (
                <button
                  type="button"
                  className="btn btn-link btn-sm p-0"
                  onClick={() => patch({ tags: null })}>
                  {t('clear')}
                </button>
              )}
            </div>
            {!catalog || catalog.length === 0 ? (
              <div className="text-secondary small">{t('no_tags')}</div>
            ) : (
              <div className="d-flex flex-wrap gap-1">
                {catalog.map((tag) => {
                  const on = tagIds.includes(tag.id);
                  return (
                    <button
                      key={tag.id}
                      type="button"
                      className={`btn btn-sm ${
                        on ? 'btn-primary' : 'btn-outline-secondary'
                      }`}
                      onClick={() => toggleTag(tag.id)}>
                      {tag.name}
                    </button>
                  );
                })}
              </div>
            )}
          </Card.Body>
        </Card>

        {(tagIds.length > 0 ||
          queryStr ||
          openToMentoring ||
          openToCollaboration ||
          openToHire) && (
          <Button
            variant="outline-secondary"
            className="mt-3 w-100"
            onClick={() => clearAll()}>
            {t('clear_all')}
          </Button>
        )}

        {!isLoading && (
          <div className="text-secondary small mt-3 text-center">
            {t('matches', { count: totalCount })}
            {tagIds.length > 0 &&
              ` · ${t('tags_count', { count: tagIds.length })}`}
          </div>
        )}

        {tagIds.length > 0 && (
          <div className="mt-2 d-flex flex-wrap gap-1">
            {tagIds.map((id) => {
              const tag = tagById[id];
              if (!tag) return null;
              return (
                <Badge
                  key={id}
                  bg="primary"
                  className="d-flex align-items-center gap-1">
                  {tag.name}
                  <button
                    type="button"
                    aria-label={t('remove_tag', { name: tag.name })}
                    className="btn btn-sm btn-link p-0 text-white text-decoration-none ms-1"
                    onClick={() => toggleTag(id)}>
                    ×
                  </button>
                </Badge>
              );
            })}
          </div>
        )}
      </Col>
    </Row>
  );
};

export default Members;
