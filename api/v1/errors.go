package v1

import (
	"encoding/json"
	"fmt"
	"github.com/manyminds/api2go"
	"net/http"
	"strconv"
)

type Error api2go.Error

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Title, e.Detail)
}

type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

func apiError(rw http.ResponseWriter, title, detail string, status int) {
	doc := ErrorResponse{
		Errors: []Error{
			Error{
				Status: strconv.Itoa(status),
				Title:  title,
				Detail: detail,
			},
		},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	rw.WriteHeader(status)
	_, _ = rw.Write(data)
}
