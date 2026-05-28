package entity

import "time"

type UserSiteRank struct {
	ID        string    `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt time.Time `xorm:"updated TIMESTAMP updated_at"`
	UserID    string    `xorm:"not null BIGINT(20) UNIQUE(ux_user_site) user_id"`
	SiteID    string    `xorm:"not null VARCHAR(36) UNIQUE(ux_user_site) INDEX site_id"`
	Rank      int       `xorm:"not null default 1 INT(11) rank"`
}

func (UserSiteRank) TableName() string {
	return "user_site_rank"
}
