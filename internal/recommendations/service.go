package recommendations

import (
	"log"
	"marginalia/internal/common"
	"marginalia/internal/interop/wayback"
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

func (s *Service) Insert(options CreateOptions) (*Recommendation, error) {
	if options.URL == "" {
		return nil, common.ServiceError{Reason: "invalid url", Code: 400}
	}

	article, err := extractFromURL(options.URL)
	if err != nil {
		return nil, common.ServiceError{Reason: "extraction failed: " + err.Error(), Code: 502}
	}

	go func() {
		if err := s.wayback.RequestSave(options.URL); err != nil {
			log.Printf("wayback save failed for %s: %v", options.URL, err)
		}
	}()

	rec, inserted, err := s.repo.Insert(options.URL, article.Title, article.Byline, article.Excerpt, article.Content, article.SiteName)
	if err != nil {
		log.Printf("failed to insert recommendation: %v", err)
		return nil, common.ServiceError{Reason: "failed to insert recommendation", Code: 500}
	}
	if !inserted {
		return nil, common.ServiceError{Reason: "url already exists", Code: 409}
	}

	return rec, nil
}

func (s *Service) Delete(id int64) error {
	found, err := s.repo.Delete(id)
	if err != nil {
		log.Printf("failed to delete recommendation: %v", err)
		return common.ServiceError{Reason: "failed to delete recommendation", Code: 500}
	}
	if !found {
		return common.ServiceError{Reason: "not found", Code: 404}
	}
	return nil
}

func (s *Service) All() ([]Recommendation, error) {
	recs, err := s.repo.All()
	if err != nil {
		log.Printf("failed to fetch recommendations: %v", err)
		return nil, common.ServiceError{Reason: "failed to fetch recommendations", Code: 500}
	}
	return recs, nil
}
