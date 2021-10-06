package service

import (
	netHttp "net/http"

	"github.com/plgd-dev/cloud/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/json"
)

func errToJsonRes(err error) map[string]string {
	return map[string]string{"err": err.Error()}
}

func writeError(w netHttp.ResponseWriter, err error, status int) {
	if err == nil {
		w.WriteHeader(netHttp.StatusNoContent)
		return
	}
	log.Errorf("%v", err)
	b, _ := json.Encode(errToJsonRes(err))
	netHttp.Error(w, string(b), status)
}
