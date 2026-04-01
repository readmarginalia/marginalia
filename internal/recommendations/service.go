package recommendations

import (
	"context"
	"marginalia/internal/common"
	"marginalia/internal/telemetry/logging"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateOptions struct {
	URL string `json:"url"`
}

func (s *Service) Insert(ctx context.Context, options CreateOptions) (*Recommendation, error) {
	ctx, span := tracer.Start(ctx, "service.Insert")
	defer span.End()

	logger := logging.FromContext(ctx)

	if options.URL == "" {
		logger.ErrorContext(ctx,
			"invalid url",
			"url", options.URL)
		return nil, common.ServiceError{Reason: "invalid url", Code: 400}
	}

	article, err := extractFromURL(ctx, options.URL)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to extract article",
			"error", err,
			"url", options.URL)

		return nil, common.ServiceError{Reason: "extraction failed: " + err.Error(), Code: 502}
	}

	rec, inserted, err := s.repo.Insert(ctx, options.URL, article.Title, article.Byline, article.Excerpt, article.Content, article.SiteName)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to insert recommendation",
			"error", err,
			"url", options.URL)

		return nil, common.ServiceError{Reason: "failed to insert recommendation", Code: 500}
	}
	if !inserted {
		logger.ErrorContext(ctx,
			"url already exists",
			"url", options.URL)
		return nil, common.ServiceError{Reason: "url already exists", Code: 409}
	}

	logger.InfoContext(ctx,
		"added recommendation",
		"url", options.URL,
		"title", rec.Title)

	return rec, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "service.Delete")
	defer span.End()

	logger := logging.FromContext(ctx)
	found, err := s.repo.Delete(ctx, id)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to delete recommendation",
			"error", err,
			"recommendation_id", id)
		return common.ServiceError{Reason: "failed to delete recommendation", Code: 500}
	}
	if !found {
		logger.InfoContext(ctx,
			"recommendation not found",
			"recommendation_id", id)
		return common.ServiceError{Reason: "not found", Code: 404}
	}
	logger.InfoContext(ctx,
		"deleted recommendation",
		"recommendation_id", id)
	return nil
}

func (s *Service) All(ctx context.Context) ([]Recommendation, error) {
	ctx, span := tracer.Start(ctx, "service.All")
	defer span.End()

	logger := logging.FromContext(ctx)
	recs, err := s.repo.All(ctx)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to fetch recommendations",
			"error", err)

		return nil, common.ServiceError{Reason: "failed to fetch recommendations", Code: 500}
	}
	return recs, nil
}
