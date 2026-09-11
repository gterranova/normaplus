package store

import "context"

type Bookmark struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	DocID     string `json:"doc_id"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	Category  string `json:"category"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

// normalizeSource keeps an empty value meaning what it has always meant.
//
// A bookmark written before EUR-Lex existed carries no source, and every one of
// them is Italian legislation — so the absent value is not "unknown", it is
// normattiva, and saying so here stops every reader having to decide.
func normalizeSource(s string) string {
	if s == "" {
		return "normattiva"
	}
	return s
}

func (s *Store) CreateBookmark(ctx context.Context, userID int, docID, title, date, source string) (*Bookmark, error) {
	source = normalizeSource(source)
	// The source is updated on conflict as well: a document bookmarked before
	// the column existed would otherwise keep answering "normattiva" for ever,
	// with no way to correct it short of deleting the bookmark.
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO bookmarks (user_id, doc_id, title, date, category, source) VALUES (?, ?, ?, ?, 'General', ?) ON CONFLICT(user_id, doc_id) DO UPDATE SET created_at = CURRENT_TIMESTAMP, source = excluded.source",
		userID, docID, title, date, source)

	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Bookmark{
		ID:       int(id),
		UserID:   userID,
		DocID:    docID,
		Title:    title,
		Date:     date,
		Category: "General",
		Source:   source,
	}, nil
}

func (s *Store) UpdateBookmarkCategory(ctx context.Context, userID int, docID, category string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE bookmarks SET category = ? WHERE user_id = ? AND doc_id = ?", category, userID, docID)
	return err
}

func (s *Store) ListBookmarks(ctx context.Context, userID int) ([]Bookmark, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, user_id, doc_id, title, date, category, COALESCE(source, 'normattiva'), created_at FROM bookmarks WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.ID, &b.UserID, &b.DocID, &b.Title, &b.Date, &b.Category, &b.Source, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.Source = normalizeSource(b.Source)
		bookmarks = append(bookmarks, b)
	}
	return bookmarks, nil
}

func (s *Store) DeleteBookmark(ctx context.Context, userID int, docID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM bookmarks WHERE user_id = ? AND doc_id = ?", userID, docID)
	return err
}
