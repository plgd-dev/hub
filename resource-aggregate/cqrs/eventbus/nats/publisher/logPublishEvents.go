package publisher

import (
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
)

func LogPublish(logger log.Logger, event interface{}, subjects []string, err error) {
	lvl := log.DebugLevel
	if err != nil {
		lvl = log.ErrorLevel
	}
	if !logger.Check(lvl) {
		return
	}
	v := struct {
		Subjects []string    `json:"subjects,omitempty"`
		Body     interface{} `json:"body,omitempty"`
		Error    string      `json:"error,omitempty"`
	}{}
	v.Subjects = subjects
	if logger.Config().DumpBody {
		v.Body = server.DecodeToJsonObject(event)
	}
	if err != nil {
		v.Error = err.Error()
	}
	logger = logger.With(log.ProtocolKey, "NATS", log.MessageKey, v)
	logger = server.FillLoggerWithDeviceIDHrefCorrelationID(logger, event)
	logger.GetLogFunc(lvl)("published event message")
}
