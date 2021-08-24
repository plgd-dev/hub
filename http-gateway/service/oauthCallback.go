package service

import (
	"net/http"

	"github.com/plgd-dev/cloud/pkg/log"
)

func convertValue(val []string) interface{} {
	if len(val) == 1 {
		return val[0]
	}
	return val
}

func responseToMap(r *http.Request) map[string]interface{} {
	res := make(map[string]interface{})
	for key, val := range r.URL.Query() {
		res[key] = convertValue(val)
	}
	if err := r.ParseForm(); err == nil {
		for key, val := range r.PostForm {
			res[key] = convertValue(val)
		}
	}
	if err := r.ParseMultipartForm(1024 * 1024); err == nil {
		for key, val := range r.MultipartForm.Value {
			res[key] = convertValue(val)
		}
	}
	return res
}

func oauthCallback(w http.ResponseWriter, r *http.Request) {
	v := responseToMap(r)
	if err := jsonResponseWriter(w, v); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
