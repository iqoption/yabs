package minidump

type CrashInfo struct {
	Address string `json:"address"`
	Thread  uint   `json:"crashing_thread"`
	Type    string `json:"type"`
}

type ModuleInfo struct {
	Address       string `json:"base_addr"`
	Id            string `json:"code_id"`
	DebugFile     string `json:"debug_file"`
	DebugId       string `json:"debug_id"`
	EndAddr       string `json:"end_addr"`
	File          string `json:"filename"`
	Version       string `json:"version"`
	LoadedSymbols bool   `json:"loaded_symbols"`
}

type Sensitive struct {
	Exploitability string `json:"exploitability"`
}

type SysInfo struct {
	CpuArch    string `json:"cpu_arch"`
	CpuCount   uint   `json:"cpu_count"`
	CpuInfo    string `json:"cpu_info"`
	OS         string `json:"os"`
	OS_Version string `json:"os_ver"`
}

type TrheadFrame struct {
	File           string            `json:"file"`
	Frame          uint              `json:"frame"`
	Function       string            `json:"function"`
	FunctionOffset string            `json:"function_offset"`
	Line           uint              `json:"line"`
	Module         string            `json:"module"`
	ModuleOffset   string            `json:"module_offset"`
	Registers      map[string]string `json:"registers,omitempty"`
	Trust          string            `json:"trust"`
}

type ThreadInfo struct {
	FrameCount uint          `json:"frame_count"`
	Frames     []TrheadFrame `json:"frames"`
}

type CrashingThread struct {
	Frames      []TrheadFrame `json:"frames"`
	ThreadIndex uint          `json:"threads_index"`
	TotalFrames uint          `json:"total_frames"`
}

type Context struct {
	CrashInfo      CrashInfo      `json:"crash_info"`
	CrashingThread CrashingThread `json:"crashing_thread"`
	MainModule     uint           `json:"main_module"`
	Modules        []ModuleInfo   `json:"modules"`
	Status         string         `json:"status"`
	SystemInfo     SysInfo        `json:"system_info"`
	ThreadCount    uint           `json:"thread_count"`
	Threads        []ThreadInfo   `json:"threads"`
}
