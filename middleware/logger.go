package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"one-api/common"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	maxSize = 1024
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.ResponseLogEnabled || c.Request.Body == nil {
			c.Next()
			return
		}
		bodyBytes, err := common.GetRequestBody(c)
		if err != nil {
			common.SysError("error reading request body: " + err.Error())
			c.Next()
			return
		}

		if len(bodyBytes) > 0 {
			bodyToLog := string(bodyBytes)
			if len(bodyToLog) > maxSize {
				bodyToLog = bodyToLog[:maxSize] + "..."
			}
			msg := fmt.Sprintf("%s %s %s\n%s",
				c.ClientIP(),
				c.Request.Method,
				c.Request.URL.Path,
				bodyToLog)
			common.LogInfo(c.Request.Context(), msg)
		}
		c.Next()
	}
}

func SetUpLogger(server *gin.Engine) {
	server.Use(RequestLogger())
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			requestID = param.Keys[common.RequestIdKey].(string)
		}

		logStr := fmt.Sprintf("[GIN] %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			requestID,
			param.StatusCode,
	@@ -21,5 +61,39 @@ func SetUpLogger(server *gin.Engine) {
			param.Method,
			param.Path,
		)

		logStr += responseLog(param)

		return logStr
	}))

	// add a middleware to log the response body when gin detail enabled
	server.Use(func(c *gin.Context) {
		c = common.WrapWriter(c)
		c.Next()
	})
}

func responseLog(param gin.LogFormatterParams) string {
	if !common.DebugEnabled {
		return ""
	}
	blw, ok := param.Keys[common.KeyResponseWriter].(common.BodyLogWriter)
	if !ok {
		return ""
	}
	body := blw.String()
	// decompressed gzip response body
	if strings.Contains(param.Request.Header.Get("Accept-Encoding"), "gzip") {
		if reader, err := gzip.NewReader(bytes.NewReader([]byte(body))); err == nil {
			if decompressed, err := io.ReadAll(reader); err == nil {
				body = string(decompressed)
			}
		}
	}

	if len(body) > maxSize {
		body = string(body[:maxSize]) + "..."
	}
	return fmt.Sprintf("%s\n", body)
}
