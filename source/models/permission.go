package models

import (
	"../tools"
)

type PermissionStore struct {
}

type Permission struct {
	ID   int64  `db:"id"   json:""`
	Code string `db:"code" json:""`
}

func (s *PermissionStore) Create(permission *Permission) error {
	return tools.NotImplementedError
}

func (s *PermissionStore) Update(permission *Permission) error {
	return tools.NotImplementedError
}

func (s *PermissionStore) Delete(id int64) error {
	return tools.NotImplementedError
}
