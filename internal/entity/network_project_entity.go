package entity

import "time"

const (
	NetworkProjectStatusActive   = 1
	NetworkProjectStatusPaused   = 2
	NetworkProjectStatusArchived = 9
)

// NetworkProject is something a member is working on. Surfaces on the
// member profile and on the directory home as a "recent projects" feed.
type NetworkProject struct {
	ID           string    `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt    time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt    time.Time `xorm:"updated TIMESTAMP updated_at"`
	UserID       string    `xorm:"not null INDEX BIGINT(20) user_id"`
	Title        string    `xorm:"not null VARCHAR(200) title"`
	Description  string    `xorm:"TEXT description"`
	RepoURL      string    `xorm:"VARCHAR(512) repo_url"`
	Status       int       `xorm:"not null default 1 INT(11) status"`
	SeekingHelp  bool      `xorm:"not null default false BOOL seeking_help"`
}

func (NetworkProject) TableName() string {
	return "network_project"
}
