package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetTag_Success(t *testing.T) {
	db, mock := newTestDB(t)

	tagID := "test-id"
	expectedTag := testTag(tagID, "Test Tag")

	rows := mock.NewRows([]string{
		"id", "user", "changed", "icon", "budget_income", "budget_outcome",
		"required", "archive", "color", "picture", "title", "show_income", "show_outcome",
		"parent", "static_id",
	}).AddRow(
		expectedTag.ID, expectedTag.User, expectedTag.Changed, expectedTag.Icon,
		expectedTag.BudgetIncome, expectedTag.BudgetOutcome, expectedTag.Required,
		expectedTag.Archive, expectedTag.Color, expectedTag.Picture, expectedTag.Title, expectedTag.ShowIncome,
		expectedTag.ShowOutcome, expectedTag.Parent, expectedTag.StaticID,
	)

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, archive, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnRows(rows)

	result, err := db.GetTag(context.Background(), tagID)
	assert.NoError(t, err)
	assert.Equal(t, expectedTag, result)
}

func TestGetTag_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	tagID := "non-existing-id"

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, archive, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnError(pgx.ErrNoRows)

	result, err := db.GetTag(context.Background(), tagID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")
}

func TestGetTag_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	tagID := "test-id"

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, archive, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnError(errors.New("query error"))

	result, err := db.GetTag(context.Background(), tagID)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get tag")
}

func TestListTags_Success(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	rows := mock.NewRows([]string{
		"id", "user", "changed", "icon", "budget_income", "budget_outcome",
		"required", "archive", "color", "picture", "title", "show_income", "show_outcome",
		"parent", "static_id",
	}).AddRow(
		"test-id", 1, 1234567890, new("icon"), true, false,
		new(true), true, new(int64(123456)), new("picture"), "Test Tag", true, false,
		new("parent-id"), "static-id",
	)

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, archive, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	tags, err := db.ListTags(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, "test-id", tags[0].ID)
}

func TestListTags_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, archive, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnError(errors.New("query error"))

	tags, err := db.ListTags(context.Background(), filter)
	assert.Error(t, err)
	assert.Nil(t, tags)
	assert.Contains(t, err.Error(), "failed to list tags")
}

func TestListTags_NoResults(t *testing.T) {
	db, mock := newTestDB(t)

	filter := testPageFilter()

	rows := mock.NewRows([]string{
		"id", "user", "changed", "icon", "budget_income", "budget_outcome",
		"required", "archive", "color", "picture", "title", "show_income", "show_outcome",
		"parent", "static_id",
	})

	mock.ExpectQuery(`SELECT id, "user", changed, icon, budget_income, budget_outcome, required, archive, color, picture, title, show_income, show_outcome, parent, static_id FROM tag WHERE "user" = \$1 LIMIT \$2 OFFSET \$3`).
		WithArgs(1, 10, 0).
		WillReturnRows(rows)

	tags, err := db.ListTags(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, tags, 0)
}

func TestCreateTag_Success(t *testing.T) {
	db, mock := newTestDB(t)

	tag := testTag("test-id", "Test Tag")

	mock.ExpectExec(`INSERT INTO tag`).
		WithArgs(tagArgs(tag)...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.CreateTag(context.Background(), tag)
	assert.NoError(t, err)
}

func TestCreateTag_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	tag := testTag("test-id", "Test Tag")

	mock.ExpectExec(`INSERT INTO tag`).
		WithArgs(tagArgs(tag)...).
		WillReturnError(errors.New("insert error"))

	err := db.CreateTag(context.Background(), tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create tag")
}

func TestUpdateTag_Success(t *testing.T) {
	db, mock := newTestDB(t)

	tag := testTag("test-id", "Updated Tag")

	mock.ExpectExec(`UPDATE tag SET`).
		WithArgs(tagArgs(tag)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateTag(context.Background(), tag)
	assert.NoError(t, err)
}

func TestUpdateTag_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	tag := testTag("non-existing-id", "Updated Tag")

	mock.ExpectExec(`UPDATE tag SET`).
		WithArgs(tagArgs(tag)...).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := db.UpdateTag(context.Background(), tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")
}

func TestUpdateTag_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	tag := testTag("test-id", "Updated Tag")

	mock.ExpectExec(`UPDATE tag SET`).
		WithArgs(tagArgs(tag)...).
		WillReturnError(errors.New("update error"))

	err := db.UpdateTag(context.Background(), tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update tag")
}

func TestDeleteTag_Success(t *testing.T) {
	db, mock := newTestDB(t)

	tagID := "test-id"

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteTag(context.Background(), tagID)
	assert.NoError(t, err)
}

func TestDeleteTag_NotFound(t *testing.T) {
	db, mock := newTestDB(t)

	tagID := "non-existing-id"

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := db.DeleteTag(context.Background(), tagID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")
}

func TestDeleteTag_QueryError(t *testing.T) {
	db, mock := newTestDB(t)

	tagID := "test-id"

	mock.ExpectExec(`DELETE FROM tag WHERE id = \$1`).
		WithArgs(tagID).
		WillReturnError(errors.New("delete error"))

	err := db.DeleteTag(context.Background(), tagID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete tag")
}
