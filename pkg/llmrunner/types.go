package llmrunner

type GpuInfo struct {
	Name               string
	TemperatureC       int32
	MemoryTotalMB      uint64
	MemoryUsedMB       uint64
	UtilizationPercent uint32
}

type ServerInfo struct {
	Hostname      string
	OS            string
	Arch          string
	CPUCores      int32
	MemoryTotalMB uint64
	Models        []string
}

type LoadedModelStatus struct {
	Loaded       bool
	DisplayName  string
	GGUFBasename string
}

type RunnerInfo struct {
	ID            int64
	Address       string
	Host          string
	Port          int32
	Name          string
	Enabled       bool
	SelectedModel string

	Connected   bool
	Gpus        []GpuInfo
	Server      *ServerInfo
	LoadedModel *LoadedModelStatus
}

type RunnerProbeResult struct {
	Connected   bool
	Gpus        []GpuInfo
	Server      *ServerInfo
	LoadedModel *LoadedModelStatus
}
