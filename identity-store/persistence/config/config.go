package config

import (
	"github.com/plgd-dev/hub/v2/identity-store/persistence/cqldb"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
)

type Config = database.Config[*mongodb.Config, *cqldb.Config]
