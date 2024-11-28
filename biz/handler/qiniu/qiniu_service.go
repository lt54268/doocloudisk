// Code generated by hertz generator.

package qiniu

import (
	"context"
	"strconv"

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

	resp := new(qiniu.DownloadResp)

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

	resp := new(qiniu.DownloadResp)

	c.JSON(consts.StatusOK, resp)
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

	resp := new(qiniu.RemoveResp)

	c.JSON(consts.StatusOK, resp)
}
