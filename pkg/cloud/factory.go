// pkg/cloud/factory.go
package cloud

import (
	"context"
	"fmt"

	"github.com/cloudisk/biz/handler/aliyun"
	_ "github.com/cloudisk/biz/handler/qiniuyun"
	_ "github.com/cloudisk/biz/handler/txyun"
	"github.com/cloudwego/hertz/pkg/app"
)

type CloudStorage interface {
	Upload(ctx context.Context, c *app.RequestContext)
	OfficeUpload(ctx context.Context, c *app.RequestContext)
	Save(ctx context.Context, c *app.RequestContext)
	Download(ctx context.Context, c *app.RequestContext)
	Remove(ctx context.Context, c *app.RequestContext)
}

func GetCloudStorageProvider(provider string) (CloudStorage, error) {
	switch provider {
	case "aliyun":
		return &aliyun.AliyunOSS{}, nil
	case "tencent":
		//return &txyun.TencentCOS{}, nil
		return nil, fmt.Errorf("unsupported cloud provider: %s", provider)
	case "qiniu":
		//return &qiniuyun.QiniuCloud{}, nil
		return nil, fmt.Errorf("unsupported cloud provider: %s", provider)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", provider)
	}
}
