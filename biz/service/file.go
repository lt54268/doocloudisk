package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudisk/biz/dal/query"
	"github.com/cloudisk/biz/model/common"
	"github.com/cloudisk/biz/model/gorm_gen"
)

// CloudUploader 定义统一的云存储上传接口
type CloudUploader interface {
	Upload(file multipart.File, objectName string, pid int64) (ContentLength int64, err error)
	ReaderUpload(file io.ReadCloser, objectName string) (ContentLength int64, err error)
}

var (
	alioss            *OssUploader   = NewOssUploader()
	cosUploader       *CosUploader   = NewCosUploader()
	qiniuUploader     *QiniuCommoner = NewQiniuClient()
	folderCreateMutex sync.Mutex
)

// getCloudUploader 根据环境变量返回对应的云存储上传器
func getCloudUploader() CloudUploader {
	cloudProvider := os.Getenv("CLOUD_PROVIDER")
	switch cloudProvider {
	case "aliyun":
		return alioss
	// case "tencent":
	// return cosUploader
	// case "qiniu":
	// return qiniuUploader
	default:
		return alioss // 默认使用阿里云OSS
	}
}

func IsContainInt(items []int64, item int64) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func B2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func isContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func getShareInfo(file *gorm_gen.File) (*gorm_gen.File, error) {
	if file.Share == 1 {
		return file, nil
	}
	pid := file.Pid
	for pid > 0 {
		result, err := query.Q.File.Where(query.File.Pid.Eq(int64(pid))).First()
		if err != nil {
			return nil, err
		}

		if result.Share == 1 {
			return result, nil
		}

		pid = result.Pid
	}
	return nil, nil
}

func getPermission(file *gorm_gen.File, userids []int64) int {
	if IsContainInt(userids, file.Userid) || IsContainInt(userids, file.CreatedID) {
		return 1000
	}
	row, err := getShareInfo(file)
	if err != nil {
		return -1
	}
	fileUser, err := query.Q.File_User.Where(query.File_User.FileID.Eq(row.ID)).Where(query.File_User.Userid.In(userids...)).Order(query.File_User.Permission).First()
	if err != nil {
		return -1
	}
	return int(fileUser.Permission)
}

func permissionFind(id int, user *User, limit int) (*gorm_gen.File, error) {
	file, err := query.Q.File.Where(query.File.ID.Eq(int64(id))).First()
	if err != nil {
		return nil, errors.New("文件夹不存在或已被删除")
	}
	var userids []int64
	if isContain(user.Identity, "temp") {
		userids = []int64{int64(user.Userid)}
	} else {
		userids = []int64{0, int64(user.Userid)}
	}
	permission := getPermission(file, userids)
	if permission < limit {
		switch limit {
		case 1000:
			return nil, errors.New("仅限所有者或创建者操作")
		case 1:
			return nil, errors.New("没有修改写入权限")
		default:
			return nil, errors.New("没有查看访问权限")
		}
	}
	return file, err
}

func saveBeforePP(f *gorm_gen.File) bool {
	if f == nil {
		return false
	}

	var pid int64 = f.Pid
	var pshare int64 = 0
	if f.Share > 0 {
		pshare = f.ID
	}

	var parentIds []int64
	for pid > 0 {
		parentIds = append(parentIds, pid)
		file, err := query.Q.File.Where(query.File.ID.Eq(pid)).First()
		if err != nil {
			// 如果找不到父文件夹，就停止查找，但不影响当前文件的保存
			break
		}
		pid = file.Pid
		if file.Share > 0 {
			pshare = file.ID
		}
	}

	// 设置文件的 Pids
	if len(parentIds) > 0 {
		// 反转数组以获得正确的路径顺序
		for i, j := 0, len(parentIds)-1; i < j; i, j = i+1, j-1 {
			parentIds[i], parentIds[j] = parentIds[j], parentIds[i]
		}
		f.Pids = "," + fmt.Sprintf("%v", parentIds) + ","
	} else {
		f.Pids = ""
	}
	f.Pshare = pshare

	// 保存文件记录
	err := query.Q.File.Create(f)
	if err != nil {
		return false
	}

	return true
}

