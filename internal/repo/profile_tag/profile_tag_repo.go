package profile_tag

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/errors"
	"xorm.io/xorm"
)

type ProfileTagRepo struct {
	data *data.Data
}

func NewProfileTagRepo(data *data.Data) *ProfileTagRepo {
	return &ProfileTagRepo{data: data}
}

// ListActive returns all active profile tags, optionally filtered by kind
// (0 means any kind).
func (r *ProfileTagRepo) ListActive(ctx context.Context, kind int) ([]*entity.ProfileTag, error) {
	var tags []*entity.ProfileTag
	session := r.data.DB.Context(ctx).Where("status = ?", entity.ProfileTagStatusActive)
	if kind > 0 {
		session = session.And("kind = ? OR kind = ?", kind, entity.ProfileTagKindBoth)
	}
	err := session.OrderBy("name ASC").Find(&tags)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return tags, nil
}

// ListAll returns every profile tag including inactive — admin-only because
// the public picker shouldn't surface retired tags.
func (r *ProfileTagRepo) ListAll(ctx context.Context) ([]*entity.ProfileTag, error) {
	var tags []*entity.ProfileTag
	err := r.data.DB.Context(ctx).OrderBy("status ASC, name ASC").Find(&tags)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return tags, nil
}

func (r *ProfileTagRepo) GetBySlug(ctx context.Context, slug string) (*entity.ProfileTag, bool, error) {
	t := &entity.ProfileTag{}
	exist, err := r.data.DB.Context(ctx).Where("slug = ?", slug).Get(t)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return t, exist, nil
}

func (r *ProfileTagRepo) GetByIDs(ctx context.Context, ids []string) ([]*entity.ProfileTag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var tags []*entity.ProfileTag
	err := r.data.DB.Context(ctx).In("id", ids).Find(&tags)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return tags, nil
}

func (r *ProfileTagRepo) Insert(ctx context.Context, t *entity.ProfileTag) error {
	_, err := r.data.DB.Context(ctx).Insert(t)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *ProfileTagRepo) Update(ctx context.Context, t *entity.ProfileTag) error {
	_, err := r.data.DB.Context(ctx).ID(t.ID).
		Cols("name", "kind", "description", "status").Update(t)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

// SetUserTags replaces the user's tag set in one shot — delete all existing,
// insert the new set. Single transaction so the user is never in a partial
// state from a UI mid-save.
func (r *ProfileTagRepo) SetUserTags(ctx context.Context, userID string, tagIDs []string) error {
	_, err := r.data.DB.Transaction(func(session *xorm.Session) (any, error) {
		session = session.Context(ctx)
		if _, err := session.Where("user_id = ?", userID).Delete(&entity.UserProfileTag{}); err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
		if len(tagIDs) == 0 {
			return nil, nil
		}
		rows := make([]*entity.UserProfileTag, 0, len(tagIDs))
		for _, tid := range tagIDs {
			rows = append(rows, &entity.UserProfileTag{UserID: userID, TagID: tid})
		}
		if _, err := session.Insert(rows); err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
		return nil, nil
	})
	return err
}

// GetUserTags returns the tag IDs for a single user.
func (r *ProfileTagRepo) GetUserTags(ctx context.Context, userID string) ([]string, error) {
	var rows []*entity.UserProfileTag
	err := r.data.DB.Context(ctx).Where("user_id = ?", userID).Find(&rows)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.TagID)
	}
	return ids, nil
}

// GetUsersByTag returns the user IDs of members tagged with tagID. Reverse
// lookup for "who knows X" directory queries.
func (r *ProfileTagRepo) GetUsersByTag(ctx context.Context, tagID string) ([]string, error) {
	var rows []*entity.UserProfileTag
	err := r.data.DB.Context(ctx).Where("tag_id = ?", tagID).Find(&rows)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}
	return ids, nil
}

// GetTagsForUsers returns a map of user_id → []tag_id for a batch of users.
// Used by the directory to populate member cards in one round-trip.
func (r *ProfileTagRepo) GetTagsForUsers(ctx context.Context, userIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []*entity.UserProfileTag
	err := r.data.DB.Context(ctx).In("user_id", userIDs).Find(&rows)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], r.TagID)
	}
	return out, nil
}
