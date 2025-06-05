package data

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

type PromptTemplate struct {
	ID       int64
	Name     string
	Template string
}

type PromptTemplateDataAccess struct {
	DB       *sql.DB
	InfoLog  *log.Logger
	ErrorLog *log.Logger
}

func (m PromptTemplateDataAccess) Get(id int64) (PromptTemplate, error) {
	if id < 1 {
		return PromptTemplate{}, ErrRecordNotFound
	}

	query := `
		SELECT id, name, template
		FROM templates
 		WHERE id = $1
 		`

	var template PromptTemplate

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.Name,
		&template.Template)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return PromptTemplate{}, ErrRecordNotFound
		default:
			return PromptTemplate{}, err
		}
	}

	return template, nil
}

func (m PromptTemplateDataAccess) Insert(template *PromptTemplate) error {
	query := `
		INSERT INTO templates (name, template) 
		VALUES ($1, $2) 
		RETURNING id
		`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{template.Name, template.Template}

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&template.ID)
}