func HandleDuplicateName(f *gorm_gen.File) error {
	// 检查是否存在同名文件
	count, err := query.Q.File.Where(query.File.Pid.Eq(f.Pid),
		query.File.Userid.Eq(f.Userid),
		query.File.Ext.Eq(f.Ext),
		query.File.Name.Eq(f.Name)).Count()
	if err != nil {
		return err
	}

	// 如果不存在同名文件，直接返回
	if count == 0 {
		return nil
	}

	// 解析原始文件名
	baseName := f.Name
	re := regexp.MustCompile(`^(.*?)\s*(?:\((\d+)\))?$`)
	matches := re.FindStringSubmatch(f.Name)
	if len(matches) >= 2 {
		baseName = matches[1]
	}

	// 查找所有相关文件名的最大编号
	var files []*gorm_gen.File
	err = query.Q.File.Where(query.File.Pid.Eq(f.Pid),
		query.File.Userid.Eq(f.Userid),
		query.File.Ext.Eq(f.Ext),
		query.File.Name.Like(baseName+"%")).Scan(&files)
	if err != nil {
		return err
	}

	// 找出最大编号
	maxNum := 1
	re = regexp.MustCompile(`\((\d+)\)$`)
	for _, file := range files {
		if matches := re.FindStringSubmatch(file.Name); len(matches) == 2 {
			if num, err := strconv.Atoi(matches[1]); err == nil && num >= maxNum {
				maxNum = num + 1
			}
		} else if file.Name == baseName {
			// 如果存在没有编号的原始文件名，确保从2开始
			if maxNum == 1 {
				maxNum = 2
			}
		}
	}

	// 生成新文件名
	newName := fmt.Sprintf("%s (%d)", baseName, maxNum)

	// 再次检查新文件名是否存在（以防并发）
	count, err = query.Q.File.Where(query.File.Pid.Eq(f.Pid),
		query.File.Userid.Eq(f.Userid),
		query.File.Ext.Eq(f.Ext),
		query.File.Name.Eq(newName)).Count()
	if err != nil {
		return err
	}

	if count > 0 {
		// 如果新文件名已存在，递归处理
		f.Name = newName
		return HandleDuplicateName(f)
	}

	f.Name = newName
	return nil
}

func getFileType(filename string) string {
	ext := getFileNameExt(filename)
	switch ext {
	case "text", "md", "markdown":
		return "document"
	case "drawio":
		return "drawio"
	case "mind":
		return "mind"
	case "doc", "docx":
		return "word"
	case "xls", "xlsx":
		return "excel"
	case "ppt", "pptx":
		return "ppt"
	case "wps":
		return "wps"
	case "jpg", "jpeg", "webp", "png", "gif", "bmp", "ico", "raw", "svg":
		return "picture"
	case "rar", "zip", "jar", "7-zip", "tar", "gzip", "7z", "gz", "apk", "dmg":
		return "archive"
	case "tif", "tiff":
		return "tif"
	case "dwg", "dxf":
		return "cad"
	case "ofd":
		return "ofd"
	case "pdf":
		return "pdf"
	case "txt":
		return "txt"
	case "htaccess", "htgroups", "htpasswd", "conf", "bat", "cmd", "cpp", "c", "cc", "cxx", "h", "hh", "hpp", "ino", "cs", "css",
		"dockerfile", "go", "golang", "html", "htm", "xhtml", "vue", "we", "wpy", "java", "js", "jsm", "jsx", "json", "jsp", "less", "lua", "makefile", "gnumakefile",
		"ocamlmakefile", "make", "mysql", "nginx", "ini", "cfg", "prefs", "m", "mm", "pl", "pm", "p6", "pl6", "pm6", "pgsql", "php",
		"inc", "phtml", "shtml", "php3", "php4", "php5", "phps", "phpt", "aw", "ctp", "module", "ps1", "py", "r", "rb", "ru", "gemspec", "rake", "guardfile", "rakefile",
		"gemfile", "rs", "sass", "scss", "sh", "bash", "bashrc", "sql", "sqlserver", "swift", "ts", "typescript", "str", "vbs", "vb", "v", "vh", "sv", "svh", "xml",
		"rdf", "rss", "wsdl", "xslt", "atom", "mathml", "mml", "xul", "xbl", "xaml", "yaml", "yml",
		"asp", "properties", "gitignore", "log", "bas", "prg", "python", "ftl", "aspx", "plist":
		return "code"
	case "mp3", "wav", "mp4", "flv", "avi", "mov", "wmv", "mkv", "3gp", "rm":
		return "media"
	case "xmind":
		return "xmind"
	case "rp":
		return "axure"
	default:
		return ""
	}
}

