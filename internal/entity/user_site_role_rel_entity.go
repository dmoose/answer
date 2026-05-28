package entity

import "time"

type UserSiteRoleRel struct {
	ID        int       `xorm:"not null pk autoincr INT(11) id"`
	CreatedAt time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt time.Time `xorm:"updated TIMESTAMP updated_at"`
	UserID    string    `xorm:"not null BIGINT(20) UNIQUE(ux_user_site) user_id"`
	SiteID    string    `xorm:"not null VARCHAR(36) UNIQUE(ux_user_site) site_id"`
	RoleID    int       `xorm:"not null default 1 INT(11) role_id"`
}

func (UserSiteRoleRel) TableName() string {
	return "user_site_role_rel"
}
