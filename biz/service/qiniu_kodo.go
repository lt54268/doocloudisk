package service

import (
	"context"
	//"dooqiniu/internal/model"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
)

type QiniuCommoner struct {
	accessKey  string
	secretKey  string
	bucketName string
	endpoint   string
}

func NewQiniuClient() *QiniuCommoner {
	return &QiniuCommoner{
		accessKey:  os.Getenv("QINIU_ACCESSKEY"),
		secretKey:  os.Getenv("QINIU_SECRETKEY"),
		bucketName: os.Getenv("QINIU_BUCKET"),
		endpoint:   os.Getenv("QINIU_ENDPOINT"),
	}
}

// 上传将文件上传到七牛云
func (q *QiniuCommoner) Upload(filePath, objectName string) (*storage.PutRet, error) {
	mac := credentials.NewCredentials(q.accessKey, q.secretKey)

	// 创建具有凭证的上传管理器
	options := uploader.UploadManagerOptions{
		Options: http_client.Options{
			Credentials: mac,
		},
	}
	uploadManager := uploader.NewUploadManager(&options)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// 使用目标路径设置对象选项
	objectOptions := &uploader.ObjectOptions{
		BucketName: q.bucketName,
		ObjectName: &objectName,
		FileName:   filepath.Base(filePath),
	}

	// 执行上传
	var putRet storage.PutRet
	err = uploadManager.UploadFile(context.Background(), filePath, objectOptions, &putRet)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	/*
		// 上传后，使用 ListFiles 获取文件信息
		files, _, err := q.ListFiles(objectName, "", 1)
		if err != nil || len(files) == 0 {
			return nil, fmt.Errorf("failed to retrieve file info: %v", err)
		}

		// 提取所需信息
		fileInfo := files[0]
		uploadResponse := &model.UploadResponse{
			ContentLength: fileInfo.Fsize,
			ETag:          fileInfo.Hash,
			LastModified:  time.Unix(fileInfo.PutTime/1e7, 0).UTC(), // Convert to time
		}
	*/

	return &putRet, nil
}

// GeneratePublicURL 生成公开访问的下载链接
func (q *QiniuCommoner) GeneratePublicURL(objectName string) string {
	return storage.MakePublicURL(q.endpoint, objectName)
}

// GeneratePrivateURL 生成私有访问的下载链接
func (q *QiniuCommoner) GeneratePrivateURL(objectName string, expiryTime int64) string {
	mac := auth.New(q.accessKey, q.secretKey)
	return storage.MakePrivateURL(mac, q.endpoint, objectName, expiryTime)
}

// Delete 从七牛云中删除文件
func (q *QiniuCommoner) Delete(objectName string) error {
	mac := auth.New(q.accessKey, q.secretKey)

	bucketManager := storage.NewBucketManager(mac, &storage.Config{})

	// 执行删除操作
	err := bucketManager.Delete(q.bucketName, objectName)
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}

// ListFiles 列出七牛云桶中的文件
func (q *QiniuCommoner) ListFiles(prefix, marker string, limit int) ([]storage.ListItem, string, error) {
	mac := auth.New(q.accessKey, q.secretKey)

	bucketManager := storage.NewBucketManager(mac, &storage.Config{})

	// 获取文件列表
	entries, _, nextMarker, hasNext, err := bucketManager.ListFiles(q.bucketName, prefix, "", marker, limit)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list files: %v", err)
	}

	// 只返回文件项列表和下一页的 marker
	if !hasNext {
		nextMarker = ""
	}

	// 返回文件列表和下一页的游标
	return entries, nextMarker, nil
}

// Copy 从七牛云中复制文件到新位置
func (q *QiniuCommoner) Copy(srcKey, destKey string, force bool) error {
	mac := auth.New(q.accessKey, q.secretKey)

	bucketManager := storage.NewBucketManager(mac, &storage.Config{})

	// 执行复制操作
	err := bucketManager.Copy(q.bucketName, srcKey, q.bucketName, destKey, force)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}
	return nil
}

// Move 移动文件到七牛云存储中的新位置
func (q *QiniuCommoner) Move(srcObject, destObject string, force bool) error {
	mac := auth.New(q.accessKey, q.secretKey)

	bucketManager := storage.NewBucketManager(mac, &storage.Config{})

	// 执行移动操作
	err := bucketManager.Move(q.bucketName, srcObject, q.bucketName, destObject, force)
	if err != nil {
		return fmt.Errorf("failed to move file: %v", err)
	}
	return nil
}