func getFileNameWithoutExt(filename string) string {
	// 获取文件名中最后一个 . 的位置
	index := strings.LastIndex(filename, ".")

	// 如果没有找到 .，则返回整个文件名
	if index == -1 {
		return filename
	}
	// 获取文件名中最后一个 . 之前的名称
	name := filename[:index]
	return name
}
func getFileNameExt(filename string) string {
	// 获取文件名中最后一个 . 的位置
	index := strings.LastIndex(filename, ".")

	// 如果没有找到 .，则返回整个文件名
	if index == -1 {
		return filename
	}
	fmt.Println(index, filename[index:])
	// 获取文件名中最后一个 . 之后的名称
	name := strings.Split(filename[index:], ".")[1]
	return name
}

func Upload(user *User, pid int, webkitRelativePath string, overwrite bool, file multipart.FileHeader) (*common.File, error) {
	user_id := int64(user.Userid)
	if pid > 0 {
		count, _ := query.Q.File.Where(query.File.Pid.Eq(int64(pid))).Count()
		if count >= 300 {
			return nil, errors.New("每个文件夹里最多只能创建300个文件或文件夹")
		}
		row, err := permissionFind(pid, user, 1)
		if err != nil {
			return nil, err
		}
		user_id = row.Userid
	} else {
		count, _ := query.Q.File.Where(query.File.Userid.Eq(int64(user.Userid)), query.File.Pid.Eq(0)).Count()
		if count >= 300 {
			return nil, errors.New("每个文件夹里最多只能创建300个文件或文件夹")
		}
	}

	// 处理文件夹路径
	var current_pid int64 = int64(pid)
	if webkitRelativePath != "" {
		dirs := strings.Split(webkitRelativePath, "/")
		// 创建文件夹层级
		for _, dirName := range dirs[0 : len(dirs)-1] {
			if dirName == "" {
				continue
			}

			// 使用互斥锁确保同一时间只有一个goroutine可以创建文件夹
			folderCreateMutex.Lock()
			var folder_id int64

			// 在锁内先查询文件夹是否存在
			existingFolder, err := query.Q.File.Where(
				query.File.Pid.Eq(current_pid),
				query.File.Name.Eq(dirName),
				query.File.Type.Eq("folder"),
			).First()

			if err == nil {
				// 文件夹已存在，直接使用
				folder_id = existingFolder.ID
				folderCreateMutex.Unlock()
			} else {
				// 文件夹不存在，在事务中创建
				err = query.Q.Transaction(func(tx *query.Query) error {
					// 再次检查文件夹是否存在（双重检查）
					existingFolder, err := tx.File.Where(
						query.File.Pid.Eq(current_pid),
						query.File.Name.Eq(dirName),
						query.File.Type.Eq("folder"),
					).First()

					if err == nil {
						// 另一个进程已经创建了文件夹
						folder_id = existingFolder.ID
						return nil
					}

					// 创建新文件夹
					newFolder := &gorm_gen.File{
						Pid:       current_pid,
						Type:      "folder",
						Name:      dirName,
						Userid:    user_id,
						CreatedID: int64(user.Userid),
					}
					HandleDuplicateName(newFolder)
					if err := tx.File.Create(newFolder); err != nil {
						return err
					}
					folder_id = newFolder.ID
					return nil
				})
				folderCreateMutex.Unlock()

				if err != nil {
					return nil, fmt.Errorf("创建文件夹失败: %v", err)
				}
			}

			if folder_id == 0 {
				return nil, fmt.Errorf("创建文件夹失败：无法获取有效的文件夹ID")
			}
			current_pid = folder_id
		}
	}

	// 处理文件上传
	_file_open, _ := file.Open()
	defer _file_open.Close()

	uploader := getCloudUploader()

	// 构建上传路径
	uploadPath := file.Filename
	if webkitRelativePath != "" {
		uploadPath = webkitRelativePath
	}

	contentLength, err := uploader.Upload(_file_open, uploadPath, int64(pid))
	if err != nil {
		return nil, err
	}

	filetype := getFileType(file.Filename)
	_file := gorm_gen.File{
		Pid:       current_pid,
		Type:      filetype,
		Name:      getFileNameWithoutExt(file.Filename),
		Ext:       getFileNameExt(file.Filename),
		Userid:    user_id,
		CreatedID: int64(user.Userid),
		Size:      contentLength,
	}

	var newfile *gorm_gen.File
	if overwrite {
		newfile, _ = query.Q.File.Where(
			query.File.Ext.Eq(_file.Ext),
			query.File.Pid.Eq(_file.Pid),
			query.File.Name.Eq(_file.Name),
		).First()
	}
	if newfile == nil {
		overwrite = false
		newfile = &_file
	}

	// 保存文件记录
	err = query.Q.Transaction(func(tx *query.Query) error {
		HandleDuplicateName(newfile)
		if err := tx.File.Create(newfile); err != nil {
			return err
		}

		baseURL := os.Getenv("SERVER_URL")
		downloadURL := fmt.Sprintf("http://%s/api/file/content/downloading?id=%d", baseURL, newfile.ID)
		content := map[string]interface{}{
			"from":      "",
			"type":      newfile.Type,
			"ext":       newfile.Ext,
			"url":       "",
			"cloud_url": downloadURL,
		}
		jsonData, err := json.Marshal(content)
		if err != nil {
			return err
		}

		filecontent := gorm_gen.FileContent{
			Fid:     newfile.ID,
			Content: string(jsonData),
			Text:    "",
			Size:    contentLength,
			Userid:  user_id,
		}
		return tx.FileContent.Create(&filecontent)
	})

	if err != nil {
		return nil, fmt.Errorf("file upload failed, SQL create failed: %v", err)
	}

	// 获取最新的文件记录
	newfile, _ = query.Q.File.Where(query.File.ID.Eq(newfile.ID)).First()

	fullName := newfile.Name + "." + newfile.Ext
	if webkitRelativePath != "" {
		fullName = webkitRelativePath
	}

	resp := &common.File{
		ID:        newfile.ID,
		Pid:       newfile.Pid,
		Pids:      newfile.Pids,
		Cid:       newfile.Cid,
		Name:      newfile.Name,
		Type:      newfile.Type,
		Ext:       newfile.Ext,
		Size:      newfile.Size,
		Userid:    newfile.Userid,
		Share:     newfile.Share,
		Pshare:    newfile.Pshare,
		CreatedID: newfile.CreatedID,
		CreatedAt: newfile.CreatedAt.Format("YYYY-mm-dd HH:MM:SS"),
		UpdatedAt: newfile.UpdatedAt.Format("YYYY-mm-dd HH:MM:SS"),
		FullName:  fullName,
		Overwrite: B2i(overwrite),
	}

	return resp, nil
}

