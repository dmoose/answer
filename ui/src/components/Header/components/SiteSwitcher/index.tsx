import { FC } from 'react';
import { Dropdown } from 'react-bootstrap';

import currentSiteStore from '@/stores/currentSite';

const SiteSwitcher: FC = () => {
  const { currentSite, sites } = currentSiteStore();

  if (sites.length <= 1) {
    return null;
  }

  const handleSelect = (slug: string) => {
    if (slug === 'default') {
      window.location.href = '/';
    } else {
      window.location.href = `/s/${slug}`;
    }
  };

  return (
    <Dropdown className="ms-auto me-3">
      <Dropdown.Toggle
        variant="link"
        className="nav-link text-capitalize text-nowrap p-0"
        id="site-switcher">
        {currentSite?.name || 'Select Site'}
      </Dropdown.Toggle>
      <Dropdown.Menu align="end">
        {sites.map((site) => (
          <Dropdown.Item
            key={site.id}
            active={site.id === currentSite?.id}
            onClick={() => handleSelect(site.slug)}>
            {site.name}
            {site.description ? (
              <small className="d-block text-muted">{site.description}</small>
            ) : null}
          </Dropdown.Item>
        ))}
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default SiteSwitcher;
