package router

import (
	aliyun "github.com/cloudisk/biz/router/aliyun"
	"github.com/cloudwego/hertz/pkg/app/server"
	// tencent "github.com/cloudisk/biz/router/tencent"
	// qiniu "github.com/cloudisk/biz/router/qiniu"
)

// CloudProvider defines the interface for cloud providers.
type CloudProvider interface {
	Register(r *server.Hertz)
}

// AliyunProvider implements CloudProvider for Aliyun.
type AliyunProvider struct{}

func (a *AliyunProvider) Register(r *server.Hertz) {
	aliyun.Register(r)
}

// Add similar structs for Tencent and Qiniu
// type TencentProvider struct{}
// func (t *TencentProvider) Register(r *server.Hertz) {
//     tencent.Register(r)
// }

// type QiniuProvider struct{}
// func (q *QiniuProvider) Register(r *server.Hertz) {
//     qiniu.Register(r)
// }

// CloudFactory returns the appropriate CloudProvider based on the input.
func CloudFactory(provider string) CloudProvider {
	switch provider {
	case "aliyun":
		return &AliyunProvider{}
	// case "tencent":
	//     return &TencentProvider{}
	// case "qiniu":
	//     return &QiniuProvider{}
	default:
		return nil
	}
}
