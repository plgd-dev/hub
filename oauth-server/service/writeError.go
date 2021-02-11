package service

import (
	netHttp "net/http"

	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/http"
)

func errToJsonRes(err error) map[string]string {
	return map[string]string{"err": err.Error()}
}

func writeError(w netHttp.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(netHttp.StatusNoContent)
		return
	}
	log.Errorf("%v", err)
	b, _ := json.Encode(errToJsonRes(err))
	netHttp.Error(w, string(b), http.ErrToStatus(err))
}
