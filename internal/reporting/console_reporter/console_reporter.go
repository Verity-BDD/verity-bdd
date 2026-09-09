package console_reporter

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/reporting"
)

// activeStep represents a currently executing step
type activeStep struct {
	description string
	indentLevel int
	startTime   time.Time
}

// ConsoleReporter provides console-based test reporting
type ConsoleReporter struct {
	output      io.Writer
	currentTest string
	indentLevel int
	mutex       sync.RWMutex
	activeSteps map[string]*activeStep // key: description + indent
}

// NewConsoleReporter creates a new console reporter
func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{
		output:      os.Stdout,
		activeSteps: make(map[string]*activeStep),
	}
}

// SetOutput sets the output destination
func (cr *ConsoleReporter) SetOutput(w io.Writer) {
	cr.output = w
}

// OnLog renders a Log entry at the current nesting level.
func (cr *ConsoleReporter) OnLog(entry reporting.LogEntry) {
	cr.writeLine("%sℹ️ %s: %s", cr.getIndent(), entry.ActorName, strings.Join(entry.Values, " "))
}

// OnTestStart is called when a test begins
func (cr *ConsoleReporter) OnTestStart(testName string) {
	cr.mutex.Lock()
	cr.currentTest = testName
	cr.indentLevel = 0
	cr.mutex.Unlock()
	cr.writeLine("🚀 Starting: %s", testName)
}

// OnTestFinish is called when a test completes
func (cr *ConsoleReporter) OnTestFinish(result reporting.TestResult) {
	emoji := "✅"
	statusText := "PASSED"

	switch result.Status() {
	case reporting.StatusFailed:
		emoji = "❌"
		statusText = "FAILED"
	case reporting.StatusSkipped:
		emoji = "⏭️"
		statusText = "SKIPPED"
	}

	cr.writeLine("%s %s: %s (%.2fs)", emoji, result.Name(), statusText, result.Duration())

	if result.Error() != nil {
		cr.writeLine("   Error: %s", result.Error().Error())
	}

	attachments := result.Attachments()
	if len(attachments) > 0 {
		cr.writeLine("   Attachments:")
		for _, attachment := range attachments {
			cr.writeLine("   - %s (%s): %s", attachment.Name, attachment.ContentType, string(attachment.Content))
		}
	}

	cr.writeLine("")
}

// OnStepStart is called when a step/activity begins
func (cr *ConsoleReporter) OnStepStart(stepDescription string) {
	cr.mutex.Lock()
	cr.indentLevel++
	description := cr.formatStepDescription(stepDescription)
	indentLevel := cr.indentLevel
	cr.mutex.Unlock()

	// Add to active steps for tracking
	cr.addActiveStep(description, indentLevel)
}

// OnStepFinish is called when a step/activity completes
func (cr *ConsoleReporter) OnStepFinish(stepResult reporting.TestResult) {
	cr.mutex.Lock()
	description := cr.formatStepDescription(stepResult.Name())
	indentLevel := cr.indentLevel
	cr.mutex.Unlock()

	// Remove from active steps
	cr.removeActiveStep(description, indentLevel)

	emoji := "✅"
	if stepResult.Status() == reporting.StatusFailed {
		emoji = "❌"
	}

	indent := cr.getIndent()

	// Overwrite the current line with completion status
	cr.writeOverLine("%s%s %s (%.2fs)", indent, emoji, description, stepResult.Duration())

	// Handle error output on separate line if there's an error
	if stepResult.Error() != nil {
		cr.writeLine("%s   Error: %s", indent, stepResult.Error().Error())
	}

	cr.mutex.Lock()
	cr.indentLevel--
	cr.mutex.Unlock()
}

// formatStepDescription formats step descriptions for better readability
func (cr *ConsoleReporter) formatStepDescription(description string) string {
	// Remove #actor prefix only if no actor name is present (backward compatibility)
	formatted := description
	if len(formatted) >= 7 && formatted[:7] == "#actor " {
		// Check if it's a plain #actor without actor name
		remaining := formatted[7:] // Remove "#actor "
		if len(remaining) > 0 && !strings.HasPrefix(remaining, " ") {
			// This looks like "#actorSomething" not "#actor Something"
			formatted = remaining
		} else {
			// This is "#actor " pattern, remove it
			formatted = remaining
		}
	} else {
		// No #actor prefix, description likely already has actor name
		// Just ensure proper capitalization
		formatted = description
	}

	// Capitalize first letter only if it's not already capitalized
	if len(formatted) > 0 && formatted[0] >= 'a' && formatted[0] <= 'z' {
		formatted = strings.ToUpper(formatted[:1]) + formatted[1:]
	}

	return formatted
}

// getStepKey generates a unique key for a step
func (cr *ConsoleReporter) getStepKey(description string, indentLevel int) string {
	return fmt.Sprintf("%d:%s", indentLevel, description)
}

// addActiveStep adds a new active step
func (cr *ConsoleReporter) addActiveStep(description string, indentLevel int) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	key := cr.getStepKey(description, indentLevel)
	cr.activeSteps[key] = &activeStep{
		description: description,
		indentLevel: indentLevel,
		startTime:   time.Now(),
	}
}

// removeActiveStep removes and returns the active step
func (cr *ConsoleReporter) removeActiveStep(description string, indentLevel int) *activeStep {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	key := cr.getStepKey(description, indentLevel)
	if step, exists := cr.activeSteps[key]; exists {
		delete(cr.activeSteps, key)
		return step
	}
	return nil
}

// writeOverLine clears the current line and writes new content
func (cr *ConsoleReporter) writeOverLine(format string, args ...interface{}) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	if cr.output != nil {
		content := fmt.Sprintf(format, args...)
		_, _ = fmt.Fprintf(cr.output, "%s\n", content)
	}
}

// getIndent returns the current indentation string
func (cr *ConsoleReporter) getIndent() string {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	return strings.Repeat("  ", cr.indentLevel)
}

// writeLine writes a formatted line to the output
func (cr *ConsoleReporter) writeLine(format string, args ...interface{}) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	if cr.output != nil {
		_, _ = fmt.Fprintf(cr.output, format+"\n", args...)
	}
}
