package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	lg "github.com/danielbukowski/twitch-chatbot/internal/logger"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
)

const databaseRequestTimeout = 3 * time.Second

type accessCredentialsCipher interface {
	Encrypt(accessCredentials helix.AccessCredentials) (string, error)
	Decrypt(base64SaltNonceCiphertext string) (helix.AccessCredentials, error)
}

type SQLiteStorage struct {
	db                      *sql.DB
	accessCredentialsCipher accessCredentialsCipher
	logger                  *zap.Logger
}

func NewSQLiteStorage(ctx context.Context, dataSourceName, username, password string, accessCredentialsCipher accessCredentialsCipher, logger *zap.Logger) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_auth&_auth_user=%s&_auth_pass=%s&_auth_crypt=SHA384", dataSourceName, username, password))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return &SQLiteStorage{
		db:                      db,
		accessCredentialsCipher: accessCredentialsCipher,
		logger:                  logger.Named("access_credentials/storage"),
	}, nil
}

func (s *SQLiteStorage) Retrieve(ctx context.Context, channelName string) (helix.AccessCredentials, error) {
	query := "SELECT details FROM access_credentials WHERE channel_name = ?;"

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	defer lg.Flush(s.logger)

	row := s.db.QueryRowContext(ctx, query, channelName)

	var details string
	err := row.Scan(&details)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	accessCredentials, err := s.accessCredentialsCipher.Decrypt(details)
	if err != nil {
		return helix.AccessCredentials{}, errors.Join(errors.New("failed to decrypt access credentials"), err)
	}

	return accessCredentials, nil
}

func (s *SQLiteStorage) Save(ctx context.Context, accessCredentials helix.AccessCredentials, channelName string) error {
	query := "INSERT INTO access_credentials (channel_name, details) VALUES (?, ?);"

	defer lg.Flush(s.logger)

	base64AccessCredentials, err := s.accessCredentialsCipher.Encrypt(accessCredentials)
	if err != nil {
		return errors.Join(errors.New("failed to encrypt access credentials"), err)
	}

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}

	res, err := stmt.ExecContext(ctx, channelName, base64AccessCredentials)
	if err != nil {
		return errors.Join(errors.New("failed to save access credentials to database"), err)
	}

	if rows, err := res.RowsAffected(); err != nil || rows == 0 {
		return errors.Join(errors.New("did not save access credentials"), err)
	}

	return nil
}

func (s *SQLiteStorage) Update(ctx context.Context, accessCredentials helix.AccessCredentials, channelName string) error {
	query := "UPDATE access_credentials SET details = ? WHERE channel_name = ?;"

	defer lg.Flush(s.logger)

	base64AccessCredentials, err := s.accessCredentialsCipher.Encrypt(accessCredentials)
	if err != nil {
		return errors.Join(errors.New("failed to encrypt access credentials"), err)
	}

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}

	res, err := stmt.ExecContext(ctx, base64AccessCredentials, channelName)
	if err != nil {
		return err
	}

	if rows, err := res.RowsAffected(); err != nil || rows == 0 {
		return errors.Join(errors.New("did not save access credentials"), err)
	}

	return nil
}
