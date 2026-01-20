package steps

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// saveOutput saves Claude's output to structured log files
func (s *AgentStep) saveOutput(logDir, output string, loopCount int) {
	if logDir == "" {
		return
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("warning: failed to create log directory %s: %v", logDir, err)
		return
	}

	// Strip STDERR section if present (Claude CLI appends this after JSON)
	jsonOutput := output
	if idx := strings.Index(output, "\n--- STDERR ---"); idx != -1 {
		jsonOutput = strings.TrimSpace(output[:idx])
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &jsonData); err == nil {
		// Valid JSON - pretty print and save to loop_N.json
		jsonPath := filepath.Join(logDir, fmt.Sprintf("loop_%d.json", loopCount))
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			prettyJSON = []byte(jsonOutput) // fallback to original
		}
		if err := os.WriteFile(jsonPath, prettyJSON, 0644); err != nil {
			log.Printf("warning: failed to write JSON log %s: %v", jsonPath, err)
		}

		// Extract 'result' field and save to loop_N.md
		if result, ok := jsonData["result"].(string); ok && result != "" {
			mdPath := filepath.Join(logDir, fmt.Sprintf("loop_%d.md", loopCount))
			if err := os.WriteFile(mdPath, []byte(result), 0644); err != nil {
				log.Printf("warning: failed to write markdown log %s: %v", mdPath, err)
			}
		}
	} else {
		// Not valid JSON - fall back to .log file
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("claude_output_%s_loop%d.log", timestamp, loopCount)
		path := filepath.Join(logDir, filename)
		if err := os.WriteFile(path, []byte(output), 0644); err != nil {
			log.Printf("warning: failed to write fallback log %s: %v", path, err)
		}
	}
}
