// Package logger提供了功能丰富的日志记录功能。
package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// --- 日志级别 ---

// Level 定义了日志的级别类型
type Level uint8

// 定义了所有支持的日志级别
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// levelToString 将日志级别转换为字符串
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// --- 字段和条目 ---

// Fields 是用于结构化日志的键值对类型
type Fields map[string]interface{}

// Entry 代表一个日志条目
type Entry struct {
	Logger    *Logger
	Time      time.Time
	Level     Level
	Message   string
	Fields    Fields
	File      string
	Line      int
	Func      string
	callDepth int
}

// WithFields 为日志条目添加结构化字段
func (e *Entry) WithFields(fields Fields) *Entry {
	newFields := make(Fields, len(e.Fields)+len(fields))
	for k, v := range e.Fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	e.Fields = newFields
	return e
}

// logf 格式化并记录日志
func (e *Entry) logf(format string, args ...interface{}) {
	e.Message = fmt.Sprintf(format, args...)
	e.Logger.log(e)
}

// log 记录日志
func (e *Entry) log(args ...interface{}) {
	e.Message = fmt.Sprint(args...)
	e.Logger.log(e)
}

// --- 格式化器 ---

// Formatter 是日志格式化器的接口
type Formatter interface {
	Format(*Entry) ([]byte, error)
}

// TextFormatter 将日志格式化为纯文本
type TextFormatter struct{}

// Format 实现 Formatter 接口
func (f *TextFormatter) Format(e *Entry) ([]byte, error) {
	var fieldsStr string
	for k, v := range e.Fields {
		fieldsStr += fmt.Sprintf(" %s=%v", k, v)
	}

	return []byte(fmt.Sprintf("[%s] [%s] [%s:%d] %s%s\n",
		e.Time.Format("2006-01-02 15:04:05"),
		e.Level.String(),
		e.File,
		e.Line,
		e.Message,
		fieldsStr,
	)), nil
}

// JSONFormatter 将日志格式化为 JSON
type JSONFormatter struct{}

// Format 实现 Formatter 接口
func (f *JSONFormatter) Format(e *Entry) ([]byte, error) {
	// 使用一个 map 来构建 JSON，以避免复杂的字符串拼接
	data := make(Fields, len(e.Fields)+5)
	data["time"] = e.Time.Format(time.RFC3339)
	data["level"] = e.Level.String()
	data["message"] = e.Message
	data["file"] = fmt.Sprintf("%s:%d", e.File, e.Line)
	data["func"] = e.Func

	for k, v := range e.Fields {
		// 避免覆盖核心字段
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}

	// 手动序列化为 JSON 字符串，以避免引入额外的依赖
	var parts []string
	for k, v := range data {
		parts = append(parts, fmt.Sprintf("\"%s\":%q", k, fmt.Sprintf("%v", v)))
	}

	return []byte("{" + strings.Join(parts, ",") + "}\n"), nil
}

// --- Logger ---

// Logger 是日志记录器的核心结构
type Logger struct {
	out       io.Writer
	level     Level
	formatter Formatter
	mu        sync.Mutex
}

// Option 是用于配置 Logger 的函数类型
type Option func(*Logger)

// New 创建一个新的 Logger 实例
func New(opts ...Option) *Logger {
	logger := &Logger{
		out:       os.Stdout,
		level:     InfoLevel,
		formatter: &TextFormatter{},
	}

	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

// WithOutput 设置输出目标
func WithOutput(out io.Writer) Option {
	return func(l *Logger) {
		l.out = out
	}
}

// WithLevel 设置日志级别
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithFormatter 设置格式化器
func WithFormatter(formatter Formatter) Option {
	return func(l *Logger) {
		l.formatter = formatter
	}
}

// log 是内部的日志记录方法
func (l *Logger) log(entry *Entry) {
	if entry.Level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取调用信息
	if entry.callDepth == 0 {
		entry.callDepth = 3
	}
	pc, file, line, ok := runtime.Caller(entry.callDepth)
	if ok {
		entry.File = getShortPath(file)
		entry.Line = line
		entry.Func = runtime.FuncForPC(pc).Name()
	}

	entry.Time = time.Now()
	bytes, err := l.formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "格式化日志失败: %v\n", err)
		return
	}

	_, err = l.out.Write(bytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "写入日志失败: %v\n", err)
	}

	if entry.Level == FatalLevel {
		os.Exit(1)
	}
}

// newEntry 创建一个新的日志条目
func (l *Logger) newEntry() *Entry {
	return &Entry{Logger: l, Fields: make(Fields), callDepth: 3}
}

// WithFields 为 Logger 添加结构化字段，返回一个 Entry
func (l *Logger) WithFields(fields Fields) *Entry {
	return l.newEntry().WithFields(fields)
}

// --- 日志级别方法 ---

func (l *Logger) Debug(args ...interface{}) {
	l.newEntry().log(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.newEntry().logf(format, args...)
}

func (l *Logger) Info(args ...interface{}) {
	entry := l.newEntry()
	entry.Level = InfoLevel
	entry.log(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	entry := l.newEntry()
	entry.Level = InfoLevel
	entry.logf(format, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	entry := l.newEntry()
	entry.Level = WarnLevel
	entry.log(args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	entry := l.newEntry()
	entry.Level = WarnLevel
	entry.logf(format, args...)
}

func (l *Logger) Error(args ...interface{}) {
	entry := l.newEntry()
	entry.Level = ErrorLevel
	entry.log(args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	entry := l.newEntry()
	entry.Level = ErrorLevel
	entry.logf(format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	entry := l.newEntry()
	entry.Level = FatalLevel
	entry.log(args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	entry := l.newEntry()
	entry.Level = FatalLevel
	entry.logf(format, args...)
}

// --- 默认的全局 Logger ---

var defaultLogger = New()

// SetLevel 设置默认 logger 的级别
func SetLevel(level Level) {
	defaultLogger.level = level
}

// SetOutput 设置默认 logger 的输出
func SetOutput(out io.Writer) {
	defaultLogger.out = out
}

// SetFormatter 设置默认 logger 的格式化器
func SetFormatter(formatter Formatter) {
	defaultLogger.formatter = formatter
}

// 默认 logger 的快捷方法
func WithFields(fields Fields) *Entry {
	return defaultLogger.WithFields(fields)
}

func Debug(args ...interface{}) {
	defaultLogger.newEntry().log(args...)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.newEntry().logf(format, args...)
}

func Info(args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = InfoLevel
	entry.log(args...)
}

func Infof(format string, args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = InfoLevel
	entry.logf(format, args...)
}

func Warn(args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = WarnLevel
	entry.log(args...)
}

func Warnf(format string, args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = WarnLevel
	entry.logf(format, args...)
}

func Error(args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = ErrorLevel
	entry.log(args...)
}

func Errorf(format string, args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = ErrorLevel
	entry.logf(format, args...)
}

func Fatal(args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = FatalLevel
	entry.log(args...)
}

func Fatalf(format string, args ...interface{}) {
	entry := defaultLogger.newEntry()
	entry.Level = FatalLevel
	entry.logf(format, args...)
}

// getShortPath 获取文件路径的最后一部分，使其更易读
func getShortPath(file string) string {
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return file
}
