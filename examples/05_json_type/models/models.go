package models

import "github.com/arllen133/sqlc"

type Settings struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
}

type UserConfig struct {
	ID       int64               `db:"id,primaryKey,autoIncrement"`
	Username string              `db:"username"`
	Settings sqlc.JSON[Settings] `db:"settings,type:json"`
}
