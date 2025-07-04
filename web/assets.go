package web

import (
	"embed"
)

// 嵌入web目录下的静态资源和模板文件
//
//go:embed static templates
var EmbeddedAssets embed.FS
