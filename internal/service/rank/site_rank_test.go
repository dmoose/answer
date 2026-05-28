//go:build multisite

package rank

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
)

type mockSiteRankRepo struct {
	ranks map[string]map[string]int // userID -> siteID -> rank
}

func (m *mockSiteRankRepo) GetUserSiteRank(_ context.Context, userID, siteID string) (int, error) {
	if siteRanks, ok := m.ranks[userID]; ok {
		if r, ok := siteRanks[siteID]; ok {
			return r, nil
		}
	}
	return 1, nil
}

func (m *mockSiteRankRepo) GetUserAllSiteRanks(_ context.Context, userID string) ([]entity.UserSiteRank, error) {
	var ranks []entity.UserSiteRank
	for siteID, rank := range m.ranks[userID] {
		ranks = append(ranks, entity.UserSiteRank{UserID: userID, SiteID: siteID, Rank: rank})
	}
	return ranks, nil
}

func TestGetUserRankForPermission_WithSiteContext(t *testing.T) {
	repo := &mockSiteRankRepo{
		ranks: map[string]map[string]int{
			"user-1": {"site-a": 500, "site-b": 10},
		},
	}
	rs := &RankService{siteRankRepo: repo}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")
	got := rs.getUserRankForPermission(ctx, "user-1", 1000)
	if got != 500 {
		t.Errorf("got %d, want 500 (site-a rank)", got)
	}

	ctx = context.WithValue(context.Background(), constant.SiteIDContextKey, "site-b")
	got = rs.getUserRankForPermission(ctx, "user-1", 1000)
	if got != 10 {
		t.Errorf("got %d, want 10 (site-b rank)", got)
	}
}

func TestGetUserRankForPermission_NoSiteContext(t *testing.T) {
	repo := &mockSiteRankRepo{}
	rs := &RankService{siteRankRepo: repo}

	got := rs.getUserRankForPermission(context.Background(), "user-1", 42)
	if got != 42 {
		t.Errorf("got %d, want 42 (global rank fallback)", got)
	}
}

func TestGetUserRankForPermission_NilRepo(t *testing.T) {
	rs := &RankService{siteRankRepo: nil}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")
	got := rs.getUserRankForPermission(ctx, "user-1", 42)
	if got != 42 {
		t.Errorf("got %d, want 42 (nil repo fallback)", got)
	}
}
