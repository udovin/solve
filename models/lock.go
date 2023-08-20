package models

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

type Lock struct {
	ID         int64  `db:"id"`
	Name       string `db:"name"`
	Token      int64  `db:"token"`
	ExpireTime int64  `db:"expire_time"`
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

func (s *LockStore) AcquireByName(ctx context.Context, name string) (*LockGuard, error) {
	lock, err := s.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if lock.ExpireTime >= time.Now().Unix() {
		return nil, ErrLockAcquired
	}
	token, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	expireTime := time.Now().Add(lockTimeout).Unix()
	guard := LockGuard{
		store: s,
		lock:  lock,
	}
	if err := guard.update(ctx, token.Int64()+1, expireTime); err != nil {
		if errors.Is(err, ErrLockReleased) {
			return nil, ErrLockAcquired
		}
		return nil, err
	}
	return nil, err
}

type LockGuard struct {
	store *LockStore
	lock  Lock
	mutex sync.Mutex
}

func (l *LockGuard) Ping(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	expireTime := time.Now().Add(lockTimeout).Unix()
	return l.update(ctx, l.lock.Token, expireTime)
}

func (l *LockGuard) Release(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.lock.ExpireTime < time.Now().Unix() {
		return ErrLockReleased
	}
	return l.update(ctx, 0, 0)
}

func (l *LockGuard) update(ctx context.Context, token, expireTime int64) error {
	if tx := db.GetTx(ctx); tx != nil {
		return fmt.Errorf("ping cannot be run in transaction")
	}
	query := l.store.db.Update(l.store.table)
	query.SetNames("token", "expire_time")
	query.SetValues(token, expireTime)
	query.SetWhere(gosql.Column("id").Equal(l.lock.ID).
		And(gosql.Column("token").Equal(l.lock.Token)).
		And(gosql.Column("expire_time").Equal(l.lock.ExpireTime)),
	)
	rawQuery, values := l.store.db.Build(query)
	res, err := l.store.db.ExecContext(ctx, rawQuery, values...)
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
	l.lock.Token = token
	l.lock.ExpireTime = expireTime
	return nil
}

func NewLockStore(db *gosql.DB, table string) *LockStore {
	return &LockStore{
		db:    db,
		table: table,
	}
}
