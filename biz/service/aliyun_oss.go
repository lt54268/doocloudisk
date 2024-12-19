package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/cloudisk/biz/dal/query"
	"github.com/cloudisk/pkg/config"
)

// 实现了 Uploader 接口，支持阿里云 OSS 上传
type OssUploader struct{}

// 日志查询服务
type LogQueryService struct {
	client sls.ClientInterface
}

// 返回 OssUploader 实例
func NewOssUploader() *OssUploader {
	return &OssUploader{}
}

// 构造函数，返回 LogQueryService 实例
func NewLogQueryService() *LogQueryService {
	Endpoint := os.Getenv("OSS_LOG_ENDPOINT")
	AccessKeyId := os.Getenv("OSS_ACCESS_KEY_ID")
	AccessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")

	provider := sls.NewStaticCredentialsProvider(AccessKeyId, AccessKeySecret, "")
	client := sls.CreateNormalInterfaceV2(Endpoint, provider)

	return &LogQueryService{client: client}
}

// Upload 实现 Uploader 接口中的 Upload 方法
func (u *OssUploader) Upload(file multipart.File, objectName string, pid int64) (int64, error) {
	// 获取文件的完整路径
	fullPath := objectName
	if strings.Contains(objectName, "/") {
		// 如果已经包含路径分隔符，说明是从 webkitRelativePath 传入的，直接使用
		fullPath = objectName
	} else {
		// 从 pid 开始构建文件夹路径
		paths := []string{}
		currentPid := pid

		// 递归查找父文件夹
		for currentPid > 0 {
			parentFile, err := query.Q.File.Where(query.File.ID.Eq(currentPid)).First()
			if err != nil {
				log.Printf("找不到父文件夹，ID: %d, 错误: %v", currentPid, err)
				break
			}

			if parentFile.Type == "folder" {
				log.Printf("添加文件夹到路径: %s", parentFile.Name)
				paths = append([]string{parentFile.Name}, paths...)
			}

			currentPid = parentFile.Pid
		}

		// 构建完整的文件路径
		if len(paths) > 0 {
			fullPath = strings.Join(paths, "/") + "/" + objectName
			log.Printf("构建的完整文件路径: %s", fullPath)
		} else {
			log.Printf("没有找到父文件夹，使用原始文件名: %s", objectName)
		}
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.OssAccessKeyId,
			config.OssAccessKeySecret,
			"")).
		WithRegion(config.OssRegion).WithConnectTimeout(3 * time.Second).WithRetryMaxAttempts(3)

	client := oss.NewClient(cfg)

	// 创建上传请求
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(config.OssBucket),
		Key:    oss.Ptr(fullPath),
		Body:   file,
	}

	log.Printf("开始上传文件到路径: %s", fullPath)

	// 上传文件
	_, err := client.PutObject(context.TODO(), request)
	if err != nil {
		log.Printf("文件上传失败: %v", err)
		return 0, fmt.Errorf("failed to upload object: %v", err)
	}

	// 上传成功后，获取文件信息
	objectInfo, err := client.HeadObject(context.TODO(), &oss.HeadObjectRequest{
		Bucket: oss.Ptr(config.OssBucket),
		Key:    oss.Ptr(fullPath),
	})
	if err != nil {
		log.Printf("获取文件信息失败: %v", err)
		return 0, fmt.Errorf("failed to retrieve object info: %v", err)
	}

	log.Printf("文件上传成功，大小: %d bytes", objectInfo.ContentLength)
	return objectInfo.ContentLength, nil
}

func (u *OssUploader) ReaderUpload(file io.ReadCloser, objectName string) (int64, error) {
	log.Printf("开始上传文件到路径: %s", objectName)

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.OssAccessKeyId,
			config.OssAccessKeySecret,
			"")).
		WithRegion(config.OssRegion).WithConnectTimeout(3 * time.Second).WithRetryMaxAttempts(3)

	client := oss.NewClient(cfg)

	// 创建上传请求
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(config.OssBucket),
		Key:    oss.Ptr(objectName),
		Body:   file,
	}

	// 上传文件
	_, err := client.PutObject(context.TODO(), request)
	if err != nil {
		log.Printf("文件上传失败: %v", err)
		return 0, fmt.Errorf("failed to upload object: %v", err)
	}

	// 上传成功后，获取文件信息
	objectInfo, err := client.HeadObject(context.TODO(), &oss.HeadObjectRequest{
		Bucket: oss.Ptr(config.OssBucket),
		Key:    oss.Ptr(objectName),
	})
	if err != nil {
		log.Printf("获取文件信息失败: %v", err)
		return 0, fmt.Errorf("failed to retrieve object info: %v", err)
	}

	log.Printf("文件上传成功，大小: %d bytes", objectInfo.ContentLength)
	return objectInfo.ContentLength, nil
}

