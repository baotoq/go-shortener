//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	clicksModel "go-shortener/services/analytics-rpc/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClicksIntegration(t *testing.T) {
	conn, cleanup := setupPostgres(t)
	defer cleanup()

	model := clicksModel.NewClicksModel(conn)
	ctx := context.Background()

	// Test: Insert a click record
	clickID := uuid.Must(uuid.NewV7())
	_, err := model.Insert(ctx, &clicksModel.Clicks{
		Id:            clickID.String(),
		ShortCode:     "testcode",
		ClickedAt:     time.Now(),
		CountryCode:   "US",
		DeviceType:    "Desktop",
		TrafficSource: "Direct",
	})
	require.NoError(t, err)

	// Test: Count by short code
	count, err := model.CountByShortCode(ctx, "testcode")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Test: Insert another click for same short code
	clickID2 := uuid.Must(uuid.NewV7())
	_, err = model.Insert(ctx, &clicksModel.Clicks{
		Id:            clickID2.String(),
		ShortCode:     "testcode",
		ClickedAt:     time.Now(),
		CountryCode:   "GB",
		DeviceType:    "Mobile",
		TrafficSource: "Social",
	})
	require.NoError(t, err)

	count, err = model.CountByShortCode(ctx, "testcode")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Test: Count for non-existent short code returns 0
	count, err = model.CountByShortCode(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
