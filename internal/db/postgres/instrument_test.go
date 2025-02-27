package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetInstrument_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrumentID := 1
	expectedInstrument := &models.Instrument{
		ID:         instrumentID,
		Title:      "United States Dollar",
		ShortTitle: "USD",
		Symbol:     "$",
		Rate:       1.0,
		Changed:    1234567890,
	}

	rows := mock.NewRows([]string{"id", "title", "short_title", "symbol", "rate", "changed"}).
		AddRow(expectedInstrument.ID, expectedInstrument.Title, expectedInstrument.ShortTitle, expectedInstrument.Symbol, expectedInstrument.Rate, expectedInstrument.Changed)

	mock.ExpectQuery(`SELECT id, title, short_title, symbol, rate, changed FROM instrument WHERE id = \$1`).
		WithArgs(instrumentID).
		WillReturnRows(rows)

	result, err := db.GetInstrument(context.Background(), instrumentID)
	assert.NoError(t, err)
	assert.Equal(t, expectedInstrument, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInstrument_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrumentID := 1

	mock.ExpectQuery(`SELECT id, title, short_title, symbol, rate, changed FROM instrument WHERE id = \$1`).
		WithArgs(instrumentID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetInstrument(context.Background(), instrumentID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("instrument not found: %d", instrumentID))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInstrument_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrumentID := 1

	mock.ExpectQuery(`SELECT id, title, short_title, symbol, rate, changed FROM instrument WHERE id = \$1`).
		WithArgs(instrumentID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetInstrument(context.Background(), instrumentID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get instrument")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListInstruments_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	rows := mock.NewRows([]string{"id", "title", "short_title", "symbol", "rate", "changed"}).
		AddRow(1, "United States Dollar", "USD", "$", 1.0, 1234567890).
		AddRow(2, "Euro", "EUR", "€", 0.85, 1234567891)

	mock.ExpectQuery(`SELECT id, title, short_title, symbol, rate, changed FROM instrument WHERE user_id = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	instruments, err := db.ListInstruments(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, instruments, 2)
	assert.Equal(t, 1, instruments[0].ID)
	assert.Equal(t, "United States Dollar", instruments[0].Title)
	assert.Equal(t, "USD", instruments[0].ShortTitle)
	assert.Equal(t, "$", instruments[0].Symbol)
	assert.Equal(t, 1.0, instruments[0].Rate)
	assert.Equal(t, 1234567890, instruments[0].Changed)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListInstruments_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, title, short_title, symbol, rate, changed FROM instrument WHERE user_id = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	instruments, err := db.ListInstruments(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, instruments)
	assert.Contains(t, err.Error(), "failed to list instruments")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateInstrument_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrument := &models.Instrument{
		ID:         1,
		Title:      "United States Dollar",
		ShortTitle: "USD",
		Symbol:     "$",
		Rate:       1.0,
		Changed:    1234567890,
	}

	mock.ExpectExec(`INSERT INTO instrument \(id, title, short_title, symbol, rate, changed\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\)`).
		WithArgs(instrument.ID, instrument.Title, instrument.ShortTitle, instrument.Symbol, instrument.Rate, instrument.Changed).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateInstrument(context.Background(), instrument)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateInstrument_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrument := &models.Instrument{
		ID:         1,
		Title:      "United States Dollar",
		ShortTitle: "USD",
		Symbol:     "$",
		Rate:       1.0,
		Changed:    1234567890,
	}

	mock.ExpectExec(`INSERT INTO instrument \(id, title, short_title, symbol, rate, changed\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\)`).
		WithArgs(instrument.ID, instrument.Title, instrument.ShortTitle, instrument.Symbol, instrument.Rate, instrument.Changed).
		WillReturnError(errors.New("insert error"))

	err = db.CreateInstrument(context.Background(), instrument)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create instrument")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateInstrument_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrument := &models.Instrument{
		ID:         1,
		Title:      "Updated Title",
		ShortTitle: "UTL",
		Symbol:     "U$",
		Rate:       1.1,
		Changed:    1234567891,
	}

	mock.ExpectExec(`UPDATE instrument SET title = \$2, short_title = \$3, symbol = \$4, rate = \$5, changed = \$6 WHERE id = \$1`).
		WithArgs(instrument.ID, instrument.Title, instrument.ShortTitle, instrument.Symbol, instrument.Rate, instrument.Changed).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateInstrument(context.Background(), instrument)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateInstrument_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrument := &models.Instrument{
		ID:         1,
		Title:      "Updated Title",
		ShortTitle: "UTL",
		Symbol:     "U$",
		Rate:       1.1,
		Changed:    1234567891,
	}

	mock.ExpectExec(`UPDATE instrument SET title = \$2, short_title = \$3, symbol = \$4, rate = \$5, changed = \$6 WHERE id = \$1`).
		WithArgs(instrument.ID, instrument.Title, instrument.ShortTitle, instrument.Symbol, instrument.Rate, instrument.Changed).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateInstrument(context.Background(), instrument)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "instrument not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateInstrument_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrument := &models.Instrument{
		ID:         1,
		Title:      "Updated Title",
		ShortTitle: "UTL",
		Symbol:     "U$",
		Rate:       1.1,
		Changed:    1234567891,
	}

	mock.ExpectExec(`UPDATE instrument SET title = \$2, short_title = \$3, symbol = \$4, rate = \$5, changed = \$6 WHERE id = \$1`).
		WithArgs(instrument.ID, instrument.Title, instrument.ShortTitle, instrument.Symbol, instrument.Rate, instrument.Changed).
		WillReturnError(errors.New("update error"))

	err = db.UpdateInstrument(context.Background(), instrument)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update instrument")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteInstrument_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrumentID := 1

	mock.ExpectExec(`DELETE FROM instrument WHERE id = \$1`).
		WithArgs(instrumentID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteInstrument(context.Background(), instrumentID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteInstrument_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrumentID := 1

	mock.ExpectExec(`DELETE FROM instrument WHERE id = \$1`).
		WithArgs(instrumentID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteInstrument(context.Background(), instrumentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "instrument not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteInstrument_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	instrumentID := 1

	mock.ExpectExec(`DELETE FROM instrument WHERE id = \$1`).
		WithArgs(instrumentID).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteInstrument(context.Background(), instrumentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete instrument")

	assert.NoError(t, mock.ExpectationsWereMet())
}
