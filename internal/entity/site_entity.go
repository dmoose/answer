package entity

import "time"

const (
	SiteStatusActive    = 1
	SiteStatusSuspended = 9
)

type Site struct {
	ID          string    `xorm:"not null pk VARCHAR(36) id" json:"id"`
	CreatedAt   time.Time `xorm:"created TIMESTAMP created_at" json:"created_at"`
	UpdatedAt   time.Time `xorm:"updated TIMESTAMP updated_at" json:"updated_at"`
	Name        string    `xorm:"not null VARCHAR(255) name" json:"name"`
	Slug        string    `xorm:"not null unique VARCHAR(64) slug" json:"slug"`
	Description string    `xorm:"TEXT description" json:"description"`
	Status      int       `xorm:"not null default 1 INT(11) status" json:"status"`
	IconURL     string    `xorm:"VARCHAR(512) icon_url" json:"icon_url"`
	BaseURL     string    `xorm:"VARCHAR(512) base_url" json:"base_url"`
}

func (Site) TableName() string {
	return "site"
}
