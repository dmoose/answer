package entity

import "time"

// UserProfileTag joins members to their selected profile tags. Reverse-indexed
// by tag_id so the directory can answer "who has skill X" cheaply.
type UserProfileTag struct {
	ID        string    `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt time.Time `xorm:"created TIMESTAMP created_at"`
	UserID    string    `xorm:"not null BIGINT(20) UNIQUE(ux_user_profile_tag) user_id"`
	TagID     string    `xorm:"not null BIGINT(20) UNIQUE(ux_user_profile_tag) INDEX tag_id"`
}

func (UserProfileTag) TableName() string {
	return "user_profile_tag"
}
