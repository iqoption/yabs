// Package pipeline contains objects for processing by a conveyor
package pipeline

import (
	"fmt"
	"strings"
	"yabs/common/format"
	"yabs/common/format/minidump"
	"regexp"
)

// Pipeline stage
type Stage interface {
	//Process the report
	//If return true then pipeline stop
	Process(report *minidump.Report, info *format.Info) bool
}

type SignatureAndSource struct {
	Stage
}

type MinidumpStackUnfolding struct {
	Stage
}

func (m *SignatureAndSource) Process(report *minidump.Report, info *format.Info) bool {
	if len(report.CrashingThread.Frames) > 0 {
		frame := &report.CrashingThread.Frames[0]
		signature := frame.Function
		source := fmt.Sprintf("%s:%d", frame.File,
			frame.Line)
		report.Signature = signature
		report.Source = source
	}

	return false
}

func (m *MinidumpStackUnfolding) Process(report *minidump.Report, info *format.Info) bool {
	frames := report.CrashingThread.Frames
	if len(report.CrashingThread.Frames) == 0 {
		// go to next stage
		return false
	}

	iqoption := regexp.MustCompile("iq\\s*option")
	for _, frame := range frames {
		module := strings.ToLower(frame.Module)
		if iqoption.MatchString(module) {
			report.Signature = frame.Function
			report.Source = fmt.Sprintf("%s:%d", frame.File,
				frame.Line)
			return true
		}
	}

	return true
}
