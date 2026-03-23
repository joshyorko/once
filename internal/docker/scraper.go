package docker

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
)

// Sample represents CPU and memory stats for a single scrape interval
type Sample struct {
	Timestamp   time.Time
	CPUPercent  float64
	MemoryBytes uint64
}

// ScraperSettings configures the docker stats scraper
type ScraperSettings struct {
	BufferSize int
}

const defaultStatBufferSize = 200

func (s ScraperSettings) withDefaults() ScraperSettings {
	if s.BufferSize == 0 {
		s.BufferSize = defaultStatBufferSize
	}
	return s
}

// statsClient defines the Docker API operations needed by the scraper
type statsClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
}

// Scraper collects Docker container stats using persistent streaming connections.
// Background goroutines maintain streams for each container and update latest stats.
// Scrape() snapshots the current values without blocking on Docker API calls.
type Scraper struct {
	settings        ScraperSettings
	client          statsClient
	names           []serviceNameMatcher
	appPrefix       string
	accessoryPrefix string

	mu        sync.RWMutex
	apps      map[string]*appData
	streams   map[string]*streamInfo
	lastError error
}

type serviceNameMatcher struct {
	prefix string
	parse  func(string) string
}

type streamInfo struct {
	containerID string
	cancel      context.CancelFunc
}

type appData struct {
	samples *RingBuffer[Sample]
	latest  *liveStats
}

type liveStats struct {
	cpuPercent  float64
	memoryBytes uint64
}

func NewScraper(ns *Namespace, settings ScraperSettings) *Scraper {
	settings = settings.withDefaults()
	return &Scraper{
		settings:        settings,
		client:          ns.client,
		appPrefix:       ns.name + "-app-",
		accessoryPrefix: ns.name + "-accessory-",
		names: []serviceNameMatcher{
			{prefix: ns.name + "-app-", parse: ns.containerAppName},
			{prefix: ns.name + "-accessory-", parse: ns.containerAccessoryName},
		},
		apps:    make(map[string]*appData),
		streams: make(map[string]*streamInfo),
	}
}

// Fetch returns the last n samples for a service, ordered from newest to oldest.
// If fewer than n samples exist, only the available samples are returned.
func (s *Scraper) Fetch(serviceName string, n int) []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.apps[serviceName]
	if !ok {
		return nil
	}

	return data.samples.FetchNewestFirst(n)
}

func (s *Scraper) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

// Scrape snapshots the latest stats from streaming connections and records samples.
// It also ensures streams are running for all current containers.
func (s *Scraper) Scrape(ctx context.Context) {
	containers, err := s.findAppContainers(ctx)
	if err != nil {
		s.setError(err)
		return
	}

	s.updateStreams(ctx, containers)

	now := time.Now()
	s.mu.Lock()
	for serviceName, data := range s.apps {
		var sample Sample
		if _, running := containers[serviceName]; running && data.latest != nil {
			sample = Sample{
				Timestamp:   now,
				CPUPercent:  data.latest.cpuPercent,
				MemoryBytes: data.latest.memoryBytes,
			}
		} else {
			sample = Sample{Timestamp: now}
			data.latest = nil
		}

		data.samples.Add(sample)
	}
	s.mu.Unlock()

	s.setError(nil)
}

func (s *Scraper) findAppContainers(ctx context.Context) (map[string]string, error) {
	containers, err := s.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")
			serviceName := s.serviceName(name)
			if serviceName == "" && strings.HasPrefix(name, s.accessoryPrefix) {
				serviceName = NameFromImageRef(c.Image)
			}
			if serviceName == "" {
				continue
			}
			result[serviceName] = c.ID
		}
	}

	return result, nil
}

// Private

func (s *Scraper) serviceName(containerName string) string {
	for _, matcher := range s.names {
		if strings.HasPrefix(containerName, matcher.prefix) {
			return matcher.parse(containerName)
		}
	}
	return ""
}

func (s *Scraper) updateStreams(ctx context.Context, containers map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Start or restart streams for containers
	for serviceName, containerID := range containers {
		if stream, exists := s.streams[serviceName]; exists {
			if stream.containerID == containerID {
				continue
			}
			// Container ID changed (redeployed), restart stream
			stream.cancel()
		}

		if s.apps[serviceName] == nil {
			s.apps[serviceName] = &appData{
				samples: NewRingBuffer[Sample](s.settings.BufferSize),
			}
		}

		streamCtx, cancel := context.WithCancel(ctx)
		s.streams[serviceName] = &streamInfo{containerID: containerID, cancel: cancel}
		go s.runStream(streamCtx, serviceName, containerID)
	}

	// Stop streams for removed containers
	for serviceName, stream := range s.streams {
		if _, exists := containers[serviceName]; !exists {
			stream.cancel()
			delete(s.streams, serviceName)
		}
	}
}

func (s *Scraper) runStream(ctx context.Context, serviceName, containerID string) {
	for {
		s.streamStats(ctx, serviceName, containerID)

		select {
		case <-ctx.Done():
			return
		case <-time.After(streamRetryDelay):
			// Retry after brief delay if stream disconnected
		}
	}
}

func (s *Scraper) streamStats(ctx context.Context, serviceName, containerID string) {
	resp, err := s.client.ContainerStats(ctx, containerID, true)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	for {
		var stats container.StatsResponse
		if err := decoder.Decode(&stats); err != nil {
			return
		}

		s.mu.Lock()
		if data := s.apps[serviceName]; data != nil {
			data.latest = &liveStats{
				cpuPercent:  calculateCPUPercent(&stats),
				memoryBytes: stats.MemoryStats.Usage,
			}
		}
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (s *Scraper) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
}

// Helpers

func calculateCPUPercent(stats *container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
	}
	return 0.0
}
