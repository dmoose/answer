package migrations

import (
	"context"
	"fmt"

	"github.com/apache/answer/internal/entity"
	"xorm.io/xorm"
)

// addNetworkDirectory creates the network-level member directory tables:
// extended profile, projects-in-progress, and curated skill/interest tags.
// All tables key by user_id (BIGINT) and are network-level — no site_id —
// matching the network profile concept already in place.
func addNetworkDirectory(ctx context.Context, x *xorm.Engine) error {
	if err := x.Context(ctx).Sync(
		new(entity.NetworkProfile),
		new(entity.NetworkProject),
		new(entity.ProfileTag),
		new(entity.UserProfileTag),
	); err != nil {
		return fmt.Errorf("create network directory tables: %w", err)
	}
	return nil
}
