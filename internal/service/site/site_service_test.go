package site

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/entity"
)

type mockSiteRepo struct {
	sites map[string]*entity.Site
}

func newMockSiteRepo() *mockSiteRepo {
	return &mockSiteRepo{sites: make(map[string]*entity.Site)}
}

func (m *mockSiteRepo) AddSite(_ context.Context, s *entity.Site) error {
	m.sites[s.ID] = s
	return nil
}

func (m *mockSiteRepo) UpdateSite(_ context.Context, s *entity.Site) error {
	m.sites[s.ID] = s
	return nil
}

func (m *mockSiteRepo) GetSite(_ context.Context, id string) (*entity.Site, bool, error) {
	s, ok := m.sites[id]
	return s, ok, nil
}

func (m *mockSiteRepo) GetSiteBySlug(_ context.Context, slug string) (*entity.Site, bool, error) {
	for _, s := range m.sites {
		if s.Slug == slug {
			return s, true, nil
		}
	}
	return nil, false, nil
}

func (m *mockSiteRepo) GetAllSites(_ context.Context) ([]*entity.Site, error) {
	var sites []*entity.Site
	for _, s := range m.sites {
		if s.Status == entity.SiteStatusActive {
			sites = append(sites, s)
		}
	}
	return sites, nil
}

func TestSiteService_AddSite(t *testing.T) {
	repo := newMockSiteRepo()
	svc := NewSiteService(repo)

	s, err := svc.AddSite(context.Background(), "Go Community", "golang", "Go Q&A", "https://go.example.com")
	if err != nil {
		t.Fatalf("AddSite: %v", err)
	}
	if s.Slug != "golang" {
		t.Errorf("slug = %q, want %q", s.Slug, "golang")
	}
	if s.ID == "" {
		t.Error("ID should be generated")
	}

	// duplicate slug
	_, err = svc.AddSite(context.Background(), "Go Again", "golang", "", "")
	if err == nil {
		t.Error("expected error for duplicate slug")
	}
}

func TestSiteService_GetAllSites(t *testing.T) {
	repo := newMockSiteRepo()
	svc := NewSiteService(repo)

	_, _ = svc.AddSite(context.Background(), "Site A", "a", "", "")
	_, _ = svc.AddSite(context.Background(), "Site B", "b", "", "")

	sites, err := svc.GetAllSites(context.Background())
	if err != nil {
		t.Fatalf("GetAllSites: %v", err)
	}
	if len(sites) != 2 {
		t.Errorf("got %d sites, want 2", len(sites))
	}
}