func Io_Upload(user *User, fileID int, webkitRelativePath string, overwrite bool, file io.ReadCloser, filename string) (*common.File, error) {
	// 获取文件记录
	existingFile, err := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	if err != nil {
		return nil, fmt.Errorf("failed to get file record: %v", err)
	}

	// 构建文件夹路径
	paths := []string{}
	if existingFile.Pids != "" {
		// 移除开头和结尾的逗号
		pids := strings.Trim(existingFile.Pids, ",")
		if pids != "" {
			// 分割成ID数组
			pidArray := strings.Split(pids, ",")
			for _, pidStr := range pidArray {
				pid, err := strconv.ParseInt(pidStr, 10, 64)
				if err != nil {
					log.Printf("解析pid失败: %v", err)
					continue
				}
				// 获取文件夹信息
				folder, err := query.Q.File.Where(query.File.ID.Eq(pid)).First()
				if err != nil {
					log.Printf("获取文件夹信息失败, ID: %d, 错误: %v", pid, err)
					continue
				}
				if folder.Type == "folder" {
					log.Printf("从pids添加文件夹到路径: %s", folder.Name)
					paths = append(paths, folder.Name)
				}
			}
		}
	}

	// 构建完整的文件路径
	originalFileName := existingFile.Name
	if existingFile.Ext != "" {
		originalFileName = originalFileName + "." + existingFile.Ext
	}
	fullPath := originalFileName
	if len(paths) > 0 {
		fullPath = strings.Join(paths, "/") + "/" + originalFileName
		log.Printf("构建的完整文件路径: %s", fullPath)
	}

	// 上传文件
	uploader := getCloudUploader()
	contentLength, err := uploader.ReaderUpload(file, fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 生成下载URL
	baseURL := os.Getenv("SERVER_URL")
	downloadURL := fmt.Sprintf("http://%s/api/file/content/downloading?id=%d", baseURL, existingFile.ID)

	// 获取现有的content内容
	_, err = query.Q.FileContent.Where(query.FileContent.Fid.Eq(existingFile.ID)).First()
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	// 构建新的content内容
	content := map[string]interface{}{
		"from":      "",
		"type":      existingFile.Type,
		"ext":       existingFile.Ext,
		"url":       "",
		"cloud_url": downloadURL,
	}
	jsonData, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %v", err)
	}

	// 更新file_contents表中的content字段
	_, err = query.Q.FileContent.Where(query.FileContent.Fid.Eq(existingFile.ID)).
		Updates(map[string]interface{}{
			"content": string(jsonData),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to update content field: %v", err)
	}

	// 更新文件大小
	_, err = query.Q.File.Where(query.File.ID.Eq(existingFile.ID)).
		Updates(map[string]interface{}{
			"size": contentLength,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to update file size: %v", err)
	}

	fullName := existingFile.Name + "." + existingFile.Ext
	if webkitRelativePath != "" {
		fullName = webkitRelativePath
	}
	resp := &common.File{
		ID:        existingFile.ID,
		Pid:       existingFile.Pid,
		Pids:      existingFile.Pids,
		Cid:       existingFile.Cid,
		Name:      existingFile.Name,
		Type:      existingFile.Type,
		Ext:       existingFile.Ext,
		Size:      contentLength,
		Userid:    existingFile.Userid,
		Share:     existingFile.Share,
		Pshare:    existingFile.Pshare,
		CreatedID: existingFile.CreatedID,
		CreatedAt: existingFile.CreatedAt.Format("YYYY-mm-dd HH:MM:SS"),
		UpdatedAt: existingFile.UpdatedAt.Format("YYYY-mm-dd HH:MM:SS"),
		FullName:  fullName,
		Overwrite: B2i(overwrite),
	}

	return resp, nil
}

func OfficeUpload(user *User, id int, status int, key string, urlStr string) error {
	var loadURL string
	// 获取文件记录
	row, err := permissionFind(id, user, 1)
	if err != nil {
		return err
	}

	if status == 2 {
		// 构建文件名和路径
		originalFileName := row.Name
		if row.Ext != "" {
			originalFileName = originalFileName + "." + row.Ext
		}

		// 构建文件夹路径
		paths := []string{}
		currentPid := row.Pid
		for currentPid > 0 {
			parentFile, err := query.Q.File.Where(query.File.ID.Eq(currentPid)).First()
			if err != nil {
				break
			}
			if parentFile.Type == "folder" {
				paths = append([]string{parentFile.Name}, paths...)
			}
			currentPid = parentFile.Pid
		}

		// 构建完整的文件路径
		fullPath := originalFileName
		if len(paths) > 0 {
			fullPath = strings.Join(paths, "/") + "/" + originalFileName
		}

		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			fmt.Printf("Failed to parse URL: %v\n", err)
			return err
		}

		originalParams := strings.Split(parsedURL.RawQuery, "&")
		filenameIndex := -1

		for i, param := range originalParams {
			if strings.HasPrefix(param, "filename=") {
				filenameIndex = i
				break
			}
		}

		q := parsedURL.Query()
		q.Set("filename", key)
		if filenameIndex >= 0 {
			newParams := make([]string, len(originalParams))
			for i, param := range originalParams {
				if i == filenameIndex {
					newParams[i] = "filename=" + url.QueryEscape(key)
				} else if !strings.HasPrefix(param, "filename=") {
					newParams[i] = param
				}
			}
			parsedURL.RawQuery = strings.Join(newParams, "&")
		} else {
			parsedURL.RawQuery = q.Encode()
		}

		if appIPPR := os.Getenv("APP_IPPR"); appIPPR != "" {
			loadURL = fmt.Sprintf("http://%s.3%s?%s", appIPPR, parsedURL.Path, parsedURL.RawQuery)
		} else {
			loadURL = parsedURL.String()
		}

		response, err := http.Get(loadURL)
		if err != nil {
			fmt.Printf("Download failed: %v\n", err)
			return fmt.Errorf("failed to download file: %v", err)
		}
		defer response.Body.Close()

		uploader := getCloudUploader()
		contentLength, err := uploader.ReaderUpload(response.Body, fullPath)
		if err != nil {
			fmt.Printf("Cloud upload failed: %v\n", err)
			return fmt.Errorf("failed to upload to cloud: %v", err)
		}

		log.Printf("office文件上传成功: %s", fullPath)

		baseURL := os.Getenv("SERVER_URL")
		if baseURL == "" {
			baseURL = "localhost:8888"
		}
		downloadURL := fmt.Sprintf("http://%s/api/file/content/downloading_office?key=%s", baseURL, fullPath)
		content := map[string]interface{}{
			"from":      loadURL,
			"cloud_url": downloadURL,
		}
		jsonData, err := json.Marshal(content)
		if err != nil {
			return err
		}
		filecontent := gorm_gen.FileContent{Fid: row.ID, Content: string(jsonData), Text: "", Size: contentLength, Userid: int64(user.Userid)}
		query.Q.FileContent.Create(&filecontent)
		row.Size = contentLength
		row.UpdatedAt = time.Now()
		_, err = query.Q.File.Where(query.File.ID.Eq(row.ID)).Updates(row)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteLocalFileWithUser(user *User, fileID int32) error {
	// 查询数据库获取文件信息
	file, err := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// 获取本地下载目录
	localDir := os.Getenv("LOCAL_DOWNLOAD_DIR")
	if localDir == "" {
		return errors.New("local download directory not configured")
	}

	// 构造本地文件路径
	localFileName := fmt.Sprintf("%d_%s.%s", fileID, file.Name, file.Ext)
	localFilePath := filepath.Join(localDir, localFileName)

	// 删除本地文件
	err = os.Remove(localFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local file: %w", err)
	}

	// 获取文件内容记录
	fileContent, err := query.Q.FileContent.Where(query.FileContent.Fid.Eq(int64(fileID))).First()
	if err != nil {
		return fmt.Errorf("failed to get file content: %w", err)
	}

	// 解析当前的 content JSON
	var contentMap map[string]interface{}
	if err := json.Unmarshal([]byte(fileContent.Content), &contentMap); err != nil {
		return fmt.Errorf("failed to parse content JSON: %w", err)
	}

	// 更新 url 字段为空字符串
	contentMap["url"] = ""

	// 将更新后的 map 转回 JSON
	updatedContent, err := json.Marshal(contentMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated content: %w", err)
	}

	// 更新数据库中的 content 字段
	_, err = query.Q.FileContent.Where(query.FileContent.Fid.Eq(int64(fileID))).
		Updates(map[string]interface{}{
			"content": string(updatedContent),
		})
	if err != nil {
		return fmt.Errorf("failed to update file content: %w", err)
	}

	return nil
}

// 封装数据库更新content字段的url部分
func UpdateFileContentURLInDB(fileID int64, localFilePath string) error {
	// 查询数据库获取最新的content记录
	fileContent, err := query.Q.FileContent.
		Where(query.FileContent.Fid.Eq(fileID)).
		Order(query.FileContent.UpdatedAt.Desc()).
		First()
	if err != nil {
		return fmt.Errorf("file content not found: %v", err)
	}

	// 解析现有的content字段
	var content map[string]interface{}
	err = json.Unmarshal([]byte(fileContent.Content), &content)
	if err != nil {
		return fmt.Errorf("failed to unmarshal content: %v", err)
	}

	// 仅更新url字段，保留其他字段不变
	content["url"] = localFilePath

	// 将更新后的content转回JSON格式
	updatedContent, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal updated content: %v", err)
	}

	// 更新数据库中最新记录的content字段
	if _, err := query.Q.FileContent.
		Where(query.FileContent.Fid.Eq(fileID)).
		Where(query.FileContent.UpdatedAt.Eq(fileContent.UpdatedAt)).
		Update(query.FileContent.Content, string(updatedContent)); err != nil {
		return fmt.Errorf("failed to update content in database: %v", err)
	}

	return nil
}

func GetWorkDir() string {
	if workDir := os.Getenv("APP_WORKDIR"); workDir != "" {
		return workDir
	}
	return "/app" // 默认工作目录
}

// GetFileContentURL 获取文件内容的URL
func GetFileContentURL(fileID int64) (string, error) {
	content, err := query.Q.FileContent.Where(query.FileContent.Fid.Eq(fileID)).First()
	if err != nil {
		return "", err
	}

	var contentData map[string]interface{}
	err = json.Unmarshal([]byte(content.Content), &contentData)
	if err != nil {
		return "", err
	}

	url, ok := contentData["url"].(string)
	if !ok || url == "" {
		return "", errors.New("url not found in content")
	}

	url = strings.ReplaceAll(url, "\\/", "/")

	// 开发环境
	return "/Users/hitosea-005/Desktop/dootask_0.40.78/public/" + url, nil

	// 正式环境
	// return GetWorkDir() + "/" + url, nil

}

func DownloadFileFromURL(url string, localPath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP响应错误, 状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// SliceInt32ToInt64 converts a slice of int32 to a slice of int64
func SliceInt32ToInt64(in []int32) []int64 {
	out := make([]int64, len(in))
	for i, v := range in {
		out[i] = int64(v)
	}
	return out
}
