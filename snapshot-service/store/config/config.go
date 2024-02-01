package config

import (
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/snapshot-service/store/cqldb"
	"github.com/plgd-dev/hub/v2/snapshot-service/store/mongodb"
)

type Config = database.Config[*mongodb.Config, *cqldb.Config]