// DownloadFile 从阿里云OSS下载文件
func DownloadFile(objectName string) ([]byte, error) {
	bucketName := os.Getenv("OSS_BUCKET")
	region := os.Getenv("OSS_REGION")

	if bucketName == "" || region == "" || objectName == "" {
		return nil, errors.New("invalid parameters: bucket name, region, and object name are required")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	}

	// 发起下载请求
	result, err := client.GetObject(context.TODO(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %v", err)
	}
	defer result.Body.Close()

	// 读取文件内容
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %v", err)
	}

	return data, nil
}

// DownloadFileToLocal 从阿里云OSS下载文件到本地目录
func DownloadFileToLocal(objectName string) (string, error) {
	bucketName := os.Getenv("OSS_BUCKET")
	region := os.Getenv("OSS_REGION")
	localDir := os.Getenv("LOCAL_DOWNLOAD_DIR") // 本地下载目录

	if bucketName == "" || region == "" || objectName == "" || localDir == "" {
		return "", errors.New("invalid parameters: bucket name, region, object name, and local directory are required")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	// 构造本地文件路径
	localFilePath := fmt.Sprintf("%s/%s", localDir, objectName)

	// 使用 GetObjectToFile 方法将文件下载到本地
	_, err := client.GetObjectToFile(context.TODO(), &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	}, localFilePath)

	if err != nil {
		return "", fmt.Errorf("failed to download file to local path: %v", err)
	}

	return localFilePath, nil
}

// DeleteFile 从阿里云OSS删除文件
func DeleteFile(objectName string) error {
	bucketName := os.Getenv("OSS_BUCKET")
	region := os.Getenv("OSS_REGION")

	if bucketName == "" || region == "" || objectName == "" {
		return errors.New("invalid parameters: bucket name, region, and object name are required")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	request := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	}

	_, err := client.DeleteObject(context.TODO(), request)
	if err != nil {
		return fmt.Errorf("failed to delete object: %v", err)
	}

	return nil
}

// ListFiles 从阿里云OSS获取文件列表
func ListFiles() (any, error) {
	bucketName := os.Getenv("OSS_BUCKET")
	region := os.Getenv("OSS_REGION")

	if bucketName == "" || region == "" {
		return nil, errors.New("invalid parameters: bucket name and region are required")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	request := &oss.ListObjectsV2Request{
		Bucket: oss.Ptr(bucketName),
	}

	p := client.NewListObjectsV2Paginator(request)
	// var fileInfos []model.FileInfo

	// for p.HasNext() {
	// 	page, err := p.NextPage(context.TODO())
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to get objects list: %v", err)
	// 	}

	// 	// 收集每个对象的信息
	// 	for _, obj := range page.Contents {
	// 		fileInfos = append(fileInfos, model.FileInfo{
	// 			Key:           oss.ToString(obj.Key),
	// 			ContentLength: obj.Size,
	// 			ETag:          oss.ToString(obj.ETag),
	// 			LastModified:  oss.ToTime(obj.LastModified),
	// 		})
	// 	}
	// }

	return p, nil
}

func ListFilesV2(prefix, continuationToken string, maxKeys int) (any, string, error) {
	bucketName := os.Getenv("OSS_BUCKET")
	region := os.Getenv("OSS_REGION")

	if bucketName == "" || region == "" {
		return nil, "", errors.New("invalid parameters: bucket name and region are required")
	}

	// 设置默认值
	if prefix == "" {
		prefix = "" // 默认不筛选文件前缀，列出所有对象
	}
	if maxKeys == 0 {
		maxKeys = 1000 // 默认最多返回1000个文件
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	// 创建请求
	request := &oss.ListObjectsV2Request{
		Bucket:            oss.Ptr(bucketName),
		Prefix:            oss.Ptr(prefix),
		ContinuationToken: oss.Ptr(continuationToken),
		MaxKeys:           int32(maxKeys),
	}

	// 使用分页器
	paginator := client.NewListObjectsV2Paginator(request)
	// var fileInfos []model.FileInfo
	var nextContinuationToken string
	totalFiles := 0 // 用于控制返回文件数量

	for paginator.HasNext() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, "", fmt.Errorf("failed to get objects list: %v", err)
		}

		// 收集每个对象的信息
		for _, obj := range page.Contents {
			fmt.Print(obj)
			// fileInfos = append(fileInfos, model.FileInfo{
			// 	Key:           oss.ToString(obj.Key),
			// 	ContentLength: obj.Size,
			// 	ETag:          oss.ToString(obj.ETag),
			// 	LastModified:  oss.ToTime(obj.LastModified),
			// })
			totalFiles++
			// 如果已收集的文件数量达到了限制，则停止
			if totalFiles >= maxKeys {
				// 如果返回了 NextContinuationToken，使用它作为下一次查询的起点
				if page.NextContinuationToken != nil {
					nextContinuationToken = *page.NextContinuationToken
				}
				break
			}
		}

		// 如果已经达到限制数量，则不再请求更多页面
		if totalFiles >= maxKeys {
			break
		}

		// 如果返回了 NextContinuationToken，使用它作为下一次查询的起点
		if page.NextContinuationToken != nil {
			nextContinuationToken = *page.NextContinuationToken
		} else {
			break
		}
	}

	return nil, nextContinuationToken, nil
}

