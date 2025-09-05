// Package logger 提供了日志文件轮转的功能。
package logger

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogRotator 实现了 io.WriteCloser 接口，用于按大小和时间轮转日志文件。
type LogRotator struct {
	mu          sync.Mutex
	filename    string
	maxSize     int64 // 以字节为单位
	maxBackups  int
	maxAgeDays  int
	currentSize int64
	file        *os.File
}

// NewRotator 创建一个新的 LogRotator 实例。
// filename: 日志文件的路径。
// maxSize: 单个文件的最大大小（字节）。
// maxBackups: 要保留的旧日志文件的最大数量。
// maxAgeDays: 要保留旧日志文件的最大天数。
func NewRotator(filename string, maxSize int64, maxBackups int, maxAgeDays int) (*LogRotator, error) {
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024 // 默认 100MB
	}
	if maxBackups <= 0 {
		maxBackups = 30 // 默认 30 个备份
	}
	if maxAgeDays <= 0 {
		maxAgeDays = 7 // 默认 7 天
	}

	r := &LogRotator{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAgeDays: maxAgeDays,
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

	// 2. 将当前日志文件重命名为带时间戳的新文件
	backupName := r.backupFilename()
	if err := os.Rename(r.filename, backupName); err != nil {
		return err
	}

	// 3. 清理旧文件
	if err := r.clean(); err != nil {
		return err
	}

	// 4. 创建一个新的日志文件
	return r.openFile()
}

// clean 清理旧的备份文件，根据 maxBackups 和 maxAgeDays 进行判断。
func (r *LogRotator) clean() error {
	files, err := r.findOldLogFiles()
	if err != nil {
		return err
	}

	// 排序以便按时间或编号删除
	sort.Slice(files, func(i, j int) bool {
		return files[i].info.ModTime().Before(files[j].info.ModTime())
	})

	for i, f := range files {
		// 根据 maxAgeDays 判断是否过期
		if r.maxAgeDays > 0 && time.Since(f.info.ModTime()) > time.Duration(r.maxAgeDays)*24*time.Hour {
			os.Remove(f.path)
			continue
		}

		// 根据 maxBackups 判断是否超限
		if r.maxBackups > 0 && len(files)-i > r.maxBackups {
			os.Remove(f.path)
			continue
		}
	}

	return nil
}

type logFile struct {
	path string
	info fs.FileInfo
}

// findOldLogFiles 查找并返回所有旧的日志文件
func (r *LogRotator) findOldLogFiles() ([]logFile, error) {
	dir := filepath.Dir(r.filename)
	base := filepath.Base(r.filename)
	var files []logFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), base+".") && path != r.filename {
			files = append(files, logFile{path: path, info: info})
		}
		return nil
	})
	return files, err
}

// backupFilename 生成带时间戳的备份文件名
func (r *LogRotator) backupFilename() string {
	ext := filepath.Ext(r.filename)
	name := strings.TrimSuffix(r.filename, ext)
	return fmt.Sprintf("%s-%s%s", name, time.Now().Format("2006-01-02T15-04-05"), ext)
}
