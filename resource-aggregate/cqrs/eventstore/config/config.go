package config

import (
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
)

type Config = database.Config[*mongodb.Config, *cqldb.Config]
