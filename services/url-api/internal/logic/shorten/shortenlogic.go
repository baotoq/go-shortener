// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package shorten

import (
	"context"
	"strings"

	"go-shortener/pkg/problemdetails"
	"go-shortener/services/url-api/internal/svc"
	"go-shortener/services/url-api/internal/types"
	"go-shortener/services/url-api/model"

	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	shortCodeLength = 8
	maxRetries      = 5
	alphabet        = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type ShortenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create short URL
func NewShortenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShortenLogic {
	return &ShortenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ShortenLogic) Shorten(req *types.ShortenRequest) (resp *types.ShortenResponse, err error) {
	logx.WithContext(l.ctx).Infow("shorten URL", logx.Field("original_url", req.OriginalUrl))

	// Generate UUIDv7 for primary key
	id, err := uuid.NewV7()
	if err != nil {
		logx.WithContext(l.ctx).Errorw("failed to generate UUIDv7", logx.Field("error", err.Error()))
		return nil, problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error", "failed to generate unique ID")
	}

	// Generate short code with collision retry
	var shortCode string
	for attempt := 0; attempt < maxRetries; attempt++ {
		code, genErr := gonanoid.Generate(alphabet, shortCodeLength)
		if genErr != nil {
			logx.WithContext(l.ctx).Errorw("failed to generate short code", logx.Field("error", genErr.Error()))
			return nil, problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error", "failed to generate short code")
		}

		_, insertErr := l.svcCtx.UrlModel.Insert(l.ctx, &model.Urls{
			Id:          id.String(),
			ShortCode:   code,
			OriginalUrl: req.OriginalUrl,
			ClickCount:  0,
		})

		if insertErr != nil {
			// Check for unique constraint violation (short_code collision)
			if isUniqueViolation(insertErr) {
				logx.WithContext(l.ctx).Infow("short code collision, retrying",
					logx.Field("attempt", attempt+1),
					logx.Field("code", code),
				)
				// Generate new UUIDv7 for retry
				id, _ = uuid.NewV7()
				continue
			}
			logx.WithContext(l.ctx).Errorw("failed to insert URL", logx.Field("error", insertErr.Error()))
			return nil, problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error", "failed to create short URL")
		}

		shortCode = code
		break
	}

	if shortCode == "" {
		return nil, problemdetails.New(500, problemdetails.TypeInternalError, "Internal Error", "failed to generate unique short code after maximum retries")
	}

	return &types.ShortenResponse{
		ShortCode:   shortCode,
		ShortUrl:    l.svcCtx.Config.BaseUrl + "/" + shortCode,
		OriginalUrl: req.OriginalUrl,
	}, nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
