package logger

import (
	"fmt"
	"io"
	"os"
)

type logger struct {
	LogLevel   int
	FilePath   string
	MaxSizeMB  int
	MaxBackups int
}

// initGlobalLogger 封装了创建和设置全局日志记录器的逻辑
// 它会配置默认的 logger，使其同时输出到控制台和轮转文件
func InitGlobalLogger(c logger) (io.Closer, error) {
	// 1. 设置日志轮转
	logFile, err := NewRotator(c.FilePath, int64(c.MaxSizeMB)*1024*1024, c.MaxBackups)
	if err != nil {
		return nil, fmt.Errorf("创建日志轮转文件失败: %v", err)
	}

	// 2. 创建一个将日志写入多个位置的 writer
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// 3. 配置全局的默认 logger
	level := Level(c.LogLevel)
	SetLevel(level)
	SetOutput(multiWriter)
	SetFormatter(&JSONFormatter{})

	// 返回 closer 以便在程序结束时关闭文件
	return logFile, nil
}
