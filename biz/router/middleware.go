package router

import (
	"context"

	"github.com/cloudisk/pkg/cloud"
	"github.com/cloudwego/hertz/pkg/app"
)

func CloudProviderMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		provider := c.Query("provider") // 从请求中获取云服务厂商标识
		if provider == "" {
			provider = "aliyun" // 设置默认的云服务厂商为阿里云
		}
		storage, err := cloud.GetCloudStorageProvider(provider)
		if err != nil {
			c.String(400, "Unsupported cloud provider")
			c.Abort()
			return
		}
		c.Set("cloudStorage", storage) // 将云存储实例放入上下文中
		c.Next(ctx)                    // 继续执行下一个中间件或请求处理函数
	}
}
