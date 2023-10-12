package config

import (
	"github.com/plgd-dev/hub/v2/certificate-authority/store/cqldb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
)

type Config = database.Config[*mongodb.Config, *cqldb.Config]
