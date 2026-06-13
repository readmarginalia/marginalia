package recommendations

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Insert(ctx context.Context, url, title, byline, excerpt, content, siteName string) (Recommendation, bool, error) {
	rec := Recommendation{}

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO recommendations (url, title, byline, excerpt, content, site_name)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO NOTHING
		RETURNING id, url, title, byline, excerpt, content, site_name
	`,
		url, title, byline, excerpt, content, siteName,
	).Scan(
		&rec.ID,
		&rec.URL,
		&rec.Title,
		&rec.Byline,
		&rec.Excerpt,
		&rec.Content,
		&rec.SiteName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Recommendation{}, false, nil
		}
		return Recommendation{}, false, err
	}

	return rec, true, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM recommendations WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

func (r *Repository) All(ctx context.Context) ([]Recommendation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, url, title, byline, excerpt, content, site_name, added_at FROM recommendations ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []Recommendation
	for rows.Next() {
		var r Recommendation
		if err := rows.Scan(&r.ID, &r.URL, &r.Title, &r.Byline, &r.Excerpt, &r.Content, &r.SiteName, &r.AddedAt); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}
