package models

import "encoding/json"

// TODO: respond with error messages

type ErrorPayload struct {
	Message string `json:"message"`
}

func CreateError(msg string) []byte {
	err, _ := json.Marshal(ErrorPayload{
		Message: msg,
	})
	return err
}
