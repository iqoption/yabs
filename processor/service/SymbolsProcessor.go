package service

import (
	"yabs/common/task"
	"os"
	"bufio"
	"regexp"
	"path/filepath"
	"fmt"
	"strings"
	"yabs/common/data/base"
	"io/ioutil"
	"encoding/json"
	"time"
	"crypto/md5"
	"crypto/sha1"
	"io"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type SymbolsProcessor struct {
	symbols    string
	modulRx    *regexp.Regexp
	nameRx     *regexp.Regexp
	repository *base.Repository
}

type SymbolDescritpion struct {
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

const WebFileSymbol = "file.symbol"

func (s *SymbolsProcessor) initSymbolProcessor(path string, rep *base.Repository) {
	s.symbols = path
	s.repository = rep
	var err error = nil
	osCapGroup := "(Linux|mac|Macintosh OSX|windows|Microsoft Windows)"
	archNoCapGroup := "(?:x86|Intel IA-32|x86_64|AMD64/Intel 64|ppc|32-bit PowerPC|ppc64|64-bit PowerPC|unknown)"
	// MODULE operatingsystem architecture id name
	s.modulRx, err = regexp.Compile(fmt.Sprintf("^MODULE %s %s (\\S+) (.*)",
		osCapGroup,
		archNoCapGroup))
	if err != nil {
		log.Panic("Can't compile symbol module regex: %s", err.Error())
	}

	s.nameRx, err = regexp.Compile("([\\w\\s\\d]+)(\\.\\w+)*")
	if err != nil {
		log.Panic("Can't compile name regex: %s", err.Error())
	}
}

func (s *SymbolsProcessor) handleSymbol(t *task.Symbol) {
	log.WithFields(log.Fields{
		"info file":   t.Info,
		"symbol file": t.Path,
	}).Debug("Handle symbols")

	desc := s.readDescription(t)
	if desc == nil {
		s.removeFiles(t)
		return
	}
	log.WithFields(log.Fields{
		"platform": desc.Platform,
		"version":  desc.Version,
	}).Debug("Append")

	switch desc.Platform {
	case "web":
		s.handleWebSymbol(desc, t)
	default:
		s.handleBreakpadSymbol(desc, t)
	}
}

func (s *SymbolsProcessor) handleWebSymbol(d *SymbolDescritpion, t *task.Symbol) {
	id := uuid.NewV4().String()

	if id == "" {
		log.WithField("path", t.Path).
			Error("Can't calculate id for web symbols")
		s.removeFiles(t)
		return
	}

	dirPath := filepath.Join(s.symbols,
		"WebSymbols",
		id)

	err := os.MkdirAll(dirPath, 0777)
	if err != nil {
		log.WithFields(log.Fields{
			"path":  dirPath,
			"error": err,
		}).Error("Can't create target symbols dir")
		s.removeFiles(t)
		return
	}

	if len(t.Path) != 0 {
		targetFileName := filepath.Join(dirPath, WebFileSymbol)
		err = os.Rename(t.Path, targetFileName)
		if err != nil {
			log.WithError(err).
				Error("Can't move symbol file into target dir")
			return
		}
	}

	for i, path := range t.Paths {
		targetFileName := filepath.Join(dirPath, WebFileSymbol)
		if i > 0 {
			targetFileName = filepath.Join(dirPath, fmt.Sprintf("%d.%s", i, WebFileSymbol))
		}

		err = os.Rename(path, targetFileName)
		if err != nil {
			log.WithError(err).
				Error("Can't move symbol file into target dir")
			return
		}
	}

	targetInfoName := filepath.Join(dirPath, "info.json")
	err = os.Rename(t.Info, targetInfoName)
	if err != nil {
		log.WithError(err).
			Error("Can't move info file into target dir")
		return
	}

	s.PutSymbolInStorage(dirPath, d.Version, id, "web")
}

func (s *SymbolsProcessor) handleBreakpadSymbol(d *SymbolDescritpion, t *task.Symbol) {

	id, fullName, platform, err := s.extractInfoFromSymbols(t)
	if err != nil {
		s.removeFiles(t)
		log.WithField("task", t).
			Warning("Can't extract info from symbol file")
		return
	}

	exist, err := s.repository.IsExist(&base.Symbol{
		DebugId: id,
	})

	if err != nil {
		panic(fmt.Sprintf("Can't check symbol id in repository %s", err.Error()))
		s.removeFiles(t)
		return
	}

	if exist {
		log.WithField("debug id", id).
			Warning("Symbol file with is exist")
		s.removeFiles(t)
		return
	}

	if err != nil {
		s.removeFiles(t)
		log.WithField("task", t).
			Warning("Can't extract version")
		return
	}

	dirPath := filepath.Join(s.symbols,
		fullName,
		id)

	err = os.MkdirAll(dirPath, 0777)
	if err != nil {
		s.removeFiles(t)
		log.WithError(err).
			Error("Can't create target symbols dir")
		return
	}

	names := s.nameRx.FindStringSubmatch(fullName)
	targetFileName := filepath.Join(dirPath, fmt.Sprintf("%s.sym", names[1]))
	err = os.Rename(t.Path, targetFileName)
	if err != nil {
		s.removeFiles(t)
		log.WithError(err).
			Error("Can't move symbol file into target dir")
		return
	}

	targetInfoName := filepath.Join(dirPath, "info.json")
	err = os.Rename(t.Info, targetInfoName)
	if err != nil {
		s.removeFiles(t)
		log.WithError(err).
			Error("Can't move info file into target dir")
		return
	}

	s.PutSymbolInStorage(dirPath, d.Version, id, platform)
}

func (s *SymbolsProcessor) PutSymbolInStorage(dirPath, version, id, platform string) {
	log.WithFields(log.Fields{
		"platform": platform,
		"version":  version,
		"id":       id,
	}).Debug("Put symbols in elastic")

	now := time.Now()
	err := s.repository.AddSymbol(&base.Symbol{
		dirPath,
		version,
		id,
		platform,
		now.Format(time.RFC3339),
	})

	if err != nil {
		panic(fmt.Sprintf("Can't append new symbol in database: %s",
			err.Error()))
	}
}

func (s *SymbolsProcessor) extractInfoFromSymbols(t *task.Symbol) (id, fullName, platform string, err error) {
	file, err := os.Open(t.Path)
	if err != nil {
		log.WithError(err).
			Error("Can't open symbol file on read")
		s.removeFiles(t)
		return "", "", "", err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')

	if err != nil {
		file.Close()
		s.removeFiles(t)
		return "", "", "", fmt.Errorf("Can't read info from symbol file: %s", err.Error())

	}

	if !s.modulRx.MatchString(line) {
		file.Close()
		s.removeFiles(t)
		return "", "", "", fmt.Errorf("It isn't file symbols %s", t.Path)
	}

	match := s.modulRx.FindStringSubmatch(line)

	return strings.TrimSpace(match[2]), strings.TrimSpace(match[3]), strings.TrimSpace(match[1]), nil
}

func (s *SymbolsProcessor) removeFiles(t *task.Symbol) {
	err := os.Remove(t.Path)
	if err != nil {
		log.WithField("path", t.Path).
			Warning("Can't remove file")
	}

	err = os.Remove(t.Info)
	if err != nil {
		log.WithField("path", t.Info).
			Warning("Can't remove file")
	}
}

func (s *SymbolsProcessor) readDescription(t *task.Symbol) (*SymbolDescritpion) {
	data, err := ioutil.ReadFile(t.Info)
	if err != nil {
		log.WithError(err).Warning("Can't read symbol description file")
		return nil
	}
	var d SymbolDescritpion
	err = json.Unmarshal(data, &d)
	if err != nil {
		log.WithError(err).
			Error("Can't parse symbol description file")
		return nil
	}

	return &d
}

func (s *SymbolsProcessor) getFileMd5Hash(filepath string) string {
	f, err := os.Open(filepath)
	if err != nil {
		log.WithError(err).
			Error("Can't open file for calculate md5")
		return ""
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.WithField("path", filepath).
			Error("Can't calculate md5 for file")
		return ""
	}

	return strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil)))
}

func (s *SymbolsProcessor) getFileSha1Hash(filepath string) string {
	f, err := os.Open(filepath)
	if err != nil {
		log.WithError(err).
			Error("Can't open file for calculate md5")
		return ""
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.WithField("path", filepath).
			Error("Can't calculate sha1 for file")
		return ""
	}

	return strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil)))
}

func (m *SymbolsProcessor) webPrefix() string {
	dt := time.Now()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d", dt.Year(),
		dt.Month(),
		dt.Day(),
		dt.Hour(),
		dt.Minute(),
		dt.Second())
}
