//go:build integration

package integration_test

import (
	"context"
	"testing"

	"go-shortener/services/url-api/model"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func setupPostgres(t *testing.T) (sqlx.SqlConn, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("shortener"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.WithInitScripts(
			"../../services/migrations/000001_create_urls.up.sql",
			"../../services/migrations/000002_create_clicks.up.sql",
			"../../services/migrations/000003_add_clicks_enrichment.up.sql",
		),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	conn := sqlx.NewSqlConn("pgx", connStr)

	cleanup := func() {
		container.Terminate(ctx)
	}

	return conn, cleanup
}

func TestShortenIntegration(t *testing.T) {
	conn, cleanup := setupPostgres(t)
	defer cleanup()

	urlModel := model.NewUrlsModel(conn)
	ctx := context.Background()

	// Test: Insert a new URL
	id := uuid.Must(uuid.NewV7())
	_, err := urlModel.Insert(ctx, &model.Urls{
		Id:          id.String(),
		ShortCode:   "testcode",
		OriginalUrl: "https://example.com/integration-test",
	})
	require.NoError(t, err)

	// Test: Find by short code
	found, err := urlModel.FindOneByShortCode(ctx, "testcode")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/integration-test", found.OriginalUrl)
	assert.Equal(t, "testcode", found.ShortCode)

	// Test: Find by ID
	foundByID, err := urlModel.FindOne(ctx, id.String())
	require.NoError(t, err)
	assert.Equal(t, "testcode", foundByID.ShortCode)
}
