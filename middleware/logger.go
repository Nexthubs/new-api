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

// 请用这个函数完整替换掉你原来的 SetUpLogger 函数

func SetUpLogger(server *gin.Engine) {
	server.Use(RequestLogger())
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			// 安全地获取 requestID，避免 panic
			if val, ok := param.Keys[common.RequestIdKey].(string); ok {
				requestID = val
			}
		}

		// 修正后的 Sprintf，移除了所有的冲突标记
		logStr := fmt.Sprintf("[GIN] %s | %s | %3d | %13v | %15s | %-7s %s\n", // 使用 %-7s 让方法左对齐，更美观
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			requestID,
			param.StatusCode,
			param.Latency, // 响应延迟
			param.ClientIP,
			param.Method,
			param.Path,
		)

		if param.ErrorMessage != "" {
			logStr += fmt.Sprintf("| %s", param.ErrorMessage)
		}

		logStr += responseLog(param)

		return logStr
	}))

	// add a middleware to log the response body when gin detail enabled
	server.Use(func(c *gin.Context) {
		// 确保在执行 c.Next() 之前包装 writer
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
