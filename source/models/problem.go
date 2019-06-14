package models

import (
	"../tools"
)

type ProblemStore struct {
}

type Problem struct {
	ID         int64 `db:"id"          json:""`
	CreateTime int64 `db:"create_time" json:""`
}

func (s *ProblemStore) Create(problem *Problem) error {
	return tools.NotImplementedError
}

func (s *ProblemStore) Update(problem *Problem) error {
	return tools.NotImplementedError
}

func (s *ProblemStore) Delete(id int64) error {
	return tools.NotImplementedError
}
