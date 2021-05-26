package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/couchbaselabs/cbmultimanager/values"
)

func (db *DB) AddUser(user *values.User) error {
	_, err := db.sqlDB.Exec("INSERT INTO users (user, password, admin) VALUES (?, ?, ?);", user.User, user.Password,
		user.Admin)
	if err != nil {
		return fmt.Errorf("could not add user: %w", err)
	}

	return nil
}

func (db *DB) GetUser(user string) (*values.User, error) {
	result := db.sqlDB.QueryRow("SELECT password, admin FROM users WHERE user = ?;", user)
	returnUser := &values.User{User: user}
	if err := result.Scan(&returnUser.Password, &returnUser.Admin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, values.ErrNotFound
		}

		return nil, fmt.Errorf("could not get user: %w", err)
	}

	return returnUser, nil
}
