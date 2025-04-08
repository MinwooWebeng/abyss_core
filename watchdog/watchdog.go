package watchdog

import (
	"time"

	"github.com/phuslu/log"
)

func Init() {
	log.DefaultLogger = log.Logger{
		Level: log.InfoLevel,
		Writer: &log.FileWriter{
			Filename:     "log/" + time.Now().Format("20060102_150405") + "_abyss_net.log",
			FileMode:     0600,
			MaxSize:      100 * 1024 * 1024, //100MB
			MaxBackups:   3,
			EnsureFolder: true,
			LocalTime:    true,
		},
	}
}

func Info(msg string) {
	log.Info().Msg(msg)
}

func Warn(msg string) {
	log.Warn().Msg(msg)
}

func Error(err error) {
	log.Error().Msg(err.Error())
}

func Fatal(msg string) {
	log.Fatal().Msg(msg)
}
