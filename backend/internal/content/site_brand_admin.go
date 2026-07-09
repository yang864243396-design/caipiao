package content

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type SiteBrand struct {
	SiteName string `json:"siteName"`
	LogoURL  string `json:"logoUrl"`
	Tagline  string `json:"tagline"`
}

func (s *Service) PublicSiteBrand(ctx context.Context) (SiteBrand, error) {
	if s == nil || s.q == nil {
		return SiteBrand{}, ErrUnavailable
	}
	row, err := s.q.GetSiteBrandPublic(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSiteBrand(), nil
		}
		return SiteBrand{}, err
	}
	return mapSiteBrandRow(row.SiteName, row.LogoUrl, row.Tagline), nil
}

func defaultSiteBrand() SiteBrand {
	return SiteBrand{
		SiteName: "精密终端 · 演示站",
		LogoURL:  "https://placehold.co/120x40/0066ff/ffffff?text=LOGO",
		Tagline:  "数字精算主义 · 管理端 Mock",
	}
}

func mapSiteBrandRow(siteName, logoURL, tagline string) SiteBrand {
	return SiteBrand{
		SiteName: siteName,
		LogoURL:  logoURL,
		Tagline:  tagline,
	}
}
