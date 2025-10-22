package discover

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/obi/pkg/components/exec"
	"go.opentelemetry.io/obi/pkg/components/svc"
	"go.opentelemetry.io/obi/pkg/internal/goexec"
)

// TestConcurrentELFParsing verifies that the semaphore correctly limits
// concurrent ELF parsing operations
func TestConcurrentELFParsing(t *testing.T) {
	// Save original semaphore and restore after test
	originalSem := elfParseSem
	defer func() { elfParseSem = originalSem }()
	
	// Create a semaphore with limit of 2
	elfParseSem = make(chan struct{}, 2)
	
	// Create test typer with empty cache
	cache, _ := lru.New[uint64, InstrumentedExecutable](10)
	typer := &typer{
		instrumentableCache: cache,
		log:                 slog.Default(),
	}
	
	// Track concurrent parsers
	var activeParsers int32
	var maxActiveParsers int32
	
	// Override inspectOffsets to simulate slow ELF parsing
	originalInspectOffsets := typer.inspectOffsets
	typer.inspectOffsets = func(execElf *exec.FileInfo) (*goexec.Offsets, bool, error) {
		// Increment active parsers counter
		current := atomic.AddInt32(&activeParsers, 1)
		
		// Track maximum concurrent parsers
		for {
			max := atomic.LoadInt32(&maxActiveParsers)
			if current <= max || atomic.CompareAndSwapInt32(&maxActiveParsers, max, current) {
				break
			}
		}
		
		// Simulate parsing work
		time.Sleep(50 * time.Millisecond)
		
		// Decrement active parsers
		atomic.AddInt32(&activeParsers, -1)
		
		return &goexec.Offsets{}, true, nil
	}
	defer func() { typer.inspectOffsets = originalInspectOffsets }()
	
	// Launch 5 concurrent parsing attempts
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(pid int32) {
			defer wg.Done()
			
			execElf := &exec.FileInfo{
				Pid:        pid,
				CmdExePath: "/test/binary",
				Ino:        uint64(pid), // Different inode for each to avoid cache
			}
			
			_ = typer.asInstrumentable(execElf)
		}(int32(i))
	}
	
	wg.Wait()
	
	// Verify that no more than 2 parsers ran concurrently
	assert.LessOrEqual(t, maxActiveParsers, int32(2),
		"Expected at most 2 concurrent parsers, but had %d", maxActiveParsers)
}

// TestCacheHitAvoidsParsing verifies that cache hits skip ELF parsing entirely
func TestCacheHitAvoidsParsing(t *testing.T) {
	cache, _ := lru.New[uint64, InstrumentedExecutable](10)
	
	// Pre-populate cache
	cached := InstrumentedExecutable{
		Type:    svc.InstrumentableGolang,
		Offsets: &goexec.Offsets{},
	}
	cache.Add(uint64(123), cached)
	
	typer := &typer{
		instrumentableCache: cache,
		log:                 slog.Default(),
	}
	
	parseCount := 0
	originalInspectOffsets := typer.inspectOffsets
	typer.inspectOffsets = func(execElf *exec.FileInfo) (*goexec.Offsets, bool, error) {
		parseCount++
		return nil, false, nil
	}
	defer func() { typer.inspectOffsets = originalInspectOffsets }()
	
	// First request - should hit cache
	execElf1 := &exec.FileInfo{
		Pid: 1,
		Ino: 123,
	}
	result1 := typer.asInstrumentable(execElf1)
	
	// Second request with same inode - should also hit cache
	execElf2 := &exec.FileInfo{
		Pid: 2,
		Ino: 123,
	}
	result2 := typer.asInstrumentable(execElf2)
	
	// Verify no parsing occurred
	assert.Equal(t, 0, parseCount, "Expected no parsing for cached entries")
	assert.Equal(t, svc.InstrumentableGolang, result1.Type)
	assert.Equal(t, svc.InstrumentableGolang, result2.Type)
}