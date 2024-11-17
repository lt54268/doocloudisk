package config

import (
	"os"

	"golang.org/x/text/language"
)

var (
	AppName   = "dootask"
	Version   = "develop"
	CommitSHA = "0000000"

	Language        = []string{language.Chinese.String(), language.TraditionalChinese.String(), language.English.String(), language.Korean.String(), language.Japanese.String(), language.German.String(), language.French.String(), language.Indonesian.String()}
	YoudaoAppKey    = "YOUDAO_APP_KEY"
	YoudaoAppSecret = "YOUDAO_SEC_KEY"

	DooTaskUrl         = "http://10.98.101.3"
	Port               = os.Getenv("PORT")                  // 从环境变量读取端口
	OssRegion          = os.Getenv("OSS_REGION")            // 从环境变量读取区域
	OssEndpoint        = os.Getenv("OSS_ENDPOINT")          // 从环境变量读取 Endpoint
	OssBucket          = os.Getenv("OSS_BUCKET")            // 从环境变量读取 Bucket
	OssAccessKeyId     = os.Getenv("OSS_ACCESS_KEY_ID")     // 从环境变量读取 AccessKeyId
	OssAccessKeySecret = os.Getenv("OSS_ACCESS_KEY_SECRET") // 从环境变量读取 AccessKeySecret

)
