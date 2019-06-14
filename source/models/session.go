package models

import (
	"../tools"
)

type SessionStore struct {
}

type Session struct {
	ID         int64  `db:"id"          json:""`
	UserID     int64  `db:"user_id"     json:""`
	Secret     string `db:"secret"      json:""`
	CreateTime int64  `db:"create_time" json:""`
}

func (s *SessionStore) Create(session *Session) error {
	return tools.NotImplementedError
}

func (s *SessionStore) Update(session *Session) error {
	return tools.NotImplementedError
}

func (s *SessionStore) Delete(id int64) error {
	return tools.NotImplementedError
}
