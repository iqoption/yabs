package task

import (
	"encoding/json"
	"time"
	log "github.com/sirupsen/logrus"
)

const (
	PROCESS_SYMBOLS  = 1 << iota
	PROCESS_DUMP
	PROCESS_WEB_DUMP
)

type Symbol struct {
	Type  uint   `json:"type"`
	Path  string `json:"symbol"`
	Paths []string `json:"paths"`
	Info  string `json:"info"`
}

type Dump struct {
	Type uint   `json:"type"`
	Path string `json:"minidump"`
	Info string `json:"info"`
	Log  string `json:"log"`
	Time string `json:"time,omitempty"`
}

type WebDump struct {
	Type uint   `json:"type"`
	Path string `json:"webdump"`
	Info string `json:"info"`
	Time string `json:"time,omitempty"`
}

func FromJson(data []byte) interface{} {
	type Test struct {
		Type uint `json:"type"`
	}

	var t Test
	json.Unmarshal(data, &t)
	switch t.Type {
	case PROCESS_SYMBOLS:
		var s Symbol
		err := json.Unmarshal(data, &s)
		if err != nil {
			log.WithError(err).Error("Can't parse symbol task")
			return nil
		}
		return &s
	case PROCESS_DUMP:
		var d Dump
		err := json.Unmarshal(data, &d)
		if err != nil {
			log.WithError(err).Error("Can't parse dump task")
			return nil
		}

		if len(d.Time) == 0 {
			d.Time = getTimeStamp()
		}

		return &d
	case PROCESS_WEB_DUMP:
		var w WebDump
		err := json.Unmarshal(data, &w)
		if err != nil {
			log.WithError(err).Error("Can't parse webdump task")
			return nil
		}

		if len(w.Time) == 0 {
			w.Time = getTimeStamp()
		}

		return &w
	default:
		return nil
	}
}

func CreateSymbolTask(symbol, info string) *Symbol {
	return &Symbol{Type: PROCESS_SYMBOLS,
		Path: symbol,
		Info: info}
}

func CreateSymbolsTask(symbols []string, info string) *Symbol {
	return &Symbol{Type: PROCESS_SYMBOLS,
		Paths: symbols,
		Info: info}
}

func CreateDumpTask(dump, info, log string) *Dump {
	return &Dump{Type: PROCESS_DUMP,
		Path: dump,
		Info: info,
		Log: log,
		Time: getTimeStamp()}
}

func CreateWebDumpTask(dump, info string) *WebDump {
	return &WebDump{Type: PROCESS_WEB_DUMP,
		Path: dump,
		Info: info,
		Time: getTimeStamp()}
}

func getTimeStamp() string {
	t := time.Now()
	return t.Format(time.RFC3339)
}
