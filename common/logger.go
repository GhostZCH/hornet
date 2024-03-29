package common

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type HourlyLogger struct {
	dir    string
	logger zerolog.Logger
	file   *os.File
	ch     chan *LogData
}

type LogData struct {
	Hit   bool
	Url   []byte
	Level int
}

func newLogger(dir string) *HourlyLogger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// 如果日志文件夹不存在，则创建文件夹
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		Success(os.MkdirAll(dir, os.ModePerm))
	}

	l := &HourlyLogger{dir: dir, file: nil}
	l.update()

	l.ch = make(chan *LogData, 2000)

	go func() {
		for msg := range l.ch {
			l.logger.Info().Bytes("path", msg.Url).Bool("hit", msg.Hit).Int("level", msg.Level).Msg("")
		}
	}()

	return l
}

func (l *HourlyLogger) WriteLog(msg *LogData) {
	l.ch <- msg
}

func (l *HourlyLogger) update() {
	if l.file != nil {
		Success(l.file.Close())
	}

	filename := l.dir + time.Now().Format("2006-01-02-15.log")

	var err error
	l.file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	Success(err)

	l.logger = zerolog.New(l.file).With().Logger()
}

func NewHourlyLogger(dir string) *HourlyLogger {
	l := newLogger(dir)

	ticker := time.NewTicker(time.Duration(time.Until(time.Now().Add(time.Hour)).Seconds()) * time.Second)
	go func() {
		for range ticker.C {
			l.update()

			next := time.Now().Add(time.Hour).Truncate(time.Hour)
			duration := time.Duration(time.Until(next).Seconds()) * time.Second
			ticker = time.NewTicker(duration)
		}
	}()

	return l
}
