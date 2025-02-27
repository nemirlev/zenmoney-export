package postgres

import (
	"context"
	"errors"
	"github.com/nemirlev/zenmoney-export/v2/internal/interfaces"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nemirlev/zenmoney-go-sdk/v2/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetTag_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tagID := "test-id"
	expectedTag := &models.Tag{
		ID:            tagID,
		User:          1,
		Changed:       1234567890,
		Icon:          ptr("icon"),
		BudgetIncome:  true,
		BudgetOutcome: false,
		Required:      ptr(true),
		Color:         ptr(int64(123456)),
		Picture:       ptr("picture"),
		Title:         "Test Tag",
		ShowIncome:    true,
		ShowOutcome:   false,
		Parent:        ptr("parent-id"),
		StaticID:      "static-id",
	}

	rows := mock.NewRows([]string{
		"id", "user", "changed", "icon", "budget_income", "budget_outcome",
		"required", "color", "picture", "title", "show_income", "show_outcome",
		"parent", "static_id",
	}).AddRow(
		expectedTag.ID, expectedTag.User, expectedTag.Changed, expectedTag.Icon,
		expectedTag.BudgetIncome, expectedTag.BudgetOutcome, expectedTag.Required,
		expectedTag.Color, expectedTag.Picture, expectedTag.Title, expectedTag.ShowIncome,
		expectedTag.ShowOutcome, expectedTag.Parent, expectedTag.StaticID,
	)

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnRows(rows)

	result, err := db.GetTag(context.Background(), tagID)
	assert.NoError(t, err)
	assert.Equal(t, expectedTag, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTag_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tagID := "non-existing-id"

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetTag(context.Background(), tagID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTag_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tagID := "test-id"

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetTag(context.Background(), tagID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get tag")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTags_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	rows := mock.NewRows([]string{
		"id", "user", "changed", "icon", "budget_income", "budget_outcome",
		"required", "color", "picture", "title", "show_income", "show_outcome",
		"parent", "static_id",
	}).AddRow(
		"test-id", 1, 1234567890, ptr("icon"), true, false,
		ptr(true), ptr(int64(123456)), ptr("picture"), "Test Tag", true, false,
		ptr("parent-id"), "static-id",
	)

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	tags, err := db.ListTags(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, "test-id", tags[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTags_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	tags, err := db.ListTags(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, tags)
	assert.Contains(t, err.Error(), "failed to list tags")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListTags_NoResults(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	filter := interfaces.Filter{
		UserID: ptr(1),
		Limit:  10,
		Page:   1,
	}

	rows := mock.NewRows([]string{
		"id", "user", "changed", "icon", "budget_income", "budget_outcome",
		"required", "color", "picture", "title", "show_income", "show_outcome",
		"parent", "static_id",
	})

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	tags, err := db.ListTags(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, tags, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTag_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tag := &models.Tag{
		ID:            "test-id",
		User:          1,
		Changed:       1234567890,
		Icon:          ptr("icon"),
		BudgetIncome:  true,
		BudgetOutcome: false,
		Required:      ptr(true),
		Color:         ptr(int64(123456)),
		Picture:       ptr("picture"),
		Title:         "Test Tag",
		ShowIncome:    true,
		ShowOutcome:   false,
		Parent:        ptr("parent-id"),
		StaticID:      "static-id",
	}

	mock.ExpectExec(`INSERT INTO tag`).
		WithArgs(
			tag.ID, tag.User, tag.Changed, tag.Icon, tag.BudgetIncome, tag.BudgetOutcome,
			tag.Required, tag.Color, tag.Picture, tag.Title, tag.ShowIncome, tag.ShowOutcome,
			tag.Parent, tag.StaticID,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.CreateTag(context.Background(), tag)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTag_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tag := &models.Tag{
		ID:            "test-id",
		User:          1,
		Changed:       1234567890,
		Icon:          ptr("icon"),
		BudgetIncome:  true,
		BudgetOutcome: false,
		Required:      ptr(true),
		Color:         ptr(int64(123456)),
		Picture:       ptr("picture"),
		Title:         "Test Tag",
		ShowIncome:    true,
		ShowOutcome:   false,
		Parent:        ptr("parent-id"),
		StaticID:      "static-id",
	}

	mock.ExpectExec(`INSERT INTO tag`).
		WithArgs(
			tag.ID, tag.User, tag.Changed, tag.Icon, tag.BudgetIncome, tag.BudgetOutcome,
			tag.Required, tag.Color, tag.Picture, tag.Title, tag.ShowIncome, tag.ShowOutcome,
			tag.Parent, tag.StaticID,
		).
		WillReturnError(errors.New("insert error"))

	err = db.CreateTag(context.Background(), tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create tag")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTag_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tag := &models.Tag{
		ID:            "test-id",
		User:          1,
		Changed:       1234567890,
		Icon:          ptr("icon"),
		BudgetIncome:  true,
		BudgetOutcome: false,
		Required:      ptr(true),
		Color:         ptr(int64(123456)),
		Picture:       ptr("picture"),
		Title:         "Updated Tag",
		ShowIncome:    true,
		ShowOutcome:   false,
		Parent:        ptr("parent-id"),
		StaticID:      "static-id",
	}

	mock.ExpectExec(`UPDATE tag SET`).
		WithArgs(
			tag.ID, tag.User, tag.Changed, tag.Icon, tag.BudgetIncome, tag.BudgetOutcome,
			tag.Required, tag.Color, tag.Picture, tag.Title, tag.ShowIncome, tag.ShowOutcome,
			tag.Parent, tag.StaticID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = db.UpdateTag(context.Background(), tag)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTag_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tag := &models.Tag{
		ID:            "non-existing-id",
		User:          1,
		Changed:       1234567890,
		Icon:          ptr("icon"),
		BudgetIncome:  true,
		BudgetOutcome: false,
		Required:      ptr(true),
		Color:         ptr(int64(123456)),
		Picture:       ptr("picture"),
		Title:         "Updated Tag",
		ShowIncome:    true,
		ShowOutcome:   false,
		Parent:        ptr("parent-id"),
		StaticID:      "static-id",
	}

	mock.ExpectExec(`UPDATE tag SET`).
		WithArgs(
			tag.ID, tag.User, tag.Changed, tag.Icon, tag.BudgetIncome, tag.BudgetOutcome,
			tag.Required, tag.Color, tag.Picture, tag.Title, tag.ShowIncome, tag.ShowOutcome,
			tag.Parent, tag.StaticID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = db.UpdateTag(context.Background(), tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTag_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tag := &models.Tag{
		ID:            "test-id",
		User:          1,
		Changed:       1234567890,
		Icon:          ptr("icon"),
		BudgetIncome:  true,
		BudgetOutcome: false,
		Required:      ptr(true),
		Color:         ptr(int64(123456)),
		Picture:       ptr("picture"),
		Title:         "Updated Tag",
		ShowIncome:    true,
		ShowOutcome:   false,
		Parent:        ptr("parent-id"),
		StaticID:      "static-id",
	}

	mock.ExpectExec(`UPDATE tag SET`).
		WithArgs(
			tag.ID, tag.User, tag.Changed, tag.Icon, tag.BudgetIncome, tag.BudgetOutcome,
			tag.Required, tag.Color, tag.Picture, tag.Title, tag.ShowIncome, tag.ShowOutcome,
			tag.Parent, tag.StaticID,
		).
		WillReturnError(errors.New("update error"))

	err = db.UpdateTag(context.Background(), tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update tag")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTag_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tagID := "test-id"

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = db.DeleteTag(context.Background(), tagID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTag_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tagID := "non-existing-id"

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = db.DeleteTag(context.Background(), tagID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTag_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	db := &DB{pool: mock}

	tagID := "test-id"

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnError(errors.New("delete error"))

	err = db.DeleteTag(context.Background(), tagID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete tag")

	assert.NoError(t, mock.ExpectationsWereMet())
}
