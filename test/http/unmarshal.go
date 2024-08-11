package http

import (
	"encoding/json"
	"io"
	"net/http"

	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
)

func UnmarshalJson(code int, input io.Reader, v any) error {
	var data json.RawMessage
	err := json.NewDecoder(input).Decode(&data)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return pkgHttpPb.UnmarshalError(data)
	}
	err = json.Unmarshal(data, v)
	return err
}
