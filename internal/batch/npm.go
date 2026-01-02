//go:build linux || darwin

package batch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type packageJSON struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

// DetectNpmScript analyzes a Node process to find which npm script it's running.
// Returns the script name (e.g., "dev", "build", "test:watch") or entry file.
func DetectNpmScript(cmdline string, workDir string) string {
	// Strategy 1: Parse cmdline for "npm run <script>" pattern
	if idx := strings.Index(cmdline, "npm run "); idx != -1 {
		rest := cmdline[idx+8:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Strategy 2: Parse cmdline for "yarn <script>" pattern (not install/add)
	if idx := strings.Index(cmdline, "yarn "); idx != -1 {
		rest := cmdline[idx+5:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			script := parts[0]
			// Skip yarn commands that aren't scripts
			if script != "install" && script != "add" && script != "remove" && script != "upgrade" {
				return "yarn:" + script
			}
		}
	}

	// Strategy 3: Parse cmdline for "pnpm run <script>" or "pnpm <script>" pattern
	if idx := strings.Index(cmdline, "pnpm run "); idx != -1 {
		rest := cmdline[idx+9:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	if idx := strings.Index(cmdline, "pnpm "); idx != -1 {
		rest := cmdline[idx+5:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			script := parts[0]
			if script != "install" && script != "add" && script != "remove" && script != "update" {
				return "pnpm:" + script
			}
		}
	}

	// Strategy 4: Parse cmdline for "npx <command>" pattern
	if idx := strings.Index(cmdline, "npx "); idx != -1 {
		rest := cmdline[idx+4:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			cmd := parts[0]
			// Remove version specifier if present (e.g., "tsx@latest" -> "tsx")
			if atIdx := strings.Index(cmd, "@"); atIdx > 0 {
				cmd = cmd[:atIdx]
			}
			return "npx:" + cmd
		}
	}

	// Strategy 5: Look for package.json and match command to scripts
	if workDir != "" {
		pkgPath := filepath.Join(workDir, "package.json")
		if pkg, err := readPackageJSON(pkgPath); err == nil {
			// Try to match cmdline against known scripts
			for name, script := range pkg.Scripts {
				// Extract the main executable from the script
				// e.g., "vite --port 3000" -> "vite"
				scriptParts := strings.Fields(script)
				if len(scriptParts) > 0 {
					mainCmd := scriptParts[0]
					// Match if the main command appears as a separate word in cmdline
					// This avoids false positives like "vite" matching "invite"
					cmdlineLower := strings.ToLower(cmdline)
					mainCmdLower := strings.ToLower(mainCmd)
					// Check for word boundary match (including start of cmdline)
					if strings.Contains(cmdlineLower, " "+mainCmdLower+" ") ||
						strings.HasSuffix(cmdlineLower, " "+mainCmdLower) ||
						strings.Contains(cmdlineLower, "/"+mainCmdLower+" ") ||
						strings.HasSuffix(cmdlineLower, "/"+mainCmdLower) ||
						strings.HasPrefix(cmdlineLower, mainCmdLower+" ") ||
						cmdlineLower == mainCmdLower {
						return name
					}
				}
			}
		}
	}

	// Strategy 6: Extract entry file from cmdline
	// e.g., "node server.js" → "server.js"
	// e.g., "node dist/index.js" → "dist/index.js"
	// e.g., "/usr/local/bin/node /path/to/script.js" → "script.js"
	if idx := strings.LastIndex(cmdline, "node "); idx != -1 {
		rest := cmdline[idx+5:]
		parts := strings.Fields(rest)
		file := ""
		// Node flags that take an argument value
		flagsWithArgs := map[string]bool{
			"-r": true, "--require": true,
			"-e": true, "--eval": true,
			"-p": true, "--print": true,
			"-c": true, "--check": true,
			"--import": true, "--experimental-loader": true,
			"--loader": true, "--input-type": true,
			"--conditions": true, "-C": true,
		}
		skipNext := false
		for _, p := range parts {
			if skipNext {
				skipNext = false
				continue
			}
			if strings.HasPrefix(p, "-") {
				// Check if this flag takes an argument
				// Handle both "--require=module" and "--require module"
				flagName := p
				if eqIdx := strings.Index(p, "="); eqIdx != -1 {
					flagName = p[:eqIdx]
				}
				if flagsWithArgs[flagName] && !strings.Contains(p, "=") {
					skipNext = true
				}
				continue
			}
			file = p
			break
		}
		if file != "" {
			// Get just the filename if it's an absolute path
			if filepath.IsAbs(file) {
				file = filepath.Base(file)
			}
			// Remove workdir prefix if present
			if workDir != "" {
				file = strings.TrimPrefix(file, workDir+"/")
			}
			return file
		}
	}

	return "-"
}

func readPackageJSON(path string) (*packageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
