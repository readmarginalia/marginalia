package recommendations

import (
	"database/sql"
	"errors"
	"marginalia/internal/common"
	"marginalia/internal/infra/db"
	"marginalia/internal/interop/wayback"
	"path"
	"testing"
	"time"
)

const (
	dbPath = "data/marginalia.db"
	testUrl = "https://thevaluable.dev/vim-advanced/"
	testUrl2 = "https://ieftimov.com/posts/testing-in-go-naming-conventions/"
)

func initializeDb(t *testing.T) *sql.DB {
	dbPath := path.Join(t.TempDir(), dbPath)

	db, err := db.Open(dbPath)
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func initializeService(t *testing.T) *Service {
	waybackClient, err := wayback.NewClient("https://web.archive.org", 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create wayback client: %v", err)
	}
	return NewService(NewRepository(initializeDb(t)), waybackClient)
}

func TestInsertBadRequestEmptyURL(t *testing.T) {
	const expectedStatusCode = 400

	service := initializeService(t)
	_, err := service.Insert(CreateOptions{})
	if err == nil {
		t.Fatal("Expected error on empty URL")
	}

	var svcErr common.ServiceError
	if !errors.As(err, &svcErr) {
		t.Fatal("Expected ServiceError")
	}
	if svcErr.Code != expectedStatusCode {
		t.Fatalf("Expected %d on empty URL, got %d - %s", expectedStatusCode, svcErr.Code, svcErr.Reason)
	}
}

func TestInsertConflictURLExists(t *testing.T) {
	const expectedStatusCode = 409

	service := initializeService(t)
	_, err := service.Insert(CreateOptions{URL: testUrl})
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = service.Insert(CreateOptions{URL: testUrl})

	var svcErr common.ServiceError
	if !errors.As(err, &svcErr) {
		t.Fatal("Expected ServiceError")
	}
	if svcErr.Code != expectedStatusCode {
		t.Fatalf("Expected %d on URL exists, got %d - " + svcErr.Reason, expectedStatusCode, svcErr.Code)
	}
}

func TestDelete(t *testing.T) {
	service := initializeService(t)
	r, err := service.Insert(CreateOptions{URL: testUrl})
	if err != nil {
		t.Fatal(err.Error())
	}

	rs, err := service.All()
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(rs) != 1 {
		t.Fatalf("Expected 1, found %d", len(rs))
	}

	service.Delete(r.ID)
}
