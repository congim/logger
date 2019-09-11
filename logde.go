package logde

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"time"
)

const (
	TimeDivision = "time"
	SizeDivision = "size"

	_defaultEncoding = "console"
	_defaultDivision = "size"
	_defaultUnit     = Hour
)

var (
	Logger                    *Log
	_encoderNameToConstructor = map[string]func(zapcore.EncoderConfig) zapcore.Encoder{
		"console": func(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
			return zapcore.NewConsoleEncoder(encoderConfig)
		},
		"json": func(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
			return zapcore.NewJSONEncoder(encoderConfig)
		},
	}
)

type Log struct {
	L *zap.Logger
}

type LogOptions struct {
	// Encoding sets the logger's encoding. Valid values are "json" and
	// "console", as well as any third-party encodings registered via
	// RegisterEncoder.
	Encoding      string   `json:"encoding" yaml:"encoding" toml:"encoding"`
	InfoFilename  string   `json:"info_filename" yaml:"info_filename" toml:"info_filename"`
	ErrorFilename string   `json:"error_filename" yaml:"error_filename" toml:"error_filename"`
	MaxSize       int      `json:"max_size" yaml:"max_size" toml:"max_size"`
	MaxBackups    int      `json:"max_backups" yaml:"max_backups" toml:"max_backups"`
	MaxAge        int      `json:"max_age" yaml:"max_age" toml:"max_age"`
	Compress      bool     `json:"compress" yaml:"compress" toml:"compress"`
	Division      string   `json:"division" yaml:"division" toml:"division"`
	LevelSeparate bool     `json:"level_separate" yaml:"level_separate" toml:"level_separate"`
	TimeUnit      TimeUnit `json:"time_unit" yaml:"time_unit" toml:"time_unit"`

	caller bool
}

func infoLevel() zap.LevelEnablerFunc {
	return zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	})
}

func warnLevel() zap.LevelEnablerFunc {
	return zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})
}

func New() *LogOptions {
	return &LogOptions{
		Division:      _defaultDivision,
		LevelSeparate: false,
		TimeUnit:      _defaultUnit,
		Encoding:      _defaultEncoding,
		caller:        false,
	}
}

func NewFromToml(confPath string) *LogOptions {
	var c *LogOptions
	if _, err := toml.DecodeFile(confPath, &c); err != nil {
		panic(err)
	}
	return c
}

func NewFromYaml(confPath string) *LogOptions {
	var c *LogOptions
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(file, &c)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	return c
}

func NewFromJson(confPath string) *LogOptions {
	var c *LogOptions
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}
	err = json.Unmarshal(file, &c)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	return c
}

func (c *LogOptions) SetDivision(division string) {
	c.Division = division
}

func (c *LogOptions) SetCaller(b bool) {
	c.caller = b
}

func (c *LogOptions) SetTimeUnit(t TimeUnit) {
	c.TimeUnit = t
}

func (c *LogOptions) SetErrorFile(path string) {
	c.LevelSeparate = true
	c.ErrorFilename = path
}

func (c *LogOptions) SetInfoFile(path string) {
	c.InfoFilename = path
}

func (c *LogOptions) SetEncoding(encoding string) {
	c.Encoding = encoding
}

// isOutput whether set output file
func (c *LogOptions) isOutput() bool {
	return c.InfoFilename != ""
}

func (c *LogOptions) InitLogger() *Log {
	var (
		core               zapcore.Core
		infoHook, warnHook io.Writer
		wsInfo             []zapcore.WriteSyncer
		wsWarn             []zapcore.WriteSyncer
	)

	if c.Encoding == "" {
		c.Encoding = _defaultEncoding
	}
	encoder := _encoderNameToConstructor[c.Encoding]

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "file",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
	}

	wsInfo = append(wsInfo, zapcore.AddSync(os.Stdout))
	wsWarn = append(wsWarn, zapcore.AddSync(os.Stdout))

	// zapcore WriteSyncer setting
	if c.isOutput() {
		switch c.Division {
		case TimeDivision:
			infoHook = c.timeDivisionWriter(c.InfoFilename)
			if c.LevelSeparate {
				warnHook = c.timeDivisionWriter(c.ErrorFilename)
			}
		case SizeDivision:
			infoHook = c.sizeDivisionWriter(c.InfoFilename)
			if c.LevelSeparate {
				warnHook = c.sizeDivisionWriter(c.ErrorFilename)
			}
		}
		wsInfo = append(wsInfo, zapcore.AddSync(infoHook))
	}

	if c.ErrorFilename != "" {
		wsWarn = append(wsWarn, zapcore.AddSync(warnHook))
	}

	// Separate info and warning log
	if c.LevelSeparate {
		core = zapcore.NewTee(
			zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsInfo...), infoLevel()),
			zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsWarn...), warnLevel()),
		)
	} else {
		core = zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsInfo...), zap.InfoLevel)
	}

	// file line number display
	development := zap.Development()
	// init default key
	//filed := zap.Fields(zap.String("serviceName", "serviceName"))
	var logger *zap.Logger
	if c.caller {
		logger = zap.New(core, zap.AddCaller(), development)
	} else {
		logger = zap.New(core, development)
	}

	Logger = &Log{logger}
	return Logger
}

func (c *LogOptions) sizeDivisionWriter(filename string) io.Writer {
	hook := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxSize,
		Compress:   c.Compress,
	}
	return hook
}

func (c *LogOptions) timeDivisionWriter(filename string) io.Writer {
	hook, err := rotatelogs.New(
		filename+c.TimeUnit.Format(),
		rotatelogs.WithMaxAge(time.Duration(int64(24*time.Hour)*int64(c.MaxAge))),
		rotatelogs.WithRotationTime(c.TimeUnit.RotationGap()),
	)

	if err != nil {
		panic(err)
	}
	return hook
}

func (logger *Log) Info(msg string, args ...zap.Field) {
	logger.L.Info(msg, args...)
}

func (logger *Log) Error(msg string, args ...zap.Field) {
	logger.L.Error(msg, args...)
}

func (logger *Log) Warn(msg string, args ...zap.Field) {
	logger.L.Warn(msg, args...)
}

func (logger *Log) Debug(msg string, args ...zap.Field) {
	logger.L.Debug(msg, args...)
}

func (logger *Log) Fatal(msg string, args ...zap.Field) {
	logger.L.Fatal(msg, args...)
}

func (logger *Log) WithAny(k string, v interface{}) zap.Field {
	return zap.Any(k, v)
}

func (logger *Log) With(k string, v string) zap.Field {
	return zap.String(k, v)
}

func (logger *Log) WithError(k string, err error) zap.Field {
	return zap.NamedError(k, err)
}