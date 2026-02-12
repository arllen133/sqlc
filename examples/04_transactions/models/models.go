package models

type Account struct {
	ID      int64 `db:"id,primaryKey,autoIncrement"`
	Balance int   `db:"balance"`
}
