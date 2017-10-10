package service

import (
	"os"
	"os/exec"
	"regexp"
	"io/ioutil"
	"encoding/json"
	"yabs/common/task"
	"yabs/processor/cfg"
	"yabs/common/format"
	"yabs/common/format/minidump"
	"yabs/common/data/base"
	"path/filepath"
	"strings"
	log "github.com/sirupsen/logrus"
	"yabs/processor/pipeline"
)

type WebdumpProcessor struct {
	config        cfg.Config
	repository    *base.Repository
	ffAndChromeRx *regexp.Regexp
	pline         []pipeline.Stage
}

func (w *WebdumpProcessor) initWebdumpProcessor(c cfg.Config, rep *base.Repository) {
	w.config = c
	w.repository = rep

	var err error = nil
	w.ffAndChromeRx, err = regexp.Compile("^.* ((?:firefox|chrome)/[\\d\\.]+).*$")
	if err != nil {
		log.WithError(err).Panic("Can't compile firefox/chrome version regex")
	}

	w.pline = []pipeline.Stage{
		pipeline.NewRx(w.config.WebBlackListSignaturs()),
	}
}

func (w *WebdumpProcessor) handleWebDump(t *task.WebDump) *ReportWithId {
	defer w.removeFiles(t)

	info, browser, err := w.extractInfoAndBrowser(t)
	if err != nil {
		log.WithError(err).Error("Can't read webinfo")
		return nil
	}

	if info.Version == DEVELOPER_VERSION {
		log.WithField("info", info).Debug("Skipped developers' dump")
		return nil
	}

	sym, err := w.repository.GetSymbolForPlatform("web", info.Version)
	if err != nil || sym == nil {
		infoData, _ := ioutil.ReadFile(t.Info)
		log.WithFields(log.Fields{
			"version": info.Version,
			"data":    string(infoData),
		}).Debug("Can't search symbol for web")
		return nil
	}

	symbolPaths := w.getSymbolFiles(sym.DirPath)
	paths := []string{t.Path}
	paths = append(paths, symbolPaths...)
	cmd := exec.Command("webstackwalker", paths...)

	out, err := cmd.Output()
	if err != nil {
		log.WithField("symbols path", symbolPaths).
			Error(err)
		return nil
	}

	var trace []string
	json.Unmarshal(out, &trace)

	var dumpContext minidump.Context

	dumpContext.SystemInfo.CpuCount = 1
	dumpContext.SystemInfo.OS = info.Platform
	dumpContext.SystemInfo.OS_Version = browser
	dumpContext.CrashingThread.ThreadIndex = 0
	dumpContext.Modules = []minidump.ModuleInfo{}
	dumpContext.CrashInfo.Address = "unknow"
	dumpContext.CrashInfo.Type = "unknow"
	dumpContext.CrashInfo.Thread = 1

	dumpContext.Threads = append(dumpContext.Threads, minidump.ThreadInfo{FrameCount: uint(len(trace))})
	for i, v := range trace {
		frame := minidump.TrheadFrame{
			Frame:    uint(i),
			Function: v,
			Line:     0,
		}
		dumpContext.CrashingThread.Frames = append(dumpContext.CrashingThread.Frames,
			frame)
		dumpContext.Threads[0].Frames = append(dumpContext.Threads[0].Frames,
			frame)
	}

	dump, err := ioutil.ReadFile(t.Path)
	if err != nil {
		log.WithError(err).Warning("Can't read web dump")
		return nil
	}

	rawDump :=  string(dump)

	return w.processingReport(&dumpContext, info, rawDump, t)
}

func (w *WebdumpProcessor) processingReport(crash *minidump.Context, info *format.Info, raw_crash string, t *task.WebDump) *ReportWithId {
	var source string = ""

	report := minidump.Report{
		Context:      *crash,
		Platform:     "web",
		BuildVersion: info.Version,
		Source:       source,
		CrashType:    crash.CrashInfo.Type,
		Address:      crash.CrashInfo.Address,
		DateAdded:    t.Time,
		Gpu:          info.Gpu,
		RawCrash:     raw_crash,
		UserId: 	  info.GetUserId(),
	}

	for _, stage := range w.pline {
		if stage.Process(&report, info) {
			break
		}
	}

	if report.Signature == "" && (strings.Count(raw_crash, "\n") <= 2) {
		report.Signature = raw_crash
	}

	id, _ := w.repository.AddReport(&report)
	return &ReportWithId{
		Report: report,
		Id:     id,
	}
}

func (w *WebdumpProcessor) removeFiles(t *task.WebDump) {
	err := os.Remove(t.Path)
	if err != nil {
		log.WithError(err).Warning("Can't remove file")
	}

	err = os.Remove(t.Info)
	if err != nil {
		log.WithError(err).Warning("Can't remove file")
	}
}

func (w *WebdumpProcessor) extractInfoAndBrowser(t *task.WebDump) (*format.Info, string, error) {
	info, err := format.InfoFromFile(t.Info)
	if err != nil {
		return nil, "", err
	}

	browser := info.Browser
	if w.ffAndChromeRx.MatchString(browser) {
		match := w.ffAndChromeRx.FindStringSubmatch(browser)
		browser = match[1]
	}
	return info, browser, nil
}

func (w *WebdumpProcessor) getSymbolFiles(dir string) []string {
	paths := []string{}

	filepath.Walk(dir,
		func(path string, _ os.FileInfo, _ error) error {
			if strings.Contains(path, WebFileSymbol) {
				paths = append(paths, path)
			}
			return nil
		})

	return paths
}