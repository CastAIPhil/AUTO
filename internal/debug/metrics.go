package debug

import (
	"runtime"
	"runtime/metrics"
	"time"
)

// RuntimeMetrics contains a snapshot of runtime performance metrics
type RuntimeMetrics struct {
	Timestamp time.Time `json:"timestamp"`

	HeapAlloc     uint64  `json:"heap_alloc_bytes"`
	HeapInUse     uint64  `json:"heap_in_use_bytes"`
	HeapObjects   uint64  `json:"heap_objects"`
	StackInUse    uint64  `json:"stack_in_use_bytes"`
	TotalAlloc    uint64  `json:"total_alloc_bytes"`
	Mallocs       uint64  `json:"mallocs"`
	Frees         uint64  `json:"frees"`
	NumGC         uint32  `json:"num_gc"`
	NumGoroutines int     `json:"num_goroutines"`
	GCPauseTotal  uint64  `json:"gc_pause_total_ns"`
	LastGCPause   uint64  `json:"last_gc_pause_ns"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
}

// CollectRuntimeMetrics gathers current runtime statistics
func CollectRuntimeMetrics() *RuntimeMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &RuntimeMetrics{
		Timestamp:     time.Now(),
		HeapAlloc:     m.HeapAlloc,
		HeapInUse:     m.HeapInuse,
		HeapObjects:   m.HeapObjects,
		StackInUse:    m.StackInuse,
		TotalAlloc:    m.TotalAlloc,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		NumGC:         m.NumGC,
		NumGoroutines: runtime.NumGoroutine(),
		GCPauseTotal:  m.PauseTotalNs,
		LastGCPause:   m.PauseNs[(m.NumGC+255)%256],
		GCCPUFraction: m.GCCPUFraction,
	}
}

// MetricsSample holds a single metric from runtime/metrics
type MetricsSample struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Kind  string  `json:"kind"`
}

// CollectDetailedMetrics uses the newer runtime/metrics API for lower overhead
func CollectDetailedMetrics() []MetricsSample {
	descs := metrics.All()
	samples := make([]metrics.Sample, len(descs))
	for i := range descs {
		samples[i].Name = descs[i].Name
	}
	metrics.Read(samples)

	result := make([]MetricsSample, 0, len(samples))
	for _, s := range samples {
		ms := MetricsSample{Name: s.Name}
		switch s.Value.Kind() {
		case metrics.KindUint64:
			ms.Value = float64(s.Value.Uint64())
			ms.Kind = "uint64"
		case metrics.KindFloat64:
			ms.Value = s.Value.Float64()
			ms.Kind = "float64"
		default:
			continue
		}
		result = append(result, ms)
	}
	return result
}

// GoroutineStats returns goroutine count breakdown
type GoroutineStats struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Idle    int `json:"idle"`
}

// CollectGoroutineStats returns current goroutine statistics
func CollectGoroutineStats() *GoroutineStats {
	return &GoroutineStats{
		Total: runtime.NumGoroutine(),
	}
}
