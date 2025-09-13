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
	logDefaultMaxSize   = 100 // 日志文件最大尺寸，单位 MB
	logDefaultMaxBackup = 5   // 最大备份日志文件数
	logDefaultQueueSize = 1000
)

type logMessage struct {
	level   string
	message string
}

type AsyncLogQueue struct {
	logPrefix    string
	logMaxSize   int
	logMaxBackup int
	logQueueSize int

	infoLog      *log.Logger
	errorLog     *log.Logger
	infoLogName  string
	errorLogName string

	chanQueue chan logMessage
	wg        sync.WaitGroup
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

func NewAsyncLogQueue(product string, opts ...AsyncLogQueueOption) (q *AsyncLogQueue, err error) {
	logQueue := &AsyncLogQueue{
		logPrefix:    product + " ",
		infoLogName:  fmt.Sprintf("%s.log", product),
		errorLogName: fmt.Sprintf("%s.error.log", product),
		logMaxSize:   logDefaultMaxSize,
		logMaxBackup: logDefaultMaxBackup,
		logQueueSize: logDefaultQueueSize,
	}
	for _, opt := range opts {
		opt(logQueue)
	}

	err = logQueue.initLog()
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

func (a *AsyncLogQueue) handleLog(msg logMessage) {
	switch msg.level {
	case "INFO", "WARN":
		if a.infoLog != nil {
			a.infoLog.Output(2, msg.message)
		}
	case "ERROR":
		if a.errorLog != nil {
			a.errorLog.Output(2, msg.message)
		}
	}
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
	a.Add(logMessage{level: "INFO", message: msg})
}

func (a *AsyncLogQueue) asyncLogWarn(msg string) {
	a.Add(logMessage{level: "WARN", message: msg})
}

func (a *AsyncLogQueue) asyncLogError(msg string) {
	a.Add(logMessage{level: "ERROR", message: msg})
}

func (a *AsyncLogQueue) initLog() (err error) {

	a.chanQueue = make(chan logMessage, a.logQueueSize)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("AsyncLogQueue.initLog error:%v", err)
	}
	err = a.openFile(a.infoLogName, exePath, true)
	if err != nil {
		return fmt.Errorf("AsyncLogQueue.initLog error:%v", err)
	}
	err = a.openFile(a.errorLogName, exePath, false)
	if err != nil {
		return fmt.Errorf("AsyncLogQueue.initLog error:%v", err)
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

func (a *AsyncLogQueue) openFile(name, exePath string, info bool) error {
	logpath := filepath.Join(filepath.Dir(exePath), "logs")
	utils.EnsureDir(logpath)

	logFilePath := filepath.Join(logpath, name)
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	if info {
		a.infoLog = log.New(&lumberjack.Logger{
			Filename:   name,
			MaxSize:    a.logMaxSize,   // megabytes
			MaxBackups: a.logMaxBackup, // number of files
		}, a.logPrefix, log.Ldate|log.Ltime|log.Lshortfile)

		a.infoLog.SetOutput(io.MultiWriter(os.Stdout, file))
		return nil
	}
	a.errorLog = log.New(&lumberjack.Logger{
		Filename:   name,
		MaxSize:    a.logMaxSize,   // megabytes
		MaxBackups: a.logMaxBackup, // number of files
	}, a.logPrefix, log.Ldate|log.Ltime|log.Lshortfile)
	a.errorLog.SetOutput(io.MultiWriter(os.Stdout, file))
	return nil
}

func (a *AsyncLogQueue) Logger(level Level) *log.Logger {
	switch level {
	case Info, Warn:
		return a.infoLog
	case Error:
		return a.errorLog
	}
	return nil
}
