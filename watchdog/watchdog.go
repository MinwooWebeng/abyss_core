package watchdog

import (
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/phuslu/log"
)

var handle_count atomic.Int32
var handle_count_print_threshold atomic.Int32
var null_handle_closed_count atomic.Int32

func Init() {
	err := os.MkdirAll("log", os.ModePerm)
	if err != nil {
		panic(err)
	}
	logFile, err := os.OpenFile("log/"+time.Now().Format("0102_150405")+"_abyss_net.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	log.DefaultLogger = log.Logger{
		Level: log.InfoLevel,

		Writer: &log.ConsoleWriter{
			Writer: logFile,
		},
		// Writer: &log.FileWriter{
		// 	Filename:     ,
		// 	FileMode:     0600,
		// 	MaxSize:      100 * 1024 * 1024, //100MB
		// 	MaxBackups:   3,
		// 	EnsureFolder: true,
		// 	LocalTime:    true,
		// },
	}

	handle_count_print_threshold.Store(4)
	Info("start")
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

func CountHandleExport() {
	handle_count.Add(1)

	current_threshhold := handle_count_print_threshold.Load()
	handle_count := handle_count.Load()

	if handle_count > current_threshhold*2 {
		handle_count_print_threshold.Store(current_threshhold * 2)
		Info("open handle count: " + strconv.Itoa(int(handle_count)))
	}
}
func CountHandleRelease() {
	handle_count.Add(-1)

	current_threshhold := handle_count_print_threshold.Load()
	handle_count := handle_count.Load()
	if handle_count < current_threshhold/2 && current_threshhold > 4 {
		handle_count_print_threshold.Store(current_threshhold / 2)
		Info("open handle count: " + strconv.Itoa(int(handle_count)))
	}
}
func CountNullHandleRelease() {
	null_handle_closed_count.Add(1)
	Warn("null handle close: " + strconv.Itoa(int(null_handle_closed_count.Load())))
}
