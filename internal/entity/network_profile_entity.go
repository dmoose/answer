package entity

import "time"

// NetworkProfile is the guild-level identity record for a user, separate from
// the per-site Answer profile. One row per user; absence means the user has
// not filled in any guild fields yet.
//
// ExternalLinks is a JSON array of {label, url} objects, self-attested and
// presentation-only. Verified cross-app identity (Zulip, GitHub, etc.) is
// resolved through fastgate's directory, not from this field.
type NetworkProfile struct {
	UserID                string    `xorm:"not null pk BIGINT(20) user_id"`
	CreatedAt             time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt             time.Time `xorm:"updated TIMESTAMP updated_at"`
	Headline              string    `xorm:"VARCHAR(255) headline"`
	Pronouns              string    `xorm:"VARCHAR(64) pronouns"`
	Timezone              string    `xorm:"VARCHAR(64) timezone"`
	OpenToMentoring       bool      `xorm:"not null default false BOOL open_to_mentoring"`
	OpenToCollaboration   bool      `xorm:"not null default false BOOL open_to_collaboration"`
	OpenToHire            bool      `xorm:"not null default false BOOL open_to_hire"`
	ExternalLinks         string    `xorm:"TEXT external_links"`
}

func (NetworkProfile) TableName() string {
	return "network_profile"
}
