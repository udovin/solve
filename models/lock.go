package models

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

type Lock struct {
	ID         int64  `db:"id"`
	Name       string `db:"name"`
	Token      int64  `db:"token"`
	ExpireTime int64  `db:"expire_time"`
	acquired   bool
}

type LockStore struct {
	db    *gosql.DB
	table string
}

var (
	ErrLockAcquired = fmt.Errorf("lock already acquired")
	ErrLockReleased = fmt.Errorf("lock already released")
)

const lockTimeout = time.Second * 15

func (s *LockStore) Get(ctx context.Context, id int64) (Lock, error) {
	query := s.db.Select(s.table)
	query.SetNames("id", "name", "token", "expire_time")
	query.SetWhere(gosql.Column("id").Equal(id))
	query.SetLimit(1)
	rawQuery, values := s.db.Build(query)
	row := s.db.QueryRowContext(ctx, rawQuery, values...)
	var lock Lock
	if err := row.Scan(&lock.ID, &lock.Name, &lock.Token, &lock.ExpireTime); err != nil {
		return lock, err
	}
	return lock, row.Err()
}

func (s *LockStore) GetByName(ctx context.Context, name string) (Lock, error) {
	query := s.db.Select(s.table)
	query.SetNames("id", "name", "token", "expire_time")
	query.SetWhere(gosql.Column("name").Equal(name))
	query.SetLimit(1)
	rawQuery, values := s.db.Build(query)
	row := s.db.QueryRowContext(ctx, rawQuery, values...)
	var lock Lock
	if err := row.Scan(&lock.ID, &lock.Name, &lock.Token, &lock.ExpireTime); err != nil {
		return lock, err
	}
	return lock, row.Err()
}

func (s *LockStore) AcquireLockByName(ctx context.Context, name string) (Lock, error) {
	lock, err := s.GetByName(ctx, name)
	if err != nil {
		return lock, err
	}
	if lock.ExpireTime >= time.Now().Unix() {
		return lock, ErrLockAcquired
	}
	token, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return lock, err
	}
	expireTime := time.Now().Add(lockTimeout).Unix()
	lock.acquired = true
	if err := s.updateLock(ctx, &lock, token.Int64()+1, expireTime); err != nil {
		if errors.Is(err, ErrLockReleased) {
			return lock, ErrLockAcquired
		}
		return lock, err
	}
	return lock, err
}

func (s *LockStore) ReleaseLock(ctx context.Context, lock *Lock) error {
	if lock.ExpireTime < time.Now().Unix() {
		lock.acquired = false
		return ErrLockReleased
	}
	return s.updateLock(ctx, lock, 0, 0)
}

func (s *LockStore) PingLock(ctx context.Context, lock *Lock) error {
	expireTime := time.Now().Add(lockTimeout).Unix()
	return s.updateLock(ctx, lock, lock.Token, expireTime)
}

func (s *LockStore) updateLock(ctx context.Context, lock *Lock, token, expireTime int64) error {
	if !lock.acquired {
		return ErrLockReleased
	}
	if tx := db.GetTx(ctx); tx != nil {
		return fmt.Errorf("ping cannot be run in transaction")
	}
	query := s.db.Update(s.table)
	query.SetNames("token", "expire_time")
	query.SetValues(token, expireTime)
	query.SetWhere(gosql.Column("id").Equal(lock.ID).
		And(gosql.Column("token").Equal(lock.Token)).
		And(gosql.Column("expire_time").Equal(lock.ExpireTime)),
	)
	rawQuery, values := s.db.Build(query)
	res, err := s.db.ExecContext(ctx, rawQuery, values...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrLockReleased
	}
	lock.Token = token
	lock.ExpireTime = expireTime
	return nil
}

func NewLockStore(db *gosql.DB, table string) *LockStore {
	return &LockStore{
		db:    db,
		table: table,
	}
}
