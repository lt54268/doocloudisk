package dal

import (
	"github.com/cloudisk/biz/dal/mysql"
	"github.com/cloudisk/biz/dal/query"
)

// Init init dal
func init() {
	mysql.Init() // mysql init
	query.SetDefault(mysql.DB)
}
