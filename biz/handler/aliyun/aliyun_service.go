// Code generated by hertz generator.

package aliyun

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/cloudisk/biz/dal/query"
	aliyun "github.com/cloudisk/biz/model/aliyun"
	"github.com/cloudisk/biz/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Upload .
// @router /api/file/content/upload [POST]
func Upload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req aliyun.UploadReq
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
	resp := new(aliyun.UploadResp)
	resp.Data = append(resp.Data, item)
	resp.Ret = 1
	resp.Msg = file.Filename + " 上传成功"

	c.JSON(consts.StatusOK, resp)
}

// OfficeUpload .
// @router /api/file/content/office [POST]
func OfficeUpload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req aliyun.OfficeUploadReq
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
	resp := new(aliyun.OfficeUploadResp)
	resp.Error = "1"
	c.JSON(consts.StatusOK, resp)
}

// Save .
// @router /api/file/content/save [POST]
func Save(ctx context.Context, c *app.RequestContext) {
	var err error
	var req aliyun.SaveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(aliyun.SaveResp)

	c.JSON(consts.StatusOK, resp)
}

// Download .
// @router /api/file/content/download [GET]
func Download(ctx context.Context, c *app.RequestContext) {
	var err error
	var req aliyun.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	// 从请求中获取文件ID
	fileID := req.FileId

	// 查询数据库获取文件信息
	file, err := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	if err != nil {
		c.String(consts.StatusNotFound, "File not found")
		return
	}

	// 构造阿里云OSS对象名称
	ossFileName := file.Name + "." + file.Ext

	// 从阿里云OSS下载文件到本地
	localFilePath, err := service.DownloadFileToLocal(ossFileName)
	if err != nil {
		c.String(consts.StatusInternalServerError, "File download failed: "+err.Error())
		return
	}

	// 将本地文件路径保存到数据库的content字段url部分
	err = service.UpdateFileContentURLInDB(int64(fileID), localFilePath)
	if err != nil {
		c.String(consts.StatusInternalServerError, "Failed to update file content URL in the database: "+err.Error())
		return
	}

	resp := new(aliyun.DownloadResp)
	resp.Ret = 1
	resp.Msg = "下载成功"

	c.JSON(consts.StatusOK, resp)
}

// Remove .
// @router /api/file/content/remove [DELETE]
func Remove(ctx context.Context, c *app.RequestContext) {
	var err error
	var req aliyun.RemoveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	user, _ := service.GetUserInfo(c.GetHeader("Token"))

	// 从请求中获取文件ID
	fileID := req.FileId

	// 调用封装后的删除函数
	err = service.DeleteLocalFileWithUser(user, fileID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.String(consts.StatusNotFound, "File not found on local storage")
		} else {
			c.String(consts.StatusInternalServerError, "Local file deletion failed: "+err.Error())
		}
		return
	}

	resp := new(aliyun.RemoveResp)
	resp.Ret = 1
	resp.Msg = "删除成功"

	c.JSON(consts.StatusOK, resp)
}

// Downloading .
// @router /api/file/content/downloading [GET]
func Downloading(ctx context.Context, c *app.RequestContext) {
	var err error
	var req aliyun.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	fileID := req.FileId

	file, err := query.Q.File.Where(query.File.ID.Eq(int64(fileID))).First()
	if err != nil {
		c.String(consts.StatusNotFound, "File not found")
		return
	}

	ossFileName := file.Name + "." + file.Ext

	fileData, err := service.DownloadFile(ossFileName)
	if err != nil {
		c.String(consts.StatusInternalServerError, "File download failed: "+err.Error())
		return
	}

	// resp := new(aliyun.DownloadResp)
	// resp.Ret = 1
	// resp.Msg = "下载成功"

	// c.JSON(consts.StatusOK, resp)

	// 设置响应头，告知浏览器进行文件下载
	c.Header("Content-Disposition", "attachment; filename="+file.Name+"."+file.Ext)
	c.Header("Content-Type", "application/octet-stream") // 设置通用的文件类型，可以根据文件类型修改

	// 返回文件内容
	c.Data(consts.StatusOK, "application/octet-stream", fileData)
}
