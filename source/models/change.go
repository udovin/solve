package models

type ChangeType int8

const (
	CreateChange ChangeType = 1
	DeleteChange ChangeType = 2
	UpdateChange ChangeType = 3
)

type Change struct {
	ID   int64      `db:"change_id" json:"ChangeID"`
	Type ChangeType `db:"change_type" json:"ChangeType"`
}
