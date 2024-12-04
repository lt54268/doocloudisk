// Code generated by hertz generator.

package qiniu

import (
	"context"
	"os"
	"strconv"

	"github.com/cloudisk/biz/dal/query"
	qiniu "github.com/cloudisk/biz/model/qiniu"
	"github.com/cloudisk/biz/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Upload .
// @router /api/file/content/upload [POST]
func Upload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.UploadReq
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
	item, _ := service.Upload(user, pid, webkitRelativePath, cover, *file)
	resp := new(qiniu.UploadResp)
	resp.Data = append(resp.Data, item)
	resp.Ret = 1
	resp.Msg = file.Filename + " 上传成功"

	c.JSON(consts.StatusOK, resp)
}

// OfficeUpload .
// @router /api/file/content/office [POST]
func OfficeUpload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.OfficeUploadReq
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
	resp := new(qiniu.OfficeUploadResp)
	resp.Error = "1"
	c.JSON(consts.StatusOK, resp)
}

// Save .
// @router /api/file/content/save [POST]
func Save(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.SaveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(qiniu.SaveResp)

	c.JSON(consts.StatusOK, resp)
}

// Download .
// @router /api/file/content/download [GET]
func Download(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	fileID := req.FileId
	file, _ := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	kodoFileName := file.Name + "." + file.Ext
	localFilePath, _ := service.NewQiniuClient().DownloadFileToLocal(kodoFileName)
	_ = service.UpdateFileContentURLInDB(int64(fileID), localFilePath)

	resp := new(qiniu.DownloadResp)
	resp.Ret = 1
	resp.Msg = "下载成功"

	c.JSON(consts.StatusOK, resp)
}

// Downloading .
// @router /api/file/content/downloading [GET]
func Downloading(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	fileID := req.FileId
	file, _ := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	kodoFileName := file.Name + "." + file.Ext
	fileData, _ := service.NewQiniuClient().DownloadFile(kodoFileName)

	// resp := new(qiniu.DownloadResp)

	// c.JSON(consts.StatusOK, resp)

	c.Header("Content-Disposition", "attachment; filename="+file.Name+"."+file.Ext)
	c.Header("Content-Type", "application/octet-stream") // 设置通用的文件类型，可以根据文件类型修改

	c.Data(consts.StatusOK, "application/octet-stream", fileData)
}

// Remove .
// @router /api/file/content/remove [DELETE]
func Remove(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.RemoveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	user, _ := service.GetUserInfo(c.GetHeader("Token"))
	fileID := req.FileId
	_ = service.DeleteLocalFileWithUser(user, fileID)

	resp := new(qiniu.RemoveResp)
	resp.Ret = 1
	resp.Msg = "删除成功"

	c.JSON(consts.StatusOK, resp)
}

// IoUpload .
// @router /api/file/content/io_upload [POST]
func IoUpload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniu.IoUploadReq
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
	item, _ := service.Io_Upload(user, pid, webkitRelativePath, cover, fileReader, file.Name+"."+file.Ext)

	resp := new(qiniu.IoUploadResp)
	resp.Data = append(resp.Data, item)
	resp.Ret = 1
	resp.Msg = file.Name + "." + file.Ext + " 上传成功"

	c.JSON(consts.StatusOK, resp)
}
