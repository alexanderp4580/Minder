package store

import (
	"database/sql"
	"errors"

	"github.com/alexanderp4580/Minder/server/domain"
)

func rowError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
