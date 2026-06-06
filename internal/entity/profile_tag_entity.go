package entity

import "time"

const (
	ProfileTagKindSkill    = 1
	ProfileTagKindInterest = 2
	ProfileTagKindBoth     = 3

	ProfileTagStatusActive   = 1
	ProfileTagStatusInactive = 9
)

// ProfileTag is an admin-curated tag attached to members for the directory
// faceting (skill: "Rust", interest: "homelab", etc.). Separate from Answer's
// Q&A tag system; the lifecycles and meaning are different.
type ProfileTag struct {
	ID          string    `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt   time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt   time.Time `xorm:"updated TIMESTAMP updated_at"`
	Slug        string    `xorm:"not null unique VARCHAR(64) slug"`
	Name        string    `xorm:"not null VARCHAR(128) name"`
	Kind        int       `xorm:"not null default 1 INT(11) kind"`
	Description string    `xorm:"VARCHAR(512) description"`
	Status      int       `xorm:"not null default 1 INT(11) status"`
}

func (ProfileTag) TableName() string {
	return "profile_tag"
}
