package log

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestNewPipeReader(t *testing.T) {
	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader()

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	log.Error("scooby")
	r.Close()
	wg.Wait()

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}

func TestNewPipeReader_parallel(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("parallel#%d", i), func(t *testing.T) {
			t.Parallel()
			var (
				aPipe   = NewPipeReader()
				bPipe   = NewPipeReader()
				aReader = bufio.NewReader(aPipe)
				bReader = bufio.NewReader(bPipe)
				aLogger = getLogger("A")
				bLogger = getLogger("B")
				readWG  sync.WaitGroup
				writeWG sync.WaitGroup
			)

			writeWG.Add(20)
			for i := 0; i < 10; i++ {
				go func() {
					aLogger.Error("scooby")
					writeWG.Done()
				}()
				go func() {
					bLogger.Error("scooby")
					writeWG.Done()
				}()
			}

			readWG.Add(2)
			go func() {
				i := 0
				for {
					line, err := aReader.ReadString('\n')
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if !strings.Contains(line, "scooby") {
						t.Errorf("got %q, wanted it to contain log output", line)
					}
					i++
				}
				if i != 20 {
					t.Errorf("got %d lines, expected 20", i)
				}
				readWG.Done()
			}()
			go func() {
				i := 0
				for {
					line, err := bReader.ReadString('\n')
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if !strings.Contains(line, "scooby") {
						t.Errorf("got %q, wanted it to contain log output", line)
					}
					i++
				}
				if i != 20 {
					t.Errorf("got %d lines, expected 20", i)
				}
				readWG.Done()
			}()

			writeWG.Wait()
			aPipe.Close()
			bPipe.Close()
			readWG.Wait()
		})
	}
}

func TestNewPipeReaderFormat(t *testing.T) {
	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader(PipeFormat(PlaintextOutput))

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	log.Error("scooby")
	r.Close()
	wg.Wait()

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}

func TestNewPipeReaderLevel(t *testing.T) {
	SetupLogging(Config{
		Level:  LevelDebug,
		Format: PlaintextOutput,
	})

	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader(PipeLevel(LevelError))

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	log.Debug("scooby")
	log.Info("velma")
	log.Error("shaggy")
	r.Close()
	wg.Wait()

	lineEnding := zap.NewProductionEncoderConfig().LineEnding

	// Should only contain one log line
	if strings.Count(buf.String(), lineEnding) > 1 {
		t.Errorf("got %d log lines, wanted 1", strings.Count(buf.String(), lineEnding))
	}

	if !strings.Contains(buf.String(), "shaggy") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}
