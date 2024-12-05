// Code generated by hertz generator.

package tencent

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/cloudisk/biz/dal/query"
	tencent "github.com/cloudisk/biz/model/tencent"
	"github.com/cloudisk/biz/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Upload .
// @router /api/file/content/upload [POST]
func Upload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.UploadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}
	user, _ := service.GetUserInfo(c.GetHeader("Token"))
	form, err := c.MultipartForm()
	if err != nil {
		return
	}
	file := form.File["files"][0]
	pid, _ := strconv.Atoi(req.GetPid())
	cover, _ := strconv.ParseBool(req.GetCover())
	webkitRelativePath := req.GetWebkitRelativePath()

	log.Printf("开始上传文件: %s", file.Filename)

	item, err := service.Upload(user, pid, webkitRelativePath, cover, *file)
	if err != nil {
		log.Printf("文件上传失败: %s, 错误: %v", file.Filename, err)
		resp := new(tencent.UploadResp)
		resp.Ret = 0
		resp.Msg = "文件上传失败: " + err.Error()
		c.JSON(consts.StatusInternalServerError, resp)
		return
	}

	log.Printf("文件上传成功: %s", file.Filename)

	resp := new(tencent.UploadResp)
	resp.Data = append(resp.Data, item)
	resp.Ret = 1
	resp.Msg = file.Filename + " 上传成功"

	c.JSON(consts.StatusOK, resp)
}

// OfficeUpload .
// @router /api/file/content/office [POST]
func OfficeUpload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.OfficeUploadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}
	user, _ := service.GetUserInfo(c.GetHeader("Token"))
	id, _ := strconv.Atoi(req.GetId())
	status, _ := strconv.Atoi(req.GetStatus())
	key := req.GetKey()
	urlPath := req.GetUrl()
	err = service.OfficeUpload(user, id, status, key, urlPath)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}
	resp := new(tencent.OfficeUploadResp)
	resp.Error = "1"
	c.JSON(consts.StatusOK, resp)
}

// Save .
// @router /api/file/content/save [POST]
func Save(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.SaveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(tencent.SaveResp)

	c.JSON(consts.StatusOK, resp)
}

// Download .
// @router /api/file/content/download [GET]
func Download(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	fileID := req.FileId
	file, _ := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	cosFileName := file.Name + "." + file.Ext
	log.Printf("开始保存文件: %s, ID: %d", cosFileName, fileID)

	localFilePath, err := service.NewCosDownloader().DownloadFileToLocal(cosFileName)
	if err != nil {
		log.Printf("保存文件失败: %s, 错误: %v", cosFileName, err)
		resp := new(tencent.DownloadResp)
		resp.Ret = 0
		resp.Msg = "保存文件失败: " + err.Error()
		c.JSON(consts.StatusInternalServerError, resp)
		return
	}

	err = service.UpdateFileContentURLInDB(int64(fileID), localFilePath)
	if err != nil {
		log.Printf("更新文件URL失败, ID: %d, 错误: %v", fileID, err)
		resp := new(tencent.DownloadResp)
		resp.Ret = 0
		resp.Msg = "更新文件信息失败: " + err.Error()
		c.JSON(consts.StatusInternalServerError, resp)
		return
	}

	log.Printf("文件保存成功: %s, ID: %d", cosFileName, fileID)

	resp := new(tencent.DownloadResp)
	resp.Ret = 1
	resp.Msg = "保存成功"

	c.JSON(consts.StatusOK, resp)
}

// Downloading .
// @router /api/file/content/downloading [GET]
func Downloading(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	fileID := req.FileId
	file, _ := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	cosFileName := file.Name + "." + file.Ext
	log.Printf("开始下载文件: %s, ID: %d", cosFileName, fileID)

	fileData, err := service.NewCosDownloader().DownloadFile(cosFileName)
	if err != nil {
		log.Printf("下载文件失败: %s, 错误: %v", cosFileName, err)
		c.String(consts.StatusInternalServerError, "下载文件失败: "+err.Error())
		return
	}

	log.Printf("文件下载成功: %s, ID: %d", cosFileName, fileID)

	c.Header("Content-Disposition", "attachment; filename="+cosFileName)
	c.Header("Content-Type", "application/octet-stream")
	c.Data(consts.StatusOK, "application/octet-stream", fileData)
}

// Remove .
// @router /api/file/content/remove [DELETE]
func Remove(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.RemoveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	user, _ := service.GetUserInfo(c.GetHeader("Token"))
	fileID := req.FileId
	log.Printf("开始删除本地文件, ID: %d", fileID)
	err = service.DeleteLocalFileWithUser(user, fileID)
	if err != nil {
		log.Printf("删除文件失败, ID: %d, 错误: %v", fileID, err)
		resp := new(tencent.RemoveResp)
		resp.Ret = 0
		resp.Msg = "删除失败: " + err.Error()
		c.JSON(consts.StatusInternalServerError, resp)
		return
	}
	log.Printf("本地文件删除成功, ID: %d", fileID)

	resp := new(tencent.RemoveResp)
	resp.Ret = 1
	resp.Msg = "删除成功"

	c.JSON(consts.StatusOK, resp)
}

// IoUpload .
// @router /api/file/content/io_upload [POST]
func IoUpload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tencent.IoUploadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	user, _ := service.GetUserInfo(c.GetHeader("Token"))
	fileID := req.GetFileId()
	file, _ := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	filePath, _ := service.GetFileContentURL(int64(fileID))
	fileReader, _ := os.Open(filePath)
	defer fileReader.Close()
	pid, _ := strconv.Atoi(req.GetPid())
	cover, _ := strconv.ParseBool(req.GetCover())
	webkitRelativePath := req.GetWebkitRelativePath()

	resp := new(tencent.IoUploadResp)
	fileName := file.Name + "." + file.Ext

	// 添加上传前的日志
	log.Printf("开始上传文件: %s, 文件ID: %d", fileName, fileID)

	item, err := service.Io_Upload(user, pid, webkitRelativePath, cover, fileReader, fileName)
	if err != nil {
		log.Printf("文件上传失败: %s, 错误: %v", fileName, err)
		resp.Ret = 0
		resp.Msg = "文件上传失败: " + err.Error()
		c.JSON(consts.StatusInternalServerError, resp)
		return
	}

	// 添加上传成功的日志
	log.Printf("文件上传成功: %s, 文件ID: %d", fileName, fileID)

	resp.Data = append(resp.Data, item)
	resp.Ret = 1
	resp.Msg = file.Name + "." + file.Ext + " 上传成功"

	c.JSON(consts.StatusOK, resp)
}
