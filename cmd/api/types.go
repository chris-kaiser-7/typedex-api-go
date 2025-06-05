package main

import (
	"fmt"
	"net/http"

	"github.com/codeaucafe/snippetbox/greenlight/internal/data"
	"github.com/codeaucafe/snippetbox/greenlight/internal/validator"
)

func (app *application) createType(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type  string `json:"type"`
		Count int32  `json:"count"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Call the Insert() method on our movies model, passing in a pointer to the validated movie
	// struct. This will create a record in the database and update the movie struct with the
	// system-generated information.
	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// When sending an HTTP response,
	// we want to include a Location header to let the client know which URL they can find the
	// newly created resource at. We make an empty http.Header map and then use the Set()
	// method to add a new Location header,
	// interpolating the system-generated ID for our new movie in the URL.
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	// Write a JSON response with a 201 Created status code, the movie data in the response body,
	// and the Location header.
	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
