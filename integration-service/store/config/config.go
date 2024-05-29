package config

import (
	"github.com/plgd-dev/hub/v2/integration-service/store/cqldb"
	"github.com/plgd-dev/hub/v2/integration-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
)

type Config = database.Config[*mongodb.Config, *cqldb.Config]
