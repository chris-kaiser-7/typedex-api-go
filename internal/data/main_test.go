package data

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/lib/pq"
)

var testDB *sql.DB
var bookDa BookDataAccess
var promptDa PromptTemplateDataAccess
var openAiDa OpenAiDataAccess
var subtypeDa SubtypeDataAccess

var bookTestId int64
var subtypeTestIds []int64

var dsn string

func TestMain(m *testing.M) {
	var err error

	flag.StringVar(&dsn, "dsn", "", "Test DB connection string")
	flag.Parse()
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to test db: %v", err)
	}
	defer testDB.Close()

	if err := resetSchema(testDB); err != nil {
		log.Fatalf("failed to reset schema: %v", err)
	}

	openAiFile, err := os.OpenFile("openai.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	defer openAiFile.Close()

	subtypeFile, err := os.OpenFile("subtype.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	defer subtypeFile.Close()

	infoLog := log.New(os.Stdout, "", log.LstdFlags)
	errorLog := log.New(os.Stderr, "", log.LstdFlags)
	openAiLog := log.New(openAiFile, "", log.LstdFlags)
	subtypeLog := log.New(subtypeFile, "", log.LstdFlags)

	// Apply migrations or reset schema
	promptDa = PromptTemplateDataAccess{
		DB:       testDB,
		InfoLog:  infoLog,
		ErrorLog: errorLog,
	}
	bookDa = BookDataAccess{
		DB:       testDB,
		InfoLog:  infoLog,
		ErrorLog: errorLog,
	}
	openAiDa = OpenAiDataAccess{
		OpenAIKey: openAiKey,
		InfoLog:   infoLog,
		ErrorLog:  errorLog,
		ApiLog:    openAiLog,
	}
	subtypeDa = SubtypeDataAccess{
		DB:         testDB,
		InfoLog:    infoLog,
		ErrorLog:   errorLog,
		SubtypeLog: subtypeLog,
		BookDa:     &bookDa,
		OpenAiDa:   &openAiDa,
	}

	bookTestId, err = seedTemplateAndBook(testDB)
	if err != nil {
		log.Fatalf("failed to seed template and book data: %v", err)
	}
	subtypeTestIds, err = seedSubtypes(testDB, bookTestId)
	if err != nil {
		log.Fatalf("failed to seed subtype data: %v", err)
	}

	os.Exit(m.Run())
}

func resetSchema(db *sql.DB) error {
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subtypes;
		DROP TABLE IF EXISTS books;
		DROP TABLE IF EXISTS templates;
		
		CREATE TABLE templates (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			template TEXT NOT NULL
		);

		CREATE TABLE books (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMP NOT NULL DEFAULT now(),
			name TEXT NOT NULL,
			fields TEXT[] NOT NULL,
			field_descriptions TEXT[] NOT NULL,
			template_id INTEGER NOT NULL REFERENCES templates(id),
			instructions TEXT NOT NULL,
			version INT NOT NULL DEFAULT 1
		);

		CREATE TABLE subtypes (
			id SERIAL PRIMARY KEY,
			parent BIGINT REFERENCES subtypes(id),
			type_name TEXT NOT NULL,
			prop_values TEXT[] NOT NULL,
			ancestry BIGINT[] NOT NULL,
			children BIGINT[] NOT NULL,
			book_id BIGINT NOT NULL REFERENCES books(id),
			version INT NOT NULL DEFAULT 1
		);
	`)
	return err
}

func seedSubtypes(db *sql.DB, bookID int64) ([]int64, error) {
	query := `
		INSERT INTO subtypes (parent, type_name, prop_values, ancestry, children, book_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id;
	`

	// Root subtype (no parent, no ancestry)
	var rootID int64
	err := db.QueryRow(
		query,
		nil, // parent
		"Orc",
		pq.Array([]string{"Orc", "200", "medium", "caves", "Orcs are green agressive creatures that live in caves.", "humans and stuff", "fire"}),
		pq.Array([]int64{}),
		pq.Array([]int64{}),
		bookID,
	).Scan(&rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert root subtype: %w", err)
	}

	//fields := []string{"race", "health", "size", "habitat", "description"}
	// Child subtype of root
	var childID int64
	err = db.QueryRow(
		query,
		rootID,
		"Goblin",
		pq.Array([]string{"Goblin", "100", "small", "woods", "Goblins are small agressive green creates that live in the woods.", "literaly any type of meat", "fire and water"}),
		pq.Array([]int64{rootID}),
		pq.Array([]int64{}),
		bookID,
	).Scan(&childID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert child subtype: %w", err)
	}

	// Update root's children to include childID
	_, err = db.Exec(`
		UPDATE subtypes SET children = $1 WHERE id = $2
	`, pq.Array([]int64{childID}), rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to update root children: %w", err)
	}

	return []int64{rootID, childID}, nil
}

func seedTemplateAndBook(db *sql.DB) (bookID int64, err error) {
	var templateID int64
	template := "This GPT generates lists of subtypes when given a classification of a creature, a general description of a creature, and a ancestry of a creature. It provides specific examples based on the type and number requested. The subtypes should have a logical connection to its parent and ancestry. The GPT ensures that the subtypes are relevant and well-known, avoiding obscure references unless specified otherwise. It will give concise, accurate lists that are easy to understand. When replying to a prompt, the response should always be in JSON format and contain no other text. The response should be a list of types. Each type should have the following fields: %s. Here is an example of what the structure of the json should look like: "
	// Insert a template
	err = db.QueryRow(`
		INSERT INTO templates (name, template)
		VALUES ($1, $2)
		RETURNING id
	`, "Test Template", template).
		Scan(&templateID)

	if err != nil {
		err = fmt.Errorf("failed to insert template: %w", err)
		return
	}

	// Insert a book using the template
	fields := []string{"race", "health", "size", "habitat", "description", "diet", "weaknesses"}
	descriptions := []string{"The race of the character.", "The amount of base hp points the character has.", "The general size of the character.", "The habitat that the character lives in.", "A general description of the character.", "the type of food they eat", "what the character is weak agaianst"}
	instructions, err := generateInstuctions(template, fields, descriptions) //TODO: replace with hardcoded string to no relay on code for seed data
	if err != nil {
		err = fmt.Errorf("failed to generate instructions: %w", err)
		return
	}

	err = db.QueryRow(`
		INSERT INTO books (name, fields, field_descriptions, template_id, instructions, version)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		"Test Book",
		pq.Array(fields),
		pq.Array(descriptions),
		templateID,
		instructions,
		1,
	).Scan(&bookID)

	if err != nil {
		err = fmt.Errorf("failed to insert book: %w", err)
		return
	}

	return
}
