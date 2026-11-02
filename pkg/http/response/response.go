package response

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

var ErrEmtpyBody = errors.New("body empty")

func ReadFromBody[T any](re *http.Request) (*T, error) {
	data := new(T)

	decoder := json.NewDecoder(re.Body)
	defer re.Body.Close()

	err := decoder.Decode(data)
	if err != nil {
		if err == io.EOF {
			return nil, ErrEmtpyBody
		}
		slog.Error("error decoding body", slog.Any("error", err))
		return nil, err
	}

	return data, nil
}

func SendError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	errorBody := map[string]any{
		"message": message,
	}
	response, _ := json.Marshal(errorBody)
	w.Write(response)
}
