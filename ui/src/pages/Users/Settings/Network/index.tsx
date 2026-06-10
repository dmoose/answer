import { FC, useEffect, useState } from 'react';
import { Form, Button, Badge, Alert } from 'react-bootstrap';

import { useToast } from '@/hooks';
import {
  getNetworkProfile,
  updateNetworkProfile,
  setMyTags,
  useProfileTags,
  type ProfileExternalLink,
} from '@/services';
import { loggedUserInfoStore } from '@/stores';

const MAX_LINKS = 10;

interface LinkRow extends ProfileExternalLink {
  key: string;
}

let linkKeySeq = 0;
const makeLinkKey = () => {
  linkKeySeq += 1;
  return `link-${linkKeySeq}`;
};

const NetworkSettings: FC = () => {
  const Toast = useToast();
  const user = loggedUserInfoStore((s) => s.user);
  const { data: catalog } = useProfileTags();

  const [headline, setHeadline] = useState('');
  const [pronouns, setPronouns] = useState('');
  const [timezone, setTimezone] = useState('');
  const [openMentoring, setOpenMentoring] = useState(false);
  const [openCollab, setOpenCollab] = useState(false);
  const [openHire, setOpenHire] = useState(false);
  const [links, setLinks] = useState<LinkRow[]>([]);
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!user?.id) return;
    setLoading(true);
    getNetworkProfile(user.id)
      .then((p) => {
        if (!p) return;
        setHeadline(p.headline || '');
        setPronouns(p.pronouns || '');
        setTimezone(p.timezone || '');
        setOpenMentoring(Boolean(p.open_to_mentoring));
        setOpenCollab(Boolean(p.open_to_collaboration));
        setOpenHire(Boolean(p.open_to_hire));
        setLinks(
          (p.external_links || []).map((l) => ({ ...l, key: makeLinkKey() })),
        );
        setSelectedTagIds((p.tags || []).map((t) => t.id));
      })
      .catch(() => {
        // first-time users have no row; defaults are fine
      })
      .finally(() => setLoading(false));
  }, [user?.id]);

  function updateLink(idx: number, field: 'label' | 'url', value: string) {
    const next = [...links];
    next[idx] = { ...next[idx], [field]: value };
    setLinks(next);
  }

  function addLink() {
    if (links.length >= MAX_LINKS) return;
    setLinks([...links, { label: '', url: '', key: makeLinkKey() }]);
  }

  function removeLink(idx: number) {
    setLinks(links.filter((_, i) => i !== idx));
  }

  function toggleTag(id: string) {
    setSelectedTagIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  }

  async function save() {
    setSaving(true);
    try {
      const cleanLinks = links
        .map((l) => ({ label: l.label.trim(), url: l.url.trim() }))
        .filter((l) => l.label && l.url);

      await updateNetworkProfile({
        headline: headline.trim(),
        pronouns: pronouns.trim(),
        timezone: timezone.trim(),
        open_to_mentoring: openMentoring,
        open_to_collaboration: openCollab,
        open_to_hire: openHire,
        external_links: cleanLinks,
      });
      await setMyTags(selectedTagIds);
      Toast.onShow({ msg: 'Network profile saved', variant: 'success' });
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

  if (loading) {
    return <div className="text-secondary">Loading…</div>;
  }

  return (
    <>
      <h3 className="mb-4">Network profile</h3>
      <p className="text-secondary mb-4">
        These fields show up on the network member directory and on your profile
        page. Members search and filter by these — leave them blank if
        you&apos;d rather not appear in those facets.
      </p>

      <Form
        onSubmit={(e) => {
          e.preventDefault();
          save();
        }}>
        <Form.Group className="mb-3">
          <Form.Label>Headline</Form.Label>
          <Form.Control
            type="text"
            value={headline}
            maxLength={255}
            onChange={(e) => setHeadline(e.target.value)}
            placeholder="Short one-liner about what you do or what you're into"
          />
        </Form.Group>

        <div className="d-flex gap-3 mb-3">
          <Form.Group className="flex-grow-1">
            <Form.Label>Pronouns</Form.Label>
            <Form.Control
              type="text"
              value={pronouns}
              maxLength={64}
              onChange={(e) => setPronouns(e.target.value)}
              placeholder="she/her, he/him, they/them, …"
            />
          </Form.Group>
          <Form.Group className="flex-grow-1">
            <Form.Label>Timezone</Form.Label>
            <Form.Control
              type="text"
              value={timezone}
              maxLength={64}
              onChange={(e) => setTimezone(e.target.value)}
              placeholder="America/Los_Angeles"
            />
          </Form.Group>
        </div>

        <Form.Group className="mb-4">
          <Form.Label>Open to</Form.Label>
          <Form.Check
            type="checkbox"
            id="open-mentoring"
            label="Mentoring"
            checked={openMentoring}
            onChange={(e) => setOpenMentoring(e.target.checked)}
          />
          <Form.Check
            type="checkbox"
            id="open-collab"
            label="Collaboration"
            checked={openCollab}
            onChange={(e) => setOpenCollab(e.target.checked)}
          />
          <Form.Check
            type="checkbox"
            id="open-hire"
            label="Hire / for-hire"
            checked={openHire}
            onChange={(e) => setOpenHire(e.target.checked)}
          />
        </Form.Group>

        <Form.Group className="mb-4">
          <Form.Label>Skills &amp; Interests</Form.Label>
          {!catalog || catalog.length === 0 ? (
            <div className="text-secondary small">
              No tags defined yet. An admin can curate skill / interest tags
              under the network admin tools.
            </div>
          ) : (
            <div className="d-flex flex-wrap gap-1">
              {catalog.map((t) => {
                const on = selectedTagIds.includes(t.id);
                return (
                  <button
                    key={t.id}
                    type="button"
                    className={`btn btn-sm ${
                      on ? 'btn-primary' : 'btn-outline-secondary'
                    }`}
                    onClick={() => toggleTag(t.id)}>
                    {t.name}
                  </button>
                );
              })}
            </div>
          )}
          {selectedTagIds.length > 0 && (
            <div className="text-secondary small mt-2">
              {selectedTagIds.length} selected
            </div>
          )}
        </Form.Group>

        <Form.Group className="mb-4">
          <Form.Label className="d-flex justify-content-between align-items-center">
            <span>External links</span>
            <Button
              variant="outline-secondary"
              size="sm"
              onClick={() => addLink()}
              disabled={links.length >= MAX_LINKS}>
              Add link
            </Button>
          </Form.Label>
          <Alert variant="light" className="border small">
            <Badge bg="secondary" className="me-2">
              user-claimed
            </Badge>
            These show on your profile as labelled links. Not verified by the
            network. For routing notifications (Zulip, Discord, etc.) the
            network resolves your identity through the SSO directory, not from
            this list.
          </Alert>
          {links.length === 0 ? (
            <div className="text-secondary small">No links yet.</div>
          ) : (
            links.map((l, i) => (
              <div key={l.key} className="d-flex gap-2 mb-2">
                <Form.Control
                  type="text"
                  placeholder="Label (e.g. GitHub)"
                  value={l.label}
                  maxLength={64}
                  onChange={(e) => updateLink(i, 'label', e.target.value)}
                />
                <Form.Control
                  type="url"
                  placeholder="https://…"
                  value={l.url}
                  maxLength={512}
                  onChange={(e) => updateLink(i, 'url', e.target.value)}
                />
                <Button
                  variant="outline-danger"
                  onClick={() => removeLink(i)}
                  aria-label="Remove link">
                  ×
                </Button>
              </div>
            ))
          )}
        </Form.Group>

        <Button type="submit" variant="primary" disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </Button>
      </Form>
    </>
  );
};

export default NetworkSettings;
