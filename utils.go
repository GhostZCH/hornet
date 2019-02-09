package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Logger struct {
	info       *log.Logger
	warn       *log.Logger
	err        *log.Logger
	access     *log.Logger
	runFile    *os.File
	accessFile *os.File
}

var logger *Logger

var levelMap = map[string]int{"info": 1, "warn": 2, "error": 3}

var GConfig = make(map[string]interface{})

func InitLog() {
	var err error

	if logger != nil {
		if logger.runFile != nil {
			logger.runFile.Close()
		}
		if logger.accessFile != nil {
			logger.accessFile.Close()
		}
		logger.info = nil
		logger.warn = nil
		logger.err = nil
		logger.access = nil
	} else {
		logger = new(Logger)
	}

	// init run log
	path := GConfig["runlog.path"].(string)
	logger.runFile, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	var level = GConfig["runlog.level"].(string)
	var lv, ok = levelMap[level]
	if !ok {
		panic("level [" + level + "] not \"error\" \"warn\" or \"info\"")
	}

	if lv <= 3 {
		logger.err = log.New(logger.runFile, "[ERROR] ", log.LstdFlags)
	}

	if lv <= 2 {
		logger.warn = log.New(logger.runFile, "[WARN] ", log.LstdFlags)
	}

	if lv <= 1 {
		logger.info = log.New(logger.runFile, "[INFO] ", log.LstdFlags)
	}

	// init access log
	path = GConfig["accesslog.path"].(string)
	logger.accessFile, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	logger.access = log.New(logger.accessFile, "", log.LstdFlags)
}

func Info(v ...interface{}) {
	if logger.info != nil {
		logger.info.Println(v...)
	}
}

func Warn(v ...interface{}) {
	if logger.warn != nil {
		logger.warn.Println(v...)
	}
}

func Error(v ...interface{}) {
	if logger.err != nil {
		logger.err.Println(v...)
	}
}

func Access(r *Request) {
	if logger.access != nil {
		logger.access.Println(r)
	}
}

func readYaml(path string) (conf map[string]interface{}) {
	var err error
	var content []byte

	if content, err = ioutil.ReadFile(path); err != nil {
		if os.IsNotExist(err) {
			return conf
		}
		panic(err)
	}

	if err = yaml.Unmarshal(content, &conf); err != nil {
		panic(err)
	}

	return conf
}

func LoadConf(path string, localPath string) {
	GConfig = readYaml(path)
	lconf := readYaml(localPath)

	for k, v := range lconf {
		GConfig[k] = v
	}
}
