package service

import (
	"fmt"

	"github.com/cloudisk/biz/dal/query"
	"github.com/cloudisk/biz/model/common"
)

func Upload(user_id int, pid int, webkitRelativePath string, overwrite bool) *common.File {
	if pid > 0 {
		// if query.File.First() {

		// }
		fmt.Print(query.Q.File.First())
	}
	resp := &common.File{}

	return resp
}
