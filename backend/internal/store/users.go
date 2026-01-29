package store

import "context"

type User struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	Theme      string `json:"theme"`
	UILanguage string `json:"ui_language"`
	Mode       string `json:"mode"`
	UIState    string `json:"ui_state"` // JSON blob
	CreatedAt  string `json:"created_at"`
}

func (s *Store) CreateUser(ctx context.Context, name, color string) (*User, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO users (name, color, theme, ui_language, mode, ui_state) VALUES (?, ?, 'default', 'it', 'light', '{}')", name, color)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &User{
		ID:         int(id),
		Name:       name,
		Color:      color,
		Theme:      "default",
		UILanguage: "it",
		Mode:       "light",
		UIState:    "{}",
	}, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, color, theme, ui_language, mode, ui_state, created_at FROM users ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Color, &u.Theme, &u.UILanguage, &u.Mode, &u.UIState, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) GetUser(ctx context.Context, id int) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, "SELECT id, name, color, theme, ui_language, mode, ui_state, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Name, &u.Color, &u.Theme, &u.UILanguage, &u.Mode, &u.UIState, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET name = ?, color = ?, theme = ?, ui_language = ?, mode = ?, ui_state = ? WHERE id = ?",
		u.Name, u.Color, u.Theme, u.UILanguage, u.Mode, u.UIState, u.ID)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}
