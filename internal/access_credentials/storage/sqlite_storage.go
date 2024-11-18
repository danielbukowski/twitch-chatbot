package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nicklaw5/helix/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

const databaseRequestTimeout = 3 * time.Second

var tracer = otel.Tracer("github.com/danielbukowski/twitch-chatbot/internal/access_credentials/storage")

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

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) Retrieve(ctx context.Context, channelName string) (helix.AccessCredentials, error) {
	query := "SELECT details FROM access_credentials WHERE channel_name = ?;"

	ctx, span := tracer.Start(ctx, "retrieve")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	span.AddEvent("executing the query")
	row := s.db.QueryRowContext(ctx, query, channelName)
	span.AddEvent("executed the query")

	var details string

	span.AddEvent("copying the result to a struct")
	err := row.Scan(&details)
	if err != nil {
		errMsg := "failed to copy the result to a struct"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return helix.AccessCredentials{}, errors.Join(errors.New(errMsg), err)
	}
	span.AddEvent("successfully copied the result to a struct")

	span.AddEvent("decrypting access credentials")
	accessCredentials, err := s.accessCredentialsCipher.Decrypt(details)
	if err != nil {
		errMsg := "failed to decrypt access credentials"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return helix.AccessCredentials{}, errors.Join(errors.New(errMsg), err)
	}

	span.SetStatus(codes.Ok, "successfully decrypted access credentials")
	return accessCredentials, nil
}

func (s *SQLiteStorage) Save(ctx context.Context, accessCredentials helix.AccessCredentials, channelName string) error {
	query := "INSERT INTO access_credentials (channel_name, details) VALUES (?, ?);"

	ctx, span := tracer.Start(ctx, "save")
	defer span.End()

	span.AddEvent("encrypting access credentials")
	base64AccessCredentials, err := s.accessCredentialsCipher.Encrypt(accessCredentials)
	if err != nil {
		errMsg := "failed to encrypt access credentials"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}
	span.AddEvent("successfully encrypted access credentials")

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	span.AddEvent("creating a prepared statement")
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		errMsg := "failed to create a prepared statement"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}

	span.AddEvent("trying to save access credentials to the database")
	res, err := stmt.ExecContext(ctx, channelName, base64AccessCredentials)
	if err != nil {
		errMsg := "failed to execute a prepared statement"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}

	if rows, err := res.RowsAffected(); err != nil || rows == 0 {
		errMsg := "failed to insert access credentials to the database"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}

	span.SetStatus(codes.Ok, "successfully saved access credentials to the database")
	return nil
}

func (s *SQLiteStorage) Update(ctx context.Context, accessCredentials helix.AccessCredentials, channelName string) error {
	query := "UPDATE access_credentials SET details = ? WHERE channel_name = ?;"

	ctx, span := tracer.Start(ctx, "update")
	defer span.End()

	span.AddEvent("encrypting access credentials")
	base64AccessCredentials, err := s.accessCredentialsCipher.Encrypt(accessCredentials)
	if err != nil {
		errMsg := "failed to encrypt access credentials"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}
	span.AddEvent("successfully encrypted access credentials")

	ctx, cancel := context.WithTimeout(ctx, databaseRequestTimeout)
	defer cancel()

	span.AddEvent("creating a prepared statement")
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		errMsg := "failed to create a prepared statement"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}
	span.AddEvent("successfully created a prepared statement")

	span.AddEvent("trying to update new access credentials to the database")
	res, err := stmt.ExecContext(ctx, base64AccessCredentials, channelName)
	if err != nil {
		errMsg := "failed to execute a prepared statement"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}

	if rows, err := res.RowsAffected(); err != nil || rows == 0 {
		errMsg := "failed to update new access credentials to the database"
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(err)
		return errors.Join(errors.New(errMsg), err)
	}

	span.SetStatus(codes.Ok, "successfully updated new access credentials to the database")
	return nil
}
