package service

import "net/http"

func (requestHandler *RequestHandler) getUserInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"sub": deviceUserID,
	}
	jsonResponseWriter(w, resp)
}
