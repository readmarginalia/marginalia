package recommendations

import (
	"marginalia/internal/common"
	"marginalia/internal/extract"
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

func (s *Service) Insert(options *CreateOptions) (int64, error) {
	if options.URL == "" {
		return 0, &common.ServiceError{Reason: "invalid url", Code: 400}
	}

	article, err := extract.FromURL(options.URL)
	if err != nil {
		return 0, &common.ServiceError{Reason: "extraction failed: " + err.Error(), Code: 502}
	}

	id, inserted, err := s.repo.Insert(options.URL, article.Title, article.Byline, article.Excerpt, article.Content, article.SiteName)
	if err != nil {
		return 0, &common.ServiceError{Reason: "db error: " + err.Error(), Code: 500}
	}
	if !inserted {
		return 0, &common.ServiceError{Reason: "url already exists", Code: 409}
	}

	return id, nil
}

func (s *Service) Delete(id int64) error {
	found, err := s.repo.Delete(id)
	if err != nil {
		return &common.ServiceError{Reason: "db error: " + err.Error(), Code: 500}
	}
	if !found {
		return &common.ServiceError{Reason: "not found", Code: 404}
	}
	return nil
}

func (s *Service) All() ([]Recommendation, error) {
	recs, err := s.repo.All()
	if err != nil {
		return nil, &common.ServiceError{Reason: "db error: " + err.Error(), Code: 500}
	}
	return recs, nil
}
