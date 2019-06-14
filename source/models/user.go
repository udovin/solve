package models

import (
	"../tools"
)

type UserStore struct {
}

type User struct {
	ID           int64  `db:"id"          json:""`
	Login        string `db:"login"       json:""`
	CreateTime   int64  `db:"create_time" json:""`
	PasswordHash string `db:"password_hash"`
	PasswordSalt string `db:"password_salt"`
}

func (s *UserStore) Create(user *User) error {
	return tools.NotImplementedError
}

func (s *UserStore) Update(user *User) error {
	return tools.NotImplementedError
}

func (s *UserStore) Delete(id int64) error {
	return tools.NotImplementedError
}
