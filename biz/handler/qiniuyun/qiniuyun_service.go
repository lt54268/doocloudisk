// Code generated by hertz generator.

package qiniuyun

import (
	"context"

	qiniuyun "github.com/cloudisk/biz/model/qiniuyun"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Upload .
// @router /api/file/content/upload [POST]
func Upload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniuyun.UploadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(qiniuyun.UploadResp)

	c.JSON(consts.StatusOK, resp)
}

// OfficeUpload .
// @router /api/file/content/office [POST]
func OfficeUpload(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniuyun.OfficeUploadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(qiniuyun.OfficeUploadResp)

	c.JSON(consts.StatusOK, resp)
}

// Save .
// @router /api/file/content/save [POST]
func Save(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniuyun.SaveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(qiniuyun.SaveResp)

	c.JSON(consts.StatusOK, resp)
}

// Download .
// @router /api/file/content/download [GET]
func Download(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniuyun.DownloadReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(qiniuyun.DownloadResp)

	c.JSON(consts.StatusOK, resp)
}

// Remove .
// @router /api/file/content/remove [DELETE]
func Remove(ctx context.Context, c *app.RequestContext) {
	var err error
	var req qiniuyun.RemoveReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp := new(qiniuyun.RemoveResp)

	c.JSON(consts.StatusOK, resp)
}
