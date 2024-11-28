// Code generated by hertz generator. DO NOT EDIT.

package qiniu

import (
	qiniu "github.com/cloudisk/biz/handler/qiniu"
	"github.com/cloudwego/hertz/pkg/app/server"
)

/*
 This file will register all the routes of the services in the master idl.
 And it will update automatically when you use the "update" command for the idl.
 So don't modify the contents of the file, or your code will be deleted when it is updated.
*/

// Register register routes based on the IDL 'api.${HTTP Method}' annotation.
func Register(r *server.Hertz) {

	root := r.Group("/", rootMw()...)
	{
		_api := root.Group("/api", _apiMw()...)
		{
			_file := _api.Group("/file", _fileMw()...)
			{
				_content := _file.Group("/content", _contentMw()...)
				_content.GET("/download", append(_downloadMw(), qiniu.Download)...)
				_content.GET("/downloading", append(_downloadingMw(), qiniu.Downloading)...)
				_content.POST("/office", append(_officeuploadMw(), qiniu.OfficeUpload)...)
				_content.DELETE("/remove", append(_removeMw(), qiniu.Remove)...)
				_content.POST("/save", append(_saveMw(), qiniu.Save)...)
				_content.POST("/upload", append(_uploadMw(), qiniu.Upload)...)
			}
		}
	}
}