package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Subtype struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"-"`      // Use the - directive to never export in JSON output
	Parent     *int64    `json:"parent"` //nullable
	TypeName   string    `json:"type_name"`
	PropValues []string  `json:"prop_values"`
	Ancestry   []int64   `json:"ancestry"`
	Children   []int64   `json:"children"`
	Book       Book      `json:"book"`
	Version    int32     `json:"version"` // The version number starts at 1 and is incremented each
}

type SubtypeDataAccess struct {
	DB         *sql.DB
	OpenAiDa   *OpenAiDataAccess
	BookDa     *BookDataAccess
	InfoLog    *log.Logger
	ErrorLog   *log.Logger
	SubtypeLog *log.Logger
}

func (m SubtypeDataAccess) Get(id int64) (Subtype, error) {
	if id < 1 {
		return Subtype{}, ErrRecordNotFound
	}

	query := `
		SELECT id, parent, type_name, prop_values, ancestry, children, book_id, version
		FROM subtypes
 		WHERE id = $1
 		`

	var subtype Subtype
	var book_id int64

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&subtype.ID,
		&subtype.Parent,
		&subtype.TypeName,
		pq.Array(&subtype.PropValues),
		pq.Array(&subtype.Ancestry),
		pq.Array(&subtype.Children),
		&book_id,
		&subtype.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Subtype{}, ErrRecordNotFound
		default:
			return Subtype{}, err
		}
	}

	book, err := m.BookDa.Get(book_id)
	if err != nil {
		return Subtype{}, err
	}
	subtype.Book = book

	return subtype, nil
}

func (m SubtypeDataAccess) Insert(subtype *Subtype) error {
	query := `
		INSERT INTO subtypes (parent, type_name, prop_values, ancestry, children, book_id)
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id 
		`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{subtype.Parent, subtype.TypeName, pq.Array(subtype.PropValues), pq.Array(subtype.Ancestry), pq.Array(subtype.Children), subtype.Book.ID}

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&subtype.ID)
}

func (m SubtypeDataAccess) UpdateChildren(subtype *Subtype) error {
	query := `
		UPDATE subtypes
		SET children = $1, version = version + 1
		WHERE id = $2 AND version = $3
		RETURNING version
		`

	// Create an args slice containing the values for the placeholder parameters.
	args := []any{
		pq.Array(subtype.Children),
		subtype.ID,
		subtype.Version,
	}

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the SQL query. If no matching row could be found, we know the movie version
	// has changed (or the record has been deleted) and we return ErrEditConflict.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&subtype.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func generatePrompt(type_name string, count int, ancestry []string) string {
	if len(ancestry) == 0 {
		return fmt.Sprintf("Please create %d subtypes of %s. Type %s has no ansestry.",
			count,
			type_name,
			type_name)
	}
	return fmt.Sprintf("Please create %d subtypes of %s. Type %s has the ansestry %s",
		count,
		type_name,
		type_name,
		strings.Join(ancestry, ", "))
}

// func assistantRequest() {
// }

func (s SubtypeDataAccess) getSubtypesById(ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	query := `
		SELECT type_name
		FROM subtypes
 		WHERE id IN (%s)
 		`
	idStrs := make([]string, len(ids))

	for i, v := range ids {
		if v < 1 {
			return nil, ErrRecordNotFound
		}
		idStrs[i] = strconv.FormatInt(v, 10)
	}
	query = fmt.Sprintf(query, strings.Join(idStrs, ", "))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	defer func() {
		if err := rows.Close(); err != nil {
			s.ErrorLog.Println(err)
		}
	}()

	var names []string

	for rows.Next() {
		var typeName *string

		err := rows.Scan(&typeName)
		if err != nil {
			return nil, err
		}

		// Add the Movie struct to the slice
		names = append(names, *typeName)
	}
	return names, nil
}

func (s SubtypeDataAccess) generateChildren(parent *Subtype, count int) ([]Subtype, error) {
	typeNames, err := s.getSubtypesById(parent.Ancestry)
	if err != nil {
		return nil, err
	}
	//execute openai logic

	//TODO: have pool of assistants to keep and check
	assistant, err := s.OpenAiDa.createAssistant(parent.Book.Name, parent.Book.Instructions)
	if err != nil {
		return nil, err
	}
	thread, err := s.OpenAiDa.createThread()
	if err != nil {
		return nil, err
	}
	prompt := generatePrompt(parent.TypeName, count, typeNames)
	_, err = s.OpenAiDa.AddThreadMessage(thread, "user", prompt)
	if err != nil {
		return nil, err

	}
	_, err = s.OpenAiDa.RunAssistant(thread, assistant, "")
	if err != nil {
		return nil, err
	}

	mesgList, err := pollThread(thread, s.OpenAiDa)
	if err != nil {
		return nil, err
	}

	if mesg := mesgList.Error.Message; mesg != "" {
		s.SubtypeLog.Printf("error: %v", mesg)
		return nil, fmt.Errorf("%s : %v", mesg, OpenAiError)
	}
	if len(mesgList.Data) == 0 || len(mesgList.Data[0].Content) == 0 {
		s.SubtypeLog.Printf("error empty mesg list: %v", mesgList)
		return nil, fmt.Errorf("empty mesg list.")
	}

	childrenString := mesgList.Data[0].Content[0].Text.Value
	s.SubtypeLog.Printf("childrenString: %v", childrenString)
	//ind := strings.Index(childrenString, "{")
	r := len(childrenString) - 3

	var children []map[string]string

	err = json.Unmarshal([]byte(childrenString[7:r]), &children)

	fieldCount := len(parent.Book.Fields)
	result := make([]Subtype, len(children))

	for i := range len(children) {
		values := make([]string, fieldCount, fieldCount)
		for j, field := range parent.Book.Fields {
			values[j] = children[i][field]
		}
		childAncestry := make([]int64, len(parent.Ancestry))
		copy(childAncestry, parent.Ancestry)
		childAncestry = append(childAncestry, parent.ID)
		nextChild := Subtype{
			Parent:     &parent.ID,
			TypeName:   children[i]["name"],
			PropValues: values,
			Ancestry:   childAncestry,
			Children:   make([]int64, 0, 10), //optimization if 10 is the max children
			Book:       parent.Book,
		}
		if err := s.Insert(&nextChild); err != nil {
			return nil, err
		}
		parent.Children = append(parent.Children, nextChild.ID)
		result[i] = nextChild
	}
	if err := s.UpdateChildren(parent); err != nil {
		return nil, err
	}

	return result, nil
}

func pollThread(thread Thread, openAiDA *OpenAiDataAccess) (MessageList, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	quit := make(chan struct{})
	timeout := make(chan struct{})
	time.AfterFunc(30*time.Second, func() { close(timeout) })

	var err error
	var mesgList MessageList

	go func() {
		for {
			select {
			case <-ticker.C:
				mesgList, err = openAiDA.RefreshThread(thread)
				if len(mesgList.Data[0].Content) > 0 || mesgList.Error.Message != "" {
					quit <- struct{}{}
				}
			case <-timeout:
				return
			case <-quit:
				return
			}
		}
	}()

	select {
	case <-timeout:
		return MessageList{}, fmt.Errorf("Timed out pulling thread")
	case <-quit:
	}

	mesgList, err = openAiDA.RefreshThread(thread)
	if err != nil {
		return MessageList{}, err
	}

	return mesgList, nil
}
