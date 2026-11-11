package health

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed ui/*
var uiFS embed.FS

func registerUI(r *gin.Engine) {
	sub, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return
	}
	r.GET("/health", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/health/ui/")
	})
	serve := func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("filepath"), "/")
		if name == "" || strings.HasSuffix(c.Request.URL.Path, "/") {
			name = "index.html"
		}
		data, err := fs.ReadFile(sub, name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		ctype := mime.TypeByExtension(path.Ext(name))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		if name == "index.html" {
			ctype = "text/html; charset=utf-8"
		}
		c.Data(http.StatusOK, ctype, data)
	}
	r.GET("/health/ui", serve)
	r.GET("/health/ui/*filepath", serve)
}
