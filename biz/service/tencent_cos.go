package service

import (
	"context"
	//"dootxcos/internal/model"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type CosUploader struct {
	client *cos.Client
}

type CosDownloader struct {
	client *cos.Client
}

type CosDeleter struct {
	client *cos.Client
}

type CosLister struct {
	client *cos.Client
}

type CosCopier struct {
	client *cos.Client
}

func NewCosUploader() *CosUploader {
	return &CosUploader{
		client: NewCosClient(),
	}
}

func NewCosDownloader() *CosDownloader {
	return &CosDownloader{
		client: NewCosClient(),
	}
}

func NewCosDeleter() *CosDeleter {
	return &CosDeleter{
		client: NewCosClient(),
	}
}

func NewCosLister() *CosLister {
	return &CosLister{
		client: NewCosClient(),
	}
}

func NewCosCopier() *CosCopier {
	return &CosCopier{
		client: NewCosClient(),
	}
}

func NewCosClient() *cos.Client {
	// 创建一个通用的 COS 客户端
	u, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", os.Getenv("COS_BUCKET"), os.Getenv("COS_REGION")))
	b := &cos.BaseURL{BucketURL: u}

	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("SECRETID"),
			SecretKey: os.Getenv("SECRETKEY"),
		},
	})

	return client
}

// Upload 上传文件到腾讯云 COS
func (u *CosUploader) Upload(fileData multipart.File, objectName string) (*cos.Response, error) {
	// 上传文件流
	// resp, err := u.client.Object.Put(context.Background(), objectName, fileData, nil)
	_, err := u.client.Object.Put(context.Background(), objectName, fileData, nil)
	if err != nil {
		return nil, err
	}

	// 调用 List 方法获取刚上传的文件信息
	// lister := NewCosLister()                           // 创建 lister 实例
	// fileList, _, err := lister.List(objectName, "", 0) // 传递 objectName 作为 prefix 参数
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to retrieve uploaded file info: %v", err)
	// }

	// 查找匹配的文件信息
	// for _, file := range fileList {
	// 	if file.Key == objectName {
	// 		return resp, nil
	// 	}
	// }

	return nil, fmt.Errorf("file not found after upload")
}

// Download 从 COS 下载文件
func (d *CosDownloader) Download(objectName string) ([]byte, error) {
	resp, err := d.client.Object.Get(context.Background(), objectName, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Delete 方法删除指定的对象
func (d *CosDeleter) Delete(objectName string) error {
	_, err := d.client.Object.Delete(context.Background(), objectName, nil)
	if err != nil {
		if cos.IsNotFoundError(err) {
			return fmt.Errorf("resource not found: %v", objectName)
		}

		if e, ok := cos.IsCOSError(err); ok {
			if e.Code == "AccessDenied" {
				return fmt.Errorf("access denied. Please check COS permissions for DeleteObject operation")
			}
			return fmt.Errorf("COS error - Code: %v, Message: %v, Resource: %v, RequestId: %v", e.Code, e.Message, e.Resource, e.RequestID)
		}
	}
	return nil
}

// List 获取 COS 文件列表，格式化输出文件信息
/*
	func (l *CosLister) List(prefix, marker string, maxKeys int) ([]model.FileInfo, string, error) {
		if prefix == "" {
			prefix = "" // 默认 Prefix 为 *，返回所有对象
		}
		if maxKeys == 0 {
			maxKeys = 1000 // 默认 MaxKeys 为 1000
		}

		opt := &cos.BucketGetOptions{
			Prefix:  prefix,
			Marker:  marker,
			MaxKeys: maxKeys,
		}

		v, _, err := l.client.Bucket.Get(context.Background(), opt)
		if err != nil {
			return nil, "", err
		}

		var fileList []model.FileInfo
		for _, content := range v.Contents {
			// 解析 LastModified 字段
			// 与阿里云返回的格式不一样
			parsedTime, err := time.Parse(time.RFC3339, content.LastModified)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse LastModified: %v", err)
			}

			fileList = append(fileList, model.FileInfo{
				Key:           content.Key,
				ContentLength: content.Size,
				ETag:          content.ETag,
				LastModified:  parsedTime,
			})
		}

		// 返回 NextMarker 作为下一次查询的起点
		var nextMarker string
		if v.IsTruncated {
			nextMarker = v.NextMarker
		}

		return fileList, nextMarker, nil
	}
*/

func (c *CosCopier) CopyFile(srcBucket, srcObject, destBucket, destObject, srcRegion, destRegion string) error {
	// 使用提供的 region 信息创建源客户端和目标客户端
	if srcRegion == "" {
		srcRegion = os.Getenv("COS_REGION")
	}
	if destRegion == "" {
		destRegion = os.Getenv("COS_REGION")
	}

	// 如果未提供 destBucket，则使用 srcBucket
	if destBucket == "" {
		destBucket = srcBucket
	}

	// 构造源和目标客户端
	srcURL, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", srcBucket, srcRegion))
	destURL, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", destBucket, destRegion))

	destClient := cos.NewClient(&cos.BaseURL{BucketURL: destURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("SECRETID"),
			SecretKey: os.Getenv("SECRETKEY"),
		},
	})

	// 构建源文件的 URL
	sourceFileURL := fmt.Sprintf("%s/%s", srcURL.Host, srcObject)

	// 调用 COS 的 Copy 方法从源桶拷贝到目标桶
	_, _, err := destClient.Object.Copy(context.Background(), destObject, sourceFileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	return nil
}

func (c *CosCopier) MoveFile(srcBucket, srcObject, destBucket, destObject, srcRegion, destRegion string) error {
	// 如果没有传入目标桶，则使用源桶
	if destBucket == "" {
		destBucket = srcBucket
	}

	// 调用 CopyFile 执行复制操作
	err := c.CopyFile(srcBucket, srcObject, destBucket, destObject, srcRegion, destRegion)
	if err != nil {
		return fmt.Errorf("copy file failed: %v", err)
	}

	// 复制成功后，删除源文件
	deleter := NewCosDeleter()
	err = deleter.Delete(srcObject)
	if err != nil {
		return fmt.Errorf("delete file failed: %v", err)
	}

	return nil
}
