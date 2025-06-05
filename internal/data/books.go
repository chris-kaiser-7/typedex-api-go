package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Book struct {
	ID                int64
	CreatedAt         time.Time `json:"-"` // Use the - directive to never export in JSON output
	Name              string
	Fields            []string
	FieldDescriptions []string
	Template          PromptTemplate
	Instructions      string
	Version           int32 `json:"version"` // The version number starts at 1 and is incremented each
}

type BookDataAccess struct {
	DB       *sql.DB
	InfoLog  *log.Logger
	ErrorLog *log.Logger
}

func (m BookDataAccess) Get(id int64) (Book, error) {
	if id < 1 {
		return Book{}, ErrRecordNotFound
	}

	query := `
		SELECT b.id, b.name, b.fields, b.field_descriptions, b.instructions, b.version, b.template_id, t.name, t.template
		FROM books b
		INNER JOIN templates t ON b.template_id = t.id
 		WHERE b.id = $1
 		`

	var book Book

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
		&book.Name,
		pq.Array(&book.Fields),
		pq.Array(&book.FieldDescriptions),
		&book.Instructions,
		&book.Version,
		&book.Template.ID,
		&book.Template.Name,
		&book.Template.Template,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Book{}, ErrRecordNotFound
		default:
			return Book{}, err
		}
	}

	return book, nil
}

func (m BookDataAccess) Insert(book *Book) error {
	query := `
		INSERT INTO books (name, fields, field_descriptions, template_id, instructions)
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id, created_at, version
		`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	instructions, err := generateInstuctions(book.Template.Template, book.Fields, book.FieldDescriptions)
	if err != nil {
		return err
	}

	args := []any{book.Name, pq.Array(book.Fields), pq.Array(book.FieldDescriptions), book.Template.ID, instructions}

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&book.ID, &book.CreatedAt, &book.Version)
}

func generateInstuctions(template string, fields []string, field_descriptions []string) (string, error) {
	if len(fields) != len(field_descriptions) {
		return "", errors.New("field length mismatch.")
	}
	var builder strings.Builder
	insert_string := "[" + strings.Join(fields, ",") + "]"
	builder.WriteString(fmt.Sprintf(template, insert_string))
	builder.WriteString("```json{\\\"name\\\":\\\"the name\\\",")
	comma := ","
	for i := range fields {
		if i == len(fields)-1 { // the last iteration can't have a comma
			comma = ""
		}
		row := fmt.Sprintf("\\\"%s\\\": \\\"%s\\\"%s", fields[i], field_descriptions[i], comma)
		builder.WriteString(row)
	}
	builder.WriteString("}```")
	return builder.String(), nil
}
