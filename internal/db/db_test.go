package db

import (
	"github.com/nemirlev/zenapi"
	"testing"

	mocks "github.com/nemirlev/zenexport/mocks/internal_/db"
	"github.com/stretchr/testify/assert"
)

func TestDataStore_Save(t *testing.T) {
	mockStore := mocks.NewDataStore(t)
	data := &zenapi.Response{}
	mockStore.EXPECT().Save(data).Return(nil)

	err := mockStore.Save(data)

	assert.NoError(t, err)
}

func TestDataStore_Update(t *testing.T) {
	mockStore := mocks.NewDataStore(t)
	data := &zenapi.Response{}
	mockStore.EXPECT().Update(data).Return(nil)

	err := mockStore.Update(data)

	assert.NoError(t, err)
}

func TestDataStore_Delete(t *testing.T) {
	mockStore := mocks.NewDataStore(t)
	data := &zenapi.Deletion{}
	mockStore.EXPECT().Delete(data).Return(nil)

	err := mockStore.Delete(data)

	assert.NoError(t, err)
}
