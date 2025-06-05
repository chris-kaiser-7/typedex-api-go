package data

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestBookModel_InsertAndGet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Insert template
	var templateID int
	err := testDB.QueryRowContext(ctx, `
		INSERT INTO templates (name, template)
		VALUES ('Test Template', 'Prompt for fields: %s')
		RETURNING id
	`).Scan(&templateID)
	if err != nil {
		t.Fatalf("failed to insert template: %v", err)
	}

	book := &Book{
		Name:              "Test Book",
		Fields:            []string{"field1", "field2"},
		FieldDescriptions: []string{"desc1", "desc2"},
		Template: PromptTemplate{
			ID:       int64(templateID),
			Name:     "Test Template",
			Template: "Prompt for fields: %s",
		},
	}

	// Insert book
	err = bookDa.Insert(book)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if book.ID == 0 {
		t.Fatal("expected book ID to be set")
	}

	// Get book
	got, err := bookDa.Get(book.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != book.Name {
		t.Errorf("expected name %q, got %q", book.Name, got.Name)
	}
	if len(got.Fields) != 2 || got.Fields[0] != "field1" {
		t.Errorf("unexpected fields: %#v", got.Fields)
	}
}

func isValidJSON(data string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(data), &js) == nil
}

func TestGenerateInstructions(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		template := "Generate a record with the fields %s and the schema: "
		fields := []string{"title", "year"}
		descriptions := []string{"The movie title", "The release year"}

		result, err := generateInstuctions(template, fields, descriptions)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if !strings.Contains(result, "Generate a record with the fields [title,year]") {
			t.Fatalf("expected formatted template to be present, got:\n%s", result)
		}

		re, err := regexp.Compile("(?s)```json\\s*(.*?)\\s*```")
		if err != nil {
			t.Fatalf("Can't compile json regexp: %v", err)
		}
		matches := re.FindStringSubmatch(result)
		if len(matches) < 2 {
			t.Fatalf("Did not find matching JSON format in: %v", matches)
		}
		if !isValidJSON(strings.ReplaceAll(matches[1], "\\", "")) {
			t.Fatal("JSON could not be unmarshaled: ", matches[1])
		}
	})
	t.Run("mismatched field lengths", func(t *testing.T) {
		template := "Anything"
		fields := []string{"field1"}
		descriptions := []string{"desc1", "desc2"}

		_, err := generateInstuctions(template, fields, descriptions)
		if err == nil {
			t.Fatalf("expected error due to mismatched lengths, got nil")
		}
	})
}
