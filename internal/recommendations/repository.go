package recommendations

import "database/sql"

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Insert(url, title, byline, excerpt, content, siteName string) (int64, bool, error) {
	res, err := r.db.Exec(
		`INSERT INTO recommendations (url, title, byline, excerpt, content, site_name) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(url) DO NOTHING`,
		url, title, byline, excerpt, content, siteName,
	)
	if err != nil {
		return 0, false, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return 0, false, nil // duplicate
	}
	id, _ := res.LastInsertId()
	return id, true, nil
}

func (r *Repository) Delete(id int64) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM recommendations WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

func (r *Repository) All() ([]Recommendation, error) {
	rows, err := r.db.Query(`SELECT id, url, title, byline, excerpt, content, site_name, added_at FROM recommendations ORDER BY added_at DESC`)
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
