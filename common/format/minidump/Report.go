package minidump

import (
	"yabs/common/format"
)

type Report struct {
	Context
	UserId       uint64 `json:"user_id"`
	BuildVersion string `json:"build"`
	Platform     string `json:"platform"`
	Signature    string `json:"signature"`
	Source       string `json:"source"`
	CrashType    string `json:"crash_type"`
	Address      string `json:"address"`
	DateAdded    string `json:"date_added"`
	Gpu          format.GPUInfo `json:"gpu"`
	Ram          string `json:"ram,omitempty"`
	RawCrash     string `json:"raw_dump,omitempty"`
	Log          string `json:"raw_log,omitempty"`
}
