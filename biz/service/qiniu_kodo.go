package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

type QiniuCommoner struct {
	accessKey  string
	secretKey  string
	bucketName string
	endpoint   string
	zone       string
}

// getZone 根据区域名称获取存储区域配置
func getZone(zoneName string) *storage.Zone {
	switch strings.ToLower(zoneName) {
	case "huadong", "east":
		return &storage.ZoneHuadong
	case "huanan", "south":
		return &storage.ZoneHuanan
	case "huabei", "north":
		return &storage.ZoneHuabei
	case "beimei", "na":
		return &storage.ZoneBeimei
	case "xinjiapo", "singapore":
		return &storage.ZoneXinjiapo
	default:
		return &storage.ZoneHuanan // 默认使用华南区域
	}
}

// getConfig 获取存储配置
func (q *QiniuCommoner) getConfig() *storage.Config {
	return &storage.Config{
		Zone:          getZone(q.zone),
		UseCdnDomains: false,
		UseHTTPS:      true,
	}
}

func NewQiniuClient() *QiniuCommoner {
	return &QiniuCommoner{
		accessKey:  os.Getenv("QINIU_ACCESSKEY"),
		secretKey:  os.Getenv("QINIU_SECRETKEY"),
		bucketName: os.Getenv("QINIU_BUCKET"),
		endpoint:   os.Getenv("QINIU_ENDPOINT"),
		zone:       os.Getenv("QINIU_ZONE"), // 从环境变量获取区域设置
	}
}

// Upload 将文件上传到七牛云
func (q *QiniuCommoner) Upload(file multipart.File, objectName string) (int64, error) {
	// 创建凭证
	mac := qbox.NewMac(q.accessKey, q.secretKey)

	// 创建上传策略
	putPolicy := storage.PutPolicy{
		Scope: q.bucketName,
	}
	upToken := putPolicy.UploadToken(mac)

	// 创建上传管理器
	formUploader := storage.NewFormUploader(q.getConfig())

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "qiniu-upload-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 将multipart.File的内容复制到临时文件
	size, err := io.Copy(tempFile, file)
	if err != nil {
		return 0, fmt.Errorf("failed to copy file content: %w", err)
	}

	// 重置文件指针到开始位置
	if _, err := tempFile.Seek(0, 0); err != nil {
		return 0, fmt.Errorf("failed to seek file: %w", err)
	}

	// 定义上传返回值
	ret := storage.PutRet{}

	// 执行上传
	err = formUploader.PutFile(context.Background(), &ret, upToken, objectName, tempFile.Name(), nil)
	if err != nil {
		return 0, fmt.Errorf("upload failed: %w", err)
	}

	return size, nil
}

// ReaderUpload 使用io.ReadCloser上传文件到七牛云
func (q *QiniuCommoner) ReaderUpload(file io.ReadCloser, objectName string) (int64, error) {
	// 创建凭证
	mac := qbox.NewMac(q.accessKey, q.secretKey)

	// 创建上传策略
	putPolicy := storage.PutPolicy{
		Scope: q.bucketName,
	}
	upToken := putPolicy.UploadToken(mac)

	// 创建上传管理器
	formUploader := storage.NewFormUploader(q.getConfig())

	// 定义上传返回值
	ret := storage.PutRet{}

	// 执行上传
	err := formUploader.Put(context.Background(), &ret, upToken, objectName, file, -1, nil)
	if err != nil {
		return 0, fmt.Errorf("upload failed: %w", err)
	}

	// 获取文件信息
	bucketManager := storage.NewBucketManager(mac, q.getConfig())
	fileInfo, err := bucketManager.Stat(q.bucketName, objectName)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return fileInfo.Fsize, nil
}

// GeneratePublicURL 生成公开访问的下载链接
func (q *QiniuCommoner) GeneratePublicURL(objectName string) string {
	return storage.MakePublicURL(q.endpoint, objectName)
}

// GeneratePrivateURL 生成私有访问的下载链接
func (q *QiniuCommoner) GeneratePrivateURL(objectName string, expiryTime int64) string {
	mac := qbox.NewMac(q.accessKey, q.secretKey)
	return storage.MakePrivateURL(mac, q.endpoint, objectName, expiryTime)
}

// DownloadFileToLocal 从七牛云Kodo下载文件到本地目录
func (q *QiniuCommoner) KodoDownloadFileToLocal(objectName string) (string, error) {
	localDir := os.Getenv("LOCAL_DOWNLOAD_DIR") // 本地下载目录
	if localDir == "" || objectName == "" {
		return "", fmt.Errorf("invalid parameters: local directory and object name are required")
	}

	// 构造本地文件路径
	localFilePath := fmt.Sprintf("%s/%s", localDir, objectName)

	// 获取私有空间文件下载链接（1小时有效期）
	deadline := time.Now().Add(time.Hour).Unix()
	privateURL := q.GeneratePrivateURL(objectName, deadline)

	// 创建本地文件
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	// 下载文件
	resp, err := http.Get(privateURL)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	// 将响应内容写入本地文件
	_, err = io.Copy(localFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file content: %v", err)
	}

	return localFilePath, nil
}

// Delete 从七牛云中删除文件
func (q *QiniuCommoner) Delete(objectName string) error {
	mac := qbox.NewMac(q.accessKey, q.secretKey)
	bucketManager := storage.NewBucketManager(mac, q.getConfig())
	return bucketManager.Delete(q.bucketName, objectName)
}

// ListFiles 列出七牛云桶中的文件
func (q *QiniuCommoner) ListFiles(prefix, marker string, limit int) ([]storage.ListItem, string, error) {
	mac := qbox.NewMac(q.accessKey, q.secretKey)
	bucketManager := storage.NewBucketManager(mac, q.getConfig())

	// 获取文件列表
	entries, _, nextMarker, hasNext, err := bucketManager.ListFiles(q.bucketName, prefix, "", marker, limit)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list files: %v", err)
	}

	if !hasNext {
		nextMarker = ""
	}

	return entries, nextMarker, nil
}

// Copy 从七牛云中复制文件到新位置
func (q *QiniuCommoner) Copy(srcKey, destKey string, force bool) error {
	mac := qbox.NewMac(q.accessKey, q.secretKey)
	bucketManager := storage.NewBucketManager(mac, q.getConfig())

	// 执行复制操作
	err := bucketManager.Copy(q.bucketName, srcKey, q.bucketName, destKey, force)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}
	return nil
}

// Move 移动文件到七牛云存储中的新位置
func (q *QiniuCommoner) Move(srcObject, destObject string, force bool) error {
	mac := qbox.NewMac(q.accessKey, q.secretKey)
	bucketManager := storage.NewBucketManager(mac, q.getConfig())

	// 执行移动操作
	err := bucketManager.Move(q.bucketName, srcObject, q.bucketName, destObject, force)
	if err != nil {
		return fmt.Errorf("failed to move file: %v", err)
	}
	return nil
}
