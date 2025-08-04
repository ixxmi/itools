// Package rotator 提供了日志文件轮转的功能。
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// LogRotator 实现了 io.WriteCloser 接口，用于按大小轮转日志文件。
type LogRotator struct {
	mu          sync.Mutex
	filename    string
	maxSize     int64 // 以字节为单位
	maxBackups  int
	currentSize int64
	file        *os.File
}

// New 创建一个新的 LogRotator 实例。
// filename: 日志文件的路径。
// maxSize: 单个文件的最大大小（字节）。
// maxBackups: 要保留的旧日志文件的最大数量。
func NewRotator(filename string, maxSize int64, maxBackups int) (*LogRotator, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("maxSize 必须大于 0")
	}
	if maxBackups < 0 {
		return nil, fmt.Errorf("maxBackups 不能为负数")
	}

	r := &LogRotator{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
	}

	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, err
	}

	// 打开或创建日志文件
	err := r.openFile()
	if err != nil {
		return nil, err
	}

	return r, nil
}

// openFile 打开日志文件并获取其当前大小。
func (r *LogRotator) openFile() error {
	file, err := os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	r.file = file
	r.currentSize = stat.Size()
	return nil
}

// Write 实现了 io.Writer 接口。
func (r *LogRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否需要轮转
	if r.currentSize+int64(len(p)) > r.maxSize {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = r.file.Write(p)
	r.currentSize += int64(n)
	return n, err
}

// Close 实现了 io.Closer 接口。
func (r *LogRotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Close()
}

// rotate 执行文件轮转。
func (r *LogRotator) rotate() error {
	// 1. 关闭当前文件
	if err := r.file.Close(); err != nil {
		return err
	}

	// 2. 重命名备份文件
	for i := r.maxBackups; i > 0; i-- {
		oldPath := r.backupFilename(i - 1)
		newPath := r.backupFilename(i)

		// 检查旧文件是否存在
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// 3. 重命名当前日志文件为第一个备份
	if err := os.Rename(r.filename, r.backupFilename(0)); err != nil {
		return err
	}

	// 4. 创建一个新的日志文件
	return r.openFile()
}

// backupFilename 生成备份文件的名称。
func (r *LogRotator) backupFilename(num int) string {
	if num == 0 {
		return r.filename + ".1"
	}
	return fmt.Sprintf("%s.%d", r.filename, num+1)
}
