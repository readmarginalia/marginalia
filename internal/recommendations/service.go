package recommendations

import (
	"context"
	"fmt"
	"marginalia/internal/common"
	"marginalia/internal/interop/wayback"
	"marginalia/internal/telemetry/logging"
)

type Service struct {
	repo    *Repository
	wayback *wayback.WaybackClient
}

func NewService(repo *Repository, wayback *wayback.WaybackClient) *Service {
	return &Service{repo: repo, wayback: wayback}
}

type CreateOptions struct {
	URL string `json:"url"`
}

const componentName = "recommendations.service"

func (s *Service) Insert(ctx context.Context, options CreateOptions) (Recommendation, error) {
	ctx, span := tracer.Start(ctx, "service.Insert")
	defer span.End()

	logger := logging.WithComponent(ctx, componentName)

	if options.URL == "" {
		logger.ErrorContext(ctx,
			"invalid url",
			"url", options.URL)
		return Recommendation{}, common.ServiceError{Reason: "invalid url", Code: 400}
	}

	article, err := extractFromURL(ctx, options.URL)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to extract article",
			"error", err,
			"url", options.URL)

		return Recommendation{}, common.ServiceError{Reason: "extraction failed: " + err.Error(), Code: 502}
	}

	s.waybackSave(ctx, options.URL)

	rec, inserted, err := s.repo.Insert(ctx, options.URL, article.Title, article.Byline, article.Excerpt, article.Content, article.SiteName)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to insert recommendation",
			"error", err,
			"url", options.URL)

		return Recommendation{}, common.ServiceError{Reason: "failed to insert recommendation", Code: 500}
	}
	if !inserted {
		logger.ErrorContext(ctx,
			"url already exists",
			"url", options.URL)
		return Recommendation{}, common.ServiceError{Reason: "url already exists", Code: 409}
	}

	logger.InfoContext(ctx,
		fmt.Sprintf("added recomemendation [ %s ]", rec.Title),
		"url", options.URL,
		"title", rec.Title)

	return rec, nil
}

func (s *Service) waybackSave(ctx context.Context, url string) {
	go func(ctx context.Context, url string) {
		logger := logging.WithComponent(ctx, componentName)
		if err := s.wayback.RequestSave(context.Background(), url); err != nil {
			logger.Error("wayback save failed", "error", err, "url", url)
		}
	}(ctx, url)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "service.Delete")
	defer span.End()

	logger := logging.WithComponent(ctx, componentName)

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

	logger := logging.WithComponent(ctx, componentName)
	recs, err := s.repo.All(ctx)
	if err != nil {
		logger.ErrorContext(ctx,
			"failed to fetch recommendations",
			"error", err)

		return nil, common.ServiceError{Reason: "failed to fetch recommendations", Code: 500}
	}
	return recs, nil
}
