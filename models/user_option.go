package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type UserOption struct {
	ID     int64  `json:"" db:"id"`
	UserID int64  `json:"" db:"user_id"`
	Option string `json:"" db:"option"`
	Data   string `json:"" db:"data"`
}

type userOptionChange struct {
	BaseChange
	UserOption
}

type UserOptionStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	options     map[int64]UserOption
	userOptions map[int64]map[int64]struct{}
	mutex       sync.RWMutex
}

func NewUserOptionStore(
	db *sql.DB, table, changeTable string,
) *UserOptionStore {
	store := UserOptionStore{
		table:       table,
		changeTable: changeTable,
		options:     make(map[int64]UserOption),
		userOptions: make(map[int64]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *UserOptionStore) Create(m *UserOption) error {
	change := userOptionChange{
		BaseChange: BaseChange{Type: CreateChange},
		UserOption: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.UserOption
	return nil
}

func (s *UserOptionStore) Update(m *UserOption) error {
	change := userOptionChange{
		BaseChange: BaseChange{Type: UpdateChange},
		UserOption: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.UserOption
	return nil
}

func (s *UserOptionStore) Delete(id int64) error {
	change := userOptionChange{
		BaseChange: BaseChange{Type: DeleteChange},
		UserOption: UserOption{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *UserOptionStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *UserOptionStore) initChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *UserOptionStore) loadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "user_id", "option", "data"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *UserOptionStore) scanChange(scan Scanner) (Change, error) {
	option := userOptionChange{}
	err := scan.Scan(
		&option.BaseChange.ID, &option.Type, &option.Time,
		&option.UserOption.ID, &option.UserID, &option.Option,
		&option.Data,
	)
	return &option, err
}

func (s *UserOptionStore) saveChange(tx *sql.Tx, change Change) error {
	option := change.(*userOptionChange)
	option.Time = time.Now().Unix()
	switch option.Type {
	case CreateChange:
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("user_id", "option", "data")`+
					` VALUES ($1, $2, $3)`,
				s.table,
			),
			option.UserID, option.Option, option.Data,
		)
		if err != nil {
			return err
		}
		option.UserOption.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.options[option.UserOption.ID]; !ok {
			return fmt.Errorf(
				"user option with id = %d does not exists",
				option.UserOption.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET`+
					` "user_id" = $1, "option" = $2, "data" = $3`+
					` WHERE "id" = $4`,
				s.table,
			),
			option.UserID, option.Option, option.Data,
			option.UserOption.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.options[option.UserOption.ID]; !ok {
			return fmt.Errorf(
				"user option with id = %d does not exists",
				option.UserOption.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			option.UserOption.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			option.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s" `+
				`("change_type", "change_time", `+
				`"id", "user_id", "option", "data") `+
				`VALUES ($1, $2, $3, $4, $5, $6)`,
			s.changeTable,
		),
		option.Type, option.Time, option.UserOption.ID,
		option.UserID, option.Option, option.Data,
	)
	if err != nil {
		return err
	}
	option.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *UserOptionStore) applyChange(change Change) {
	option := change.(*userOptionChange)
	switch option.Type {
	case UpdateChange:
		if oldOption, ok := s.options[option.UserOption.ID]; ok {
			if oldOption.UserID != option.UserID {
				if userOptions, ok := s.userOptions[oldOption.UserID]; ok {
					delete(userOptions, oldOption.ID)
					if len(userOptions) == 0 {
						delete(s.userOptions, option.UserID)
					}
				}
			}
		}
		fallthrough
	case CreateChange:
		if _, ok := s.userOptions[option.UserID]; !ok {
			s.userOptions[option.UserID] = make(map[int64]struct{})
		}
		s.userOptions[option.UserID][option.UserOption.ID] = struct{}{}
		s.options[option.UserOption.ID] = option.UserOption
	case DeleteChange:
		if userOptions, ok := s.userOptions[option.UserID]; ok {
			delete(userOptions, option.UserOption.ID)
			if len(userOptions) == 0 {
				delete(s.userOptions, option.UserID)
			}
		}
		delete(s.options, option.UserOption.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			option.Type,
		))
	}
}