// CopyFile 拷贝文件到目标存储空间
func CopyFile(srcBucket, srcObject, destBucket, destObject string) error {
	region := os.Getenv("OSS_REGION")

	if srcBucket == "" || srcObject == "" || destObject == "" || region == "" {
		return errors.New("invalid parameters: source bucket, source object, destination object, and region are required")
	}

	// 如果目标存储空间未指定，默认为源存储空间
	if destBucket == "" {
		destBucket = srcBucket
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	request := &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(destBucket),
		Key:          oss.Ptr(destObject),
		SourceBucket: oss.Ptr(srcBucket),
		SourceKey:    oss.Ptr(srcObject),
	}

	_, err := client.CopyObject(context.TODO(), request)
	if err != nil {
		return fmt.Errorf("failed to copy object: %v", err)
	}

	return nil
}

// RenameFile 将源对象重命名为目标对象
func RenameFile(srcObject, destObject string) error {
	bucketName := os.Getenv("OSS_BUCKET")
	region := os.Getenv("OSS_REGION")

	if bucketName == "" || region == "" || srcObject == "" || destObject == "" {
		return errors.New("invalid parameters: bucket name, region, source object, and destination object are required")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	// 创建 CopyObject 请求，将源对象复制到目标位置
	copyRequest := &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(bucketName),
		Key:          oss.Ptr(destObject),
		SourceKey:    oss.Ptr(srcObject),
		SourceBucket: oss.Ptr(bucketName),
	}

	// 执行 CopyObject 操作
	_, err := client.CopyObject(context.TODO(), copyRequest)
	if err != nil {
		return fmt.Errorf("failed to copy object '%s' to '%s': %v", srcObject, destObject, err)
	}

	// 创建 DeleteObject 请求，删除源对象
	deleteRequest := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(srcObject),
	}

	// 执行 DeleteObject 操作
	_, err = client.DeleteObject(context.TODO(), deleteRequest)
	if err != nil {
		return fmt.Errorf("failed to delete source object '%s': %v", srcObject, err)
	}

	return nil
}

// QueryLogs 查询日志
func (s *LogQueryService) QueryLogs(projectName, logStoreName, query string, startTime, endTime int64, limit, offset int) ([]map[string]string, error) {
	// 发起日志查询
	response, err := s.client.GetLogs(projectName, logStoreName, "", startTime, endTime, query, int64(limit), int64(offset), true)
	if err != nil {
		return nil, fmt.Errorf("GetLogs failed: %v", err)
	}

	log.Println("Logs retrieved successfully.")
	return response.Logs, nil
}
