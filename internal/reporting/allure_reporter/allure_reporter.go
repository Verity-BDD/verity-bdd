package allure_reporter

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/reporting"
)

// AllureReporter writes Verity BDD events in Allure 2 result format.
type AllureReporter struct {
	resultsDir string

	mutex   sync.Mutex
	current *runningTest
}

type runningTest struct {
	uuid      string
	name      string
	startMs   int64
	steps     []allureStepResult
	openSteps []openStep
}

type openStep struct {
	name        string
	startMs     int64
	children    []allureStepResult
	attachments []allureAttachment
}

type allureResult struct {
	UUID          string             `json:"uuid"`
	Name          string             `json:"name"`
	Status        string             `json:"status"`
	StatusDetails *allureStatus      `json:"statusDetails,omitempty"`
	Start         int64              `json:"start"`
	Stop          int64              `json:"stop"`
	Steps         []allureStepResult `json:"steps,omitempty"`
	Attachments   []allureAttachment `json:"attachments,omitempty"`
}

type allureStatus struct {
	Message string `json:"message,omitempty"`
}

type allureStepResult struct {
	Name        string             `json:"name"`
	Status      string             `json:"status"`
	Start       int64              `json:"start"`
	Stop        int64              `json:"stop"`
	Steps       []allureStepResult `json:"steps,omitempty"`
	Attachments []allureAttachment `json:"attachments,omitempty"`
}

type allureAttachment struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

// NewAllureReporterWithDir creates a reporter that stores results in dir.
func NewAllureReporterWithDir(dir string) *AllureReporter {
	return &AllureReporter{resultsDir: dir}
}

func (ar *AllureReporter) SetOutput(_ io.Writer) {}

func (ar *AllureReporter) OnTestStart(testName string) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	ar.current = &runningTest{
		uuid:    mustUUID(),
		name:    testName,
		startMs: time.Now().UnixMilli(),
	}
}

// OnLog attaches the resolved log values to the currently open Log step.
func (ar *AllureReporter) OnLog(entry reporting.LogEntry) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.current == nil || len(ar.current.openSteps) == 0 {
		return
	}

	attachments := ar.persistAttachments([]reporting.Attachment{{
		Name:        "log",
		ContentType: "text/plain",
		Content:     []byte(strings.Join(entry.Values, " ")),
	}})
	index := len(ar.current.openSteps) - 1
	ar.current.openSteps[index].attachments = append(ar.current.openSteps[index].attachments, attachments...)
}

func (ar *AllureReporter) OnStepStart(stepDescription string) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.current == nil {
		return
	}

	ar.current.openSteps = append(ar.current.openSteps, openStep{
		name:    stepDescription,
		startMs: time.Now().UnixMilli(),
	})
}

func (ar *AllureReporter) OnStepFinish(stepResult reporting.TestResult) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.current == nil {
		return
	}

	startMs := time.Now().UnixMilli()
	name := stepResult.Name()
	var finished openStep
	if len(ar.current.openSteps) > 0 {
		finished = ar.current.openSteps[len(ar.current.openSteps)-1]
		ar.current.openSteps = ar.current.openSteps[:len(ar.current.openSteps)-1]
		startMs = finished.startMs
		if name == "" {
			name = finished.name
		}
	}

	stopMs := endTime(startMs, stepResult.Duration())

	attachments := ar.persistAttachments(stepResult.Attachments())
	step := allureStepResult{
		Name:        name,
		Status:      mapStatus(stepResult.Status()),
		Start:       startMs,
		Stop:        stopMs,
		Steps:       finished.children,
		Attachments: append(finished.attachments, attachments...),
	}

	if len(ar.current.openSteps) == 0 {
		ar.current.steps = append(ar.current.steps, step)
		return
	}

	parentIndex := len(ar.current.openSteps) - 1
	ar.current.openSteps[parentIndex].children = append(ar.current.openSteps[parentIndex].children, step)
}

func (ar *AllureReporter) OnTestFinish(result reporting.TestResult) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.current == nil {
		ar.current = &runningTest{
			uuid:    mustUUID(),
			name:    result.Name(),
			startMs: time.Now().UnixMilli(),
		}
	}

	if ar.current.name == "" {
		ar.current.name = result.Name()
	}

	statusDetails := (*allureStatus)(nil)
	if result.Error() != nil {
		statusDetails = &allureStatus{Message: result.Error().Error()}
	}

	out := allureResult{
		UUID:          ar.current.uuid,
		Name:          ar.current.name,
		Status:        mapStatus(result.Status()),
		StatusDetails: statusDetails,
		Start:         ar.current.startMs,
		Stop:          endTime(ar.current.startMs, result.Duration()),
		Steps:         ar.current.steps,
		Attachments:   ar.persistAttachments(result.Attachments()),
	}

	_ = os.MkdirAll(ar.resultsDir, 0o750)
	body, err := json.Marshal(out)
	if err == nil {
		_ = os.WriteFile(filepath.Join(ar.resultsDir, out.UUID+"-result.json"), body, 0o600)
	}

	ar.current = nil
}

func (ar *AllureReporter) persistAttachments(attachments []reporting.Attachment) []allureAttachment {
	if len(attachments) == 0 {
		return nil
	}

	_ = os.MkdirAll(ar.resultsDir, 0o750)

	result := make([]allureAttachment, 0, len(attachments))
	for _, att := range attachments {
		source := mustUUID() + "-attachment"
		if err := os.WriteFile(filepath.Join(ar.resultsDir, source), att.Content, 0o600); err != nil {
			continue
		}

		result = append(result, allureAttachment{
			Name:   att.Name,
			Source: source,
			Type:   att.ContentType,
		})
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func endTime(startMs int64, durationSeconds float64) int64 {
	if durationSeconds <= 0 {
		return startMs + 1
	}

	stopMs := startMs + int64(durationSeconds*1000)
	if stopMs <= startMs {
		return startMs + 1
	}

	return stopMs
}

func mapStatus(status reporting.Status) string {
	switch status {
	case reporting.StatusFailed:
		return "failed"
	case reporting.StatusSkipped:
		return "skipped"
	default:
		return "passed"
	}
}

func mustUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}
