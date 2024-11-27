// Code generated by hertz generator. DO NOT EDIT.

package router

import (
	aliyun "github.com/cloudisk/biz/router/aliyun"
	qiniu "github.com/cloudisk/biz/router/qiniu"
	tencent "github.com/cloudisk/biz/router/tencent"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// GeneratedRegister registers routers generated by IDL.
func GeneratedRegister(r *server.Hertz) {
	//INSERT_POINT: DO NOT DELETE THIS LINE!
	qiniu.Register(r)

	tencent.Register(r)

	aliyun.Register(r)
}
