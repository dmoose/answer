import { FC, useEffect, useState } from 'react';
import { Card, Badge } from 'react-bootstrap';

import { getNetworkProfile, type NetworkProfile } from '@/services';
import { featuresControlStore } from '@/stores';

const STATUS_LABEL: Record<number, string> = {
  1: 'Active',
  2: 'Paused',
  9: 'Archived',
};

// hasContent returns true when the profile has at least one non-default field
// worth rendering. We skip the section entirely for users who have never
// filled in network/guild data so their profile page looks identical to
// upstream Answer.
function hasContent(p: NetworkProfile): boolean {
  return Boolean(
    p.headline ||
      p.pronouns ||
      p.timezone ||
      p.open_to_mentoring ||
      p.open_to_collaboration ||
      p.open_to_hire ||
      (p.external_links && p.external_links.length > 0) ||
      (p.tags && p.tags.length > 0) ||
      (p.projects && p.projects.length > 0),
  );
}

interface Props {
  userId?: string;
}

const NetworkProfileSection: FC<Props> = ({ userId }) => {
  const directoryEnabled = featuresControlStore((s) => s.directory_enabled);
  const [profile, setProfile] = useState<NetworkProfile | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (!userId || !directoryEnabled) {
      setProfile(null);
      return undefined;
    }
    getNetworkProfile(userId)
      .then((p) => {
        if (!cancelled) setProfile(p ?? null);
      })
      .catch(() => {
        if (!cancelled) setProfile(null);
      });
    return () => {
      cancelled = true;
    };
  }, [userId, directoryEnabled]);

  if (!directoryEnabled || !profile || !hasContent(profile)) return null;

  return (
    <>
      {(profile.headline ||
        profile.pronouns ||
        profile.timezone ||
        profile.open_to_mentoring ||
        profile.open_to_collaboration ||
        profile.open_to_hire) && (
        <section className="mb-4">
          {profile.headline && <h5 className="mb-2">{profile.headline}</h5>}
          {(profile.pronouns || profile.timezone) && (
            <div className="text-secondary small mb-2">
              {[profile.pronouns, profile.timezone].filter(Boolean).join(' · ')}
            </div>
          )}
          {(profile.open_to_mentoring ||
            profile.open_to_collaboration ||
            profile.open_to_hire) && (
            <div className="d-flex flex-wrap gap-1">
              {profile.open_to_mentoring && (
                <Badge bg="success">Open to mentoring</Badge>
              )}
              {profile.open_to_collaboration && (
                <Badge bg="info">Open to collaboration</Badge>
              )}
              {profile.open_to_hire && <Badge bg="warning">Open to hire</Badge>}
            </div>
          )}
        </section>
      )}

      {profile.tags && profile.tags.length > 0 && (
        <section className="mb-4">
          <h6 className="mb-2">Skills &amp; Interests</h6>
          <div className="d-flex flex-wrap gap-1">
            {profile.tags.map((t) => (
              <Badge
                key={t.id}
                bg="body-tertiary"
                text="body"
                className="border">
                {t.name}
              </Badge>
            ))}
          </div>
        </section>
      )}

      {profile.projects && profile.projects.length > 0 && (
        <section className="mb-4">
          <h6 className="mb-2">Projects</h6>
          {profile.projects.map((p) => (
            <Card key={p.id} className="mb-2">
              <Card.Body>
                <div className="d-flex justify-content-between align-items-start gap-2">
                  <div className="flex-grow-1">
                    <div className="d-flex flex-wrap gap-2 align-items-center mb-1">
                      <strong>{p.title}</strong>
                      <Badge
                        bg={p.status === 1 ? 'success' : 'secondary'}
                        className="text-uppercase small">
                        {STATUS_LABEL[p.status] ?? 'Unknown'}
                      </Badge>
                      {p.seeking_help && (
                        <Badge bg="warning">Seeking collaborators</Badge>
                      )}
                    </div>
                    {p.description && (
                      <div className="small text-secondary">
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
                </div>
              </Card.Body>
            </Card>
          ))}
        </section>
      )}

      {profile.external_links && profile.external_links.length > 0 && (
        <section className="mb-4">
          <h6 className="mb-2">Links</h6>
          <div className="d-flex flex-wrap gap-2">
            {profile.external_links.map((l) => (
              <a
                key={`${l.label}-${l.url}`}
                href={l.url}
                target="_blank"
                rel="noreferrer"
                className="text-decoration-none">
                {l.label || l.url}
              </a>
            ))}
          </div>
          <div className="text-secondary small mt-1">
            User-claimed — not verified.
          </div>
        </section>
      )}
    </>
  );
};

export default NetworkProfileSection;
