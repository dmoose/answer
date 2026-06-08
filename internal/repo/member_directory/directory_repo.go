package member_directory

import (
	"context"
	"strings"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/pager"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/errors"
	"xorm.io/builder"
)

type MemberDirectoryRepo struct {
	data *data.Data
}

func NewMemberDirectoryRepo(data *data.Data) *MemberDirectoryRepo {
	return &MemberDirectoryRepo{data: data}
}

// DirectoryQuery is the resolved server-side query: filters + paging + sort.
// Empty fields are treated as "any".
type DirectoryQuery struct {
	Q                   string
	TagIDs              []string
	OpenToMentoring     bool
	OpenToCollaboration bool
	OpenToHire          bool
	Page                int
	PageSize            int
	Sort                string // rep_desc | newest | active
}

// DirectoryRow flattens the join columns into a single struct. The repo emits
// these; the service layer assembles tags and shapes the API response.
type DirectoryRow struct {
	UserID              string `xorm:"'user_id'"`
	Username            string `xorm:"'username'"`
	DisplayName         string `xorm:"'display_name'"`
	Avatar              string `xorm:"'avatar'"`
	Rank                int    `xorm:"'rank'"`
	Headline            string `xorm:"'headline'"`
	Pronouns            string `xorm:"'pronouns'"`
	Timezone            string `xorm:"'timezone'"`
	OpenToMentoring     bool   `xorm:"'open_to_mentoring'"`
	OpenToCollaboration bool   `xorm:"'open_to_collaboration'"`
	OpenToHire          bool   `xorm:"'open_to_hire'"`
}

// Search runs the faceted query and returns the page plus total match count.
// The total is the same query without LIMIT/OFFSET — useful for pagination UI
// but cheap because the filter set is small.
func (r *MemberDirectoryRepo) Search(ctx context.Context, q *DirectoryQuery) ([]*DirectoryRow, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}

	cond := builder.NewCond().And(builder.Eq{"`user`.status": entity.UserStatusAvailable})

	// Availability flags filter on network_profile columns; LEFT JOIN means
	// these are also NULL for members without a profile row. We treat
	// "filter on" as "must be present and true," so the filter implicitly
	// requires a profile row.
	if q.OpenToMentoring {
		cond = cond.And(builder.Eq{"np.open_to_mentoring": true})
	}
	if q.OpenToCollaboration {
		cond = cond.And(builder.Eq{"np.open_to_collaboration": true})
	}
	if q.OpenToHire {
		cond = cond.And(builder.Eq{"np.open_to_hire": true})
	}

	if s := strings.TrimSpace(q.Q); s != "" {
		like := "%" + s + "%"
		cond = cond.And(builder.Or(
			builder.Like{"`user`.username", like},
			builder.Like{"`user`.display_name", like},
			builder.Like{"np.headline", like},
		))
	}

	if len(q.TagIDs) > 0 {
		// EXISTS subquery: any matching tag wins. AND-semantics across many
		// tags is rarely what a directory user wants and is more expensive.
		args := make([]any, 0, len(q.TagIDs))
		for _, t := range q.TagIDs {
			args = append(args, t)
		}
		cond = cond.And(builder.Expr(
			"EXISTS (SELECT 1 FROM `user_profile_tag` ugt WHERE ugt.user_id = `user`.id AND ugt.tag_id IN ("+placeholders(len(q.TagIDs))+"))",
			args...,
		))
	}

	session := r.data.DB.Context(ctx).
		Table("user").
		Select("`user`.id AS user_id, `user`.username, `user`.display_name, `user`.avatar, `user`.rank, "+
			"COALESCE(np.headline,'') AS headline, COALESCE(np.pronouns,'') AS pronouns, "+
			"COALESCE(np.timezone,'') AS timezone, "+
			"COALESCE(np.open_to_mentoring,FALSE) AS open_to_mentoring, "+
			"COALESCE(np.open_to_collaboration,FALSE) AS open_to_collaboration, "+
			"COALESCE(np.open_to_hire,FALSE) AS open_to_hire").
		Join("LEFT", []string{"network_profile", "np"}, "np.user_id = `user`.id").
		Where(cond)

	switch q.Sort {
	case "newest":
		session = session.OrderBy("`user`.created_at DESC")
	case "active":
		session = session.OrderBy("`user`.last_login_date DESC")
	default:
		session = session.OrderBy("`user`.rank DESC, `user`.id DESC")
	}

	var rows []*DirectoryRow
	total, err := pager.Help(q.Page, q.PageSize, &rows, &DirectoryRow{}, session)
	if err != nil {
		return nil, 0, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return rows, total, nil
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
	}
	return b.String()
}
