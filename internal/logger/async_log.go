package logger

import (
	"XMedia/internal/utils"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logDefaultMaxSize   = 100  // 日志文件最大尺寸，单位 MB
	logDefaultMaxBackup = 5    // 最大备份日志文件数
	logDefaultQueueSize = 1000 // channel 队列长度
	logDefaultSaveDays  = 7    // 默认保存天数
)

type logMessage struct {
	level    string
	category string
	message  string
}

type AsyncLogQueue struct {
	logPrefix    string
	logMaxSize   int
	logMaxBackup int
	logQueueSize int
	logSaveDays  int
	exePath      string

	loggers   map[string]*log.Logger
	chanQueue chan logMessage
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

type AsyncLogQueueOption func(*AsyncLogQueue)

func WithLogMaxSize(logMaxSize int) AsyncLogQueueOption {
	return func(f *AsyncLogQueue) {
		f.logMaxSize = logMaxSize
	}
}

func WithLogMaxBackup(logMaxBackup int) AsyncLogQueueOption {
	return func(f *AsyncLogQueue) {
		f.logMaxBackup = logMaxBackup
	}
}

func WithLogQueueSize(logQueueSize int) AsyncLogQueueOption {
	return func(f *AsyncLogQueue) {
		f.logQueueSize = logQueueSize
	}
}

func WithLogSaveDays(logSaveDays int) AsyncLogQueueOption {
	return func(f *AsyncLogQueue) {
		f.logSaveDays = logSaveDays
	}
}

func NewAsyncLogQueue(product string, opts ...AsyncLogQueueOption) (*AsyncLogQueue, error) {
	logQueue := &AsyncLogQueue{
		logPrefix:    product,
		logMaxSize:   logDefaultMaxSize,
		logMaxBackup: logDefaultMaxBackup,
		logQueueSize: logDefaultQueueSize,
		logSaveDays:  logDefaultSaveDays,
		loggers:      make(map[string]*log.Logger),
	}

	for _, opt := range opts {
		opt(logQueue)
	}

	err := logQueue.initLog()
	if err != nil {
		return nil, err
	}

	return logQueue, nil
}

func (a *AsyncLogQueue) Stop() {
	close(a.chanQueue)
	a.wg.Wait()
}

func (a *AsyncLogQueue) Add(msg logMessage) {
	a.chanQueue <- msg
}

// 注册一个新的日志类别
func (a *AsyncLogQueue) RegisterCategory(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	logpath := filepath.Join(filepath.Dir(a.exePath), "logs", name)
	utils.EnsureDir(logpath)

	logFilePath := filepath.Join(logpath, fmt.Sprintf("%s.%s.log", a.logPrefix, name))
	rotateLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    a.logMaxSize,
		MaxBackups: a.logMaxBackup,
		MaxAge:     a.logSaveDays,
		Compress:   false,
	}

	writer := io.MultiWriter(os.Stdout, rotateLogger)
	logger := log.New(writer, a.logPrefix+" ", log.Ldate|log.Ltime|log.Lshortfile)

	a.loggers[name] = logger
	return nil
}

func (a *AsyncLogQueue) handleLog(msg logMessage) {
	a.mu.RLock()
	logger, ok := a.loggers[msg.category]
	a.mu.RUnlock()

	if ok && logger != nil {
		logger.Output(2, fmt.Sprintf("[%s] %s", msg.level, msg.message))
	} else {
		// fallback 默认写 info
		if l, ok := a.loggers["info"]; ok {
			l.Output(2, fmt.Sprintf("[%s] %s", msg.level, msg.message))
		}
	}
}

func (a *AsyncLogQueue) LogInfoWithCategory(category string, format string, args ...interface{}) {
	a.Add(logMessage{level: "INFO", category: category, message: fmt.Sprintf(format, args...)})
}

func (a *AsyncLogQueue) LogWarnWithCategory(category string, format string, args ...interface{}) {
	a.Add(logMessage{level: "WARN", category: category, message: fmt.Sprintf(format, args...)})
}

func (a *AsyncLogQueue) LogErrorWithCategory(category string, format string, args ...interface{}) {
	a.Add(logMessage{level: "ERROR", category: category, message: fmt.Sprintf(format, args...)})
}

// Log implements log.Writer.
func (a *AsyncLogQueue) Log(level Level, format string, args ...interface{}) {
	switch level {
	case Info:
		a.asyncLogInfo(fmt.Sprintf(format, args...))
	case Warn:
		a.asyncLogWarn(fmt.Sprintf(format, args...))
	case Error:
		a.asyncLogError(fmt.Sprintf(format, args...))
	}
}

func (a *AsyncLogQueue) asyncLogInfo(msg string) {
	a.Add(logMessage{level: "INFO", category: "info", message: msg})
}

func (a *AsyncLogQueue) asyncLogWarn(msg string) {
	a.Add(logMessage{level: "WARN", category: "info", message: msg})
}

func (a *AsyncLogQueue) asyncLogError(msg string) {
	a.Add(logMessage{level: "ERROR", category: "error", message: msg})
}

func (a *AsyncLogQueue) initLog() error {
	a.chanQueue = make(chan logMessage, a.logQueueSize)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("AsyncLogQueue.initLog error:%v", err)
	}
	a.exePath = exePath

	// 默认注册 info / error 日志
	if err := a.RegisterCategory("info"); err != nil {
		return err
	}
	if err := a.RegisterCategory("error"); err != nil {
		return err
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for msg := range a.chanQueue {
			a.handleLog(msg)
		}
	}()

	return nil
}

// 返回某个分类的 *log.Logger
func (a *AsyncLogQueue) Logger(category string) *log.Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.loggers[category]
}
