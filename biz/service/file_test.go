package service

import (
	"testing"
)

func Test_Trim(t *testing.T) {
	t.Log("start test", t.Name())
	t.Log(getFileNameExt("10.11.111会议纪要（1）.docx"))
	t.Log("very good")
}
