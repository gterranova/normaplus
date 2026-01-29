package store

import "context"

type Annotation struct {
	ID              int    `json:"id"`
	UserID          int    `json:"user_id"`
	DocID           string `json:"doc_id"`
	SelectionData   string `json:"selection_data"`
	LocationID      string `json:"location_id"`
	SelectionOffset int    `json:"selection_offset"`
	Prefix          string `json:"prefix"`
	Suffix          string `json:"suffix"`
	Comment         string `json:"comment"`
	CreatedAt       string `json:"created_at"`
}

func (s *Store) CreateAnnotation(ctx context.Context, userID int, docID, selectionData, locationID, prefix, suffix string, offset int, comment string) (*Annotation, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO annotations (user_id, doc_id, selection_data, location_id, prefix, suffix, selection_offset, comment) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		userID, docID, selectionData, locationID, prefix, suffix, offset, comment)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Annotation{
		ID:              int(id),
		UserID:          userID,
		DocID:           docID,
		SelectionData:   selectionData,
		LocationID:      locationID,
		Prefix:          prefix,
		Suffix:          suffix,
		SelectionOffset: offset,
		Comment:         comment,
	}, nil
}

func (s *Store) ListAnnotations(ctx context.Context, userID int, docID string) ([]Annotation, error) {
	// If docID is empty, list all for user? Usually we list per doc or global.
	// Let's support both.
	query := "SELECT id, user_id, doc_id, selection_data, location_id, prefix, suffix, selection_offset, comment, created_at FROM annotations WHERE user_id = ?"
	args := []interface{}{userID}

	if docID != "" {
		query += " AND doc_id = ?"
		args = append(args, docID)
	}
	query += " ORDER BY created_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var annotations []Annotation
	for rows.Next() {
		var a Annotation
		if err := rows.Scan(&a.ID, &a.UserID, &a.DocID, &a.SelectionData, &a.LocationID, &a.Prefix, &a.Suffix, &a.SelectionOffset, &a.Comment, &a.CreatedAt); err != nil {
			return nil, err
		}
		annotations = append(annotations, a)
	}
	return annotations, nil
}

func (s *Store) DeleteAnnotation(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM annotations WHERE id = ?", id)
	return err
}

func (s *Store) UpdateAnnotation(ctx context.Context, id int, comment string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE annotations SET comment = ? WHERE id = ?", comment, id)
	return err
}
