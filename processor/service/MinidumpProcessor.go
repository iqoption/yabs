package service

import (
	"os"
	"os/exec"
	"encoding/json"
	"yabs/common/task"
	"yabs/processor/cfg"
	"yabs/processor/pipeline"
	"yabs/common/format"
	"yabs/common/format/minidump"
	"yabs/common/data/base"
	"regexp"
	"strings"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
)

type MinidumpProcessor struct {
	config     cfg.Config
	repository *base.Repository
	linRx      *regexp.Regexp
	winRx      *regexp.Regexp
	macRx      *regexp.Regexp
	pline      []pipeline.Stage
}

func (s *MinidumpProcessor) initMinidumpProcessor(c cfg.Config, rep *base.Repository) {
	s.config = c
	s.repository = rep

	s.linRx = regexp.MustCompile("linux")
	s.winRx = regexp.MustCompile("windows")
	s.macRx = regexp.MustCompile("mac")
	s.pline = []pipeline.Stage{
		&pipeline.SignatureAndSource{},
		&pipeline.MinidumpStackUnfolding{},
	}
}

func (s *MinidumpProcessor) handleMiniDump(t *task.Dump) *ReportWithId {
	defer s.removeFiles(t)

	info, err := format.InfoFromFile(t.Info)
	if err != nil {
		info = &format.Info{}
	}

	if info.Version == DEVELOPER_VERSION {
		log.WithField("info", info).Debug("Skipped developers' dump")
		return nil
	}

	cmd := exec.Command("stackwalker",
		t.Path,
		s.config.SymbolsPath())

	out, err := cmd.Output()
	if err != nil {
		log.WithError(err).Error("Can't read stackwalker output")
		return nil
	}

	var dumpContext minidump.Context
	err = json.Unmarshal(out, &dumpContext)
	if err != nil {
		log.WithError(err).Error("Can't parse Json minidump context")
		return nil
	}

	if len(dumpContext.Modules) != 0 {
		did := s.findModuleID(dumpContext.Modules)
		sym, err := s.repository.GetSymbol(did)
		if err != nil {
			log.WithFields(log.Fields{
				"debug_id": did,
				"error":    err,
				"info":     info,
			}).Warning("Can't get version for debug id")
			log.Warning(string(out))
		} else {
			return s.processingReport(&dumpContext,
				sym.Version,
				info,
				s.readLog(t), t)
		}
	} else {
		log.WithField("context", dumpContext).
			Debug("Crash report don't have modules")
	}
	return nil
}

func (s *MinidumpProcessor) processingReport(crash *minidump.Context, version string, info *format.Info, log string, t *task.Dump) *ReportWithId {

	report := minidump.Report{
		Context:      *crash,
		UserId:       info.GetUserId(),
		BuildVersion: version,
		Platform:     s.getPlatform(crash.SystemInfo.OS),
		CrashType:    crash.CrashInfo.Type,
		Address:      crash.CrashInfo.Address,
		DateAdded:    t.Time,
		Gpu:          info.Gpu,
		Ram:          info.Ram,
		Log:          log,
	}

	for _, stage := range s.pline {
		if stage.Process(&report, info) {
			break
		}
	}

	report.SystemInfo.CpuInfo = info.Cpu
	id, _ := s.repository.AddReport(&report)
	return &ReportWithId{
		Report: report,
		Id:     id,
	}
}

func (s *MinidumpProcessor) removeFiles(t *task.Dump) {
	err := os.Remove(t.Path)
	if err != nil {
		log.WithFields(log.Fields{
			"path":  t.Path,
			"error": err,
		}).Warning("Can't remove file ")
	}

	err = os.Remove(t.Info)
	if err != nil {
		log.WithFields(log.Fields{
			"path":  t.Info,
			"error": err,
		}).Warning("Can't remove file")
	}

	err = os.Remove(t.Log)
	if err != nil {
		log.WithFields(log.Fields{
			"path":  t.Log,
			"error": err,
		}).Warning("Can't remove file")
	}
}

func (s *MinidumpProcessor) getPlatform(os string) string {
	os = strings.ToLower(os)
	if s.linRx.FindString(os) != "" {
		return "lin"
	} else if s.macRx.FindString(os) != "" {
		return "mac"
	} else if s.winRx.FindString(os) != "" {
		return "win"
	}
	return os
}

func (s *MinidumpProcessor) readLog(t *task.Dump) string {

	if len(t.Log) == 0 {
		return "No log"
	}

	var logData string

	data, err := ioutil.ReadFile(t.Log)
	if err != nil {
		log.WithError(err).Warning("Can't read log file")
	}

	logData = string(data)

	return logData
}

func (s *MinidumpProcessor) findModuleID(module []minidump.ModuleInfo) string {
	for _, m := range module {
		if m.LoadedSymbols {
			return m.DebugId
		}
	}
	return ""
}