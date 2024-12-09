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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudisk/biz/dal/query"
	"github.com/cloudisk/biz/model/common"
	"github.com/cloudisk/biz/model/gorm_gen"
	"gorm.io/gen"
	"gorm.io/gorm/clause"
)

// CloudUploader 定义统一的云存储上传接口
type CloudUploader interface {
	Upload(file multipart.File, objectName string) (ContentLength int64, err error)
	ReaderUpload(file io.ReadCloser, objectName string) (ContentLength int64, err error)
}

var (
	alioss        *OssUploader   = NewOssUploader()
	cosUploader   *CosUploader   = NewCosUploader()
	qiniuUploader *QiniuCommoner = NewQiniuClient()
)

// getCloudUploader 根据环境变量返回对应的云存储上传器
func getCloudUploader() CloudUploader {
	cloudProvider := os.Getenv("CLOUD_PROVIDER")
	switch cloudProvider {
	case "aliyun":
		return alioss
	case "tencent":
		return cosUploader
	case "qiniu":
		return qiniuUploader
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
	var pid int64 = f.Pid
	var pshare int64 = 0
	if f.Share > 0 {
		pshare = f.ID
	}

	var parentIds []int64
	for pid > 0 {
		parentIds = append(parentIds, pid)
		file, err := query.Q.File.Where(query.File.Pid.Eq(pid)).First()
		if err != nil {
			pid = 0
		}
		pid = file.Pid
		if file.Share > 0 {
			pshare = file.ID
		}
	}
	opids := f.Pids
	// Reverse the parent IDs to get the correct order
	for i, j := 0, len(parentIds)-1; i < j; i, j = i+1, j-1 {
		parentIds[i], parentIds[j] = parentIds[j], parentIds[i]
	}
	if len(parentIds) > 0 {
		// 反转数组
		for i, j := 0, len(parentIds)-1; i < j; i, j = i+1, j-1 {
			parentIds[i], parentIds[j] = parentIds[j], parentIds[i]
		}
		// 将数组转换为字符串
		f.Pids = "," + fmt.Sprintf("%v", parentIds) + ","
	} else {
		f.Pids = ""
	}
	f.Pshare = pshare
	// Save the file
	err := query.Q.File.Where(query.File.ID.Eq(f.ID)).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(f)
	if err != nil {
		return false
	}
	if f.Pids != opids {
		buf := make([]*gorm_gen.File, 0, 100)
		query.Q.File.Where(query.File.ID).FindInBatches(&buf, 100, func(tx gen.Dao, batch int) error {
			for _, result := range buf {
				saveBeforePP(result)
			}
			return nil
		})
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

	dirs := strings.Split(webkitRelativePath, "/")

	for _, dirName := range dirs[0 : len(dirs)-1] {
		if dirName == "" {
			continue
		}
		query.Q.Transaction(func(tx *query.Query) error {
			row, err := tx.File.Where(query.File.Pid.Eq(int64(pid)), query.File.Name.Eq(dirName)).Clauses(clause.Locking{Strength: "UPDATE"}).First()
			if err != nil {
				file := gorm_gen.File{Pid: int64(pid), Type: "folder", Name: dirName, Userid: user_id, CreatedID: int64(user.Userid)}
				HandleDuplicateName(&file)
				if saveBeforePP(&file) {
					tx.File.Create(&file)
				}
				pid = int(file.Pid)
			} else {
				pid = int(row.Pid)
			}
			return nil
		})
	}
	tmp_file := gorm_gen.File{Pid: int64(pid), Ext: getFileNameExt(file.Filename), Name: getFileNameWithoutExt(file.Filename), Userid: user_id, CreatedID: int64(user.Userid)}
	HandleDuplicateName(&tmp_file)
	_file_open, _ := file.Open()
	uploader := getCloudUploader()
	contentLength, err := uploader.Upload(_file_open, tmp_file.Name+"."+tmp_file.Ext)
	if err != nil {
		return nil, err
	}
	_file_open.Close()
	filetype := getFileType(file.Filename)

	_file := gorm_gen.File{Pid: int64(pid), Type: filetype, Name: getFileNameWithoutExt(file.Filename), Ext: getFileNameExt(file.Filename), Userid: user_id, CreatedID: int64(user.Userid), Size: contentLength}
	var newfile *gorm_gen.File
	if overwrite {
		newfile, _ = query.Q.File.Where(query.File.Ext.Eq(_file.Ext), query.File.Pid.Eq(_file.Pid), query.File.Name.Eq(_file.Name)).First()
	}
	if newfile == nil {
		overwrite = false
		newfile = &_file
	}

	fmt.Printf("res: %v\n", contentLength)
	err = query.Q.Transaction(func(tx *query.Query) error {
		HandleDuplicateName(newfile)
		saveBeforePP(newfile)
		baseURL := os.Getenv("SERVER_URL")
		downloadURL := fmt.Sprintf("%s/api/file/content/downloading?fileId=%d", baseURL, newfile.ID)
		content := map[string]interface{}{
			"from":   "",
			"type":   "document", // Assuming $type is "document"
			"ext":    filetype,
			"url":    "",
			"remote": downloadURL,
		}
		jsonData, err := json.Marshal(content)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		filecontent := gorm_gen.FileContent{Fid: newfile.ID, Content: string(jsonData), Text: "", Size: contentLength, Userid: user_id}
		tx.FileContent.Create(&filecontent)
		return nil
	})
	if err != nil {
		return nil, errors.New("file upload failed,SQL create failed: " + err.Error())
	}
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

func Io_Upload(user *User, pid int, webkitRelativePath string, overwrite bool, file io.ReadCloser, filename string) (*common.File, error) {
	uploader := getCloudUploader()
	contentLength, err := uploader.ReaderUpload(file, filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 获取已存在的文件记录
	existingFile, _ := query.Q.File.Where(query.File.Name.Eq(getFileNameWithoutExt(filename)),
		query.File.Ext.Eq(getFileNameExt(filename)),
		query.File.Pid.Eq(int64(pid))).First()

	// 生成下载URL
	baseURL := os.Getenv("SERVER_URL")
	downloadURL := fmt.Sprintf("%s/api/file/content/downloading?fileId=%d", baseURL, existingFile.ID)

	// 获取现有的content内容
	fileContent, err := query.Q.FileContent.Where(query.FileContent.Fid.Eq(existingFile.ID)).First()
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	var newContent string
	if fileContent.Content == "" {
		// 如果原内容为空，直接创建新的JSON
		newContent = fmt.Sprintf(`{"remote":"%s"}`, downloadURL)
	} else {
		// 如果原内容不为空，保持原有内容并在最后添加remote字段
		// 移除最后的 }
		trimmedContent := strings.TrimRight(strings.TrimSpace(fileContent.Content), "}")
		if trimmedContent == "{" {
			// 如果只有开括号，直接添加新字段
			newContent = fmt.Sprintf(`{"remote":"%s"}`, downloadURL)
		} else {
			// 在原有内容后添加新字段
			newContent = fmt.Sprintf(`%s,"remote":"%s"}`, trimmedContent, downloadURL)
		}
	}

	// 更新file_contents表中的content字段
	_, err = query.Q.FileContent.Where(query.FileContent.Fid.Eq(existingFile.ID)).
		Updates(map[string]interface{}{
			"content": newContent,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to update content field: %v", err)
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
	row, err := permissionFind(id, user, 1)
	if err != nil {
		return err
	}
	if status == 2 {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			fmt.Printf("Failed to parse URL: %v\n", err)
			return err
		}

		q := parsedURL.Query()
		q.Set("filename", key)
		parsedURL.RawQuery = q.Encode()

		var loadURL string

		// 开发环境
		loadURL = parsedURL.String()

		// 正式环境
		// loadURL = fmt.Sprintf("http://%s.3%s?%s", os.Getenv("APP_IPPR"), parsedURL.Path, parsedURL.RawQuery)

		fmt.Printf("Downloading from URL: %s\n", loadURL)
		response, err := http.Get(loadURL)
		if err != nil {
			fmt.Printf("Download failed: %v\n", err)
			return fmt.Errorf("failed to download file: %v", err)
		}
		defer response.Body.Close()

		uploader := getCloudUploader()
		contentLength, err := uploader.ReaderUpload(response.Body, key)
		if err != nil {
			fmt.Printf("Cloud upload failed: %v\n", err)
			return fmt.Errorf("failed to upload to cloud: %v", err)
		}

		log.Printf("office文件上传成功: %s", key)

		baseURL := os.Getenv("SERVER_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8888"
		}
		downloadURL := fmt.Sprintf("%s/api/file/content/downloading?fileId=%d", baseURL, row.ID)
		content := map[string]interface{}{
			"from": loadURL,
			"url":  downloadURL,
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
		return fmt.Errorf("file not found: %w", err)
	}

	// 获取本地下载目录
	localDir := os.Getenv("LOCAL_DOWNLOAD_DIR")
	if localDir == "" {
		return errors.New("local download directory not configured")
	}

	// 构造本地文件路径
	localFilePath := fmt.Sprintf("%s/%s.%s", localDir, file.Name, file.Ext)

	// 删除本地文件
	err = os.Remove(localFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found on local storage: %w", err)
		}
		return fmt.Errorf("failed to delete local file: %w", err)
	}

	return nil
}

// 封装数据库更新content字段的url部分
func UpdateFileContentURLInDB(fileID int64, localFilePath string) error {
	// 查询数据库获取原始content
	fileContent, err := query.Q.FileContent.Where(query.FileContent.Fid.Eq(fileID)).First()
	if err != nil {
		return fmt.Errorf("file content not found: %v", err)
	}

	// 解析现有的content字段
	var content map[string]interface{}
	err = json.Unmarshal([]byte(fileContent.Content), &content)
	if err != nil {
		return fmt.Errorf("failed to unmarshal content: %v", err)
	}

	// 更新content中的url字段为本地路径
	content["url"] = localFilePath

	// 将更新后的content转回JSON格式
	updatedContent, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal updated content: %v", err)
	}

	// 更新数据库中的content字段
	if _, err := query.Q.FileContent.Where(query.FileContent.Fid.Eq(fileID)).
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
