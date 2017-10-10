package pipeline

import (
	"regexp"
	"yabs/common/format"
	"yabs/common/format/minidump"
	log "github.com/sirupsen/logrus"
)

// Regular Expression Descent
type Rx struct {
	Stage
	Regexps []*regexp.Regexp
}

func (r *Rx) Process(report *minidump.Report, info *format.Info) bool {
	if len(r.Regexps) == 0 {
		// to next stage
		return false
	}

	frames := report.CrashingThread.Frames
	if len(report.CrashingThread.Frames) == 0 {
		// go to next stage
		return false
	}

	for _, frame := range frames {
		isMatch := false
		for _, rx := range r.Regexps {
			isMatch = rx.MatchString(frame.Function) || isMatch
			if isMatch {
				break
			}
		}
		if !isMatch {
			report.Signature = frame.Function
			return true
		}
	}

	if len(frames) > 0 {
		report.Signature = frames[0].Function
	}

	return true
}

func NewRx(regs []string) *Rx {
	var rxSlice []*regexp.Regexp
	for _, reg := range regs {
		rx, err := regexp.Compile(reg)
		log.WithField("regexp", reg).
			Debug("Rx stage: compile regexp")
		if err == nil {
			rxSlice = append(rxSlice, rx)
		} else {
			log.WithError(err).
				Error("Can't compile regular expression")
		}
	}

	return &Rx{
		Regexps: rxSlice,
	}
}
