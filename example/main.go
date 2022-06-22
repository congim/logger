package main

import (
	"time"

	"github.com/congim/logger"
)

func main() {
	c := logger.New()
	c.SetDivision("time")     // 设置归档方式，"time"时间归档 "size" 文件大小归档，文件大小等可以在配置文件配置
	c.SetTimeUnit(logger.Day) // 时间归档 可以设置切割单位
	c.SetEncoding("json")     // 输出格式 "json" 或者 "console"

	c.SetInfoFile("./logs/server.log")  // 设置info级别日志
	c.SetErrorFile("./logs/server.log") // 设置warn级别日志

	c.InitLogger()

	for {
		logger.Info("info level test")
		logger.Error("error level test")
		logger.Warn("warn level test")
		logger.Debug("debug level test")
		//logger.Fatal("fatal level test")

		// format
		logger.Infof("info level test: %s", "111")
		logger.Errorf("error level test: %s", "111")
		logger.Warnf("warn level test: %s", "111")
		logger.Debugf("debug level test: %s", "111")
		time.Sleep(1 * time.Second)
	}

}
