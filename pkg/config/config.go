package config

import (
	"golang.org/x/text/language"
)

var (
	AppName   = "dootask"
	Version   = "develop"
	CommitSHA = "0000000"

	Language        = []string{language.Chinese.String(), language.TraditionalChinese.String(), language.English.String(), language.Korean.String(), language.Japanese.String(), language.German.String(), language.French.String(), language.Indonesian.String()}
	YoudaoAppKey    = "YOUDAO_APP_KEY"
	YoudaoAppSecret = "YOUDAO_SEC_KEY"

	DooTaskUrl = "http://10.98.101.3"
)
