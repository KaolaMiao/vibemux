package driver

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveExecutablePath(command string) (string, bool) {
	if command == "" {
		return "", false
	}
	if filepath.IsAbs(command) || strings.Contains(command, string(os.PathSeparator)) {
		if _, err := os.Stat(command); err == nil {
			return command, true
		}
		return "", false
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return "", false
	}
	return path, true
}

func isNodeScript(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	if !strings.HasPrefix(line, "#!") {
		return false, nil
	}
	return strings.Contains(line, "node"), nil
}

func findNodeBinary() (string, error) {
	if path, err := exec.LookPath("node"); err == nil {
		return path, nil
	}

	home, _ := os.UserHomeDir()
	candidates := []string{}
	if nvm := os.Getenv("NVM_BIN"); nvm != "" {
		candidates = append(candidates, filepath.Join(nvm, "node"))
	}
	if volta := os.Getenv("VOLTA_HOME"); volta != "" {
		candidates = append(candidates, filepath.Join(volta, "bin", "node"))
	}

	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates,
			"/opt/homebrew/bin/node",
			"/usr/local/bin/node",
			"/opt/homebrew/opt/node/bin/node",
			filepath.Join(home, ".local/bin/node"),
		)
	case "linux":
		candidates = append(candidates,
			"/usr/bin/node",
			"/usr/local/bin/node",
			"/snap/bin/node",
			filepath.Join(home, ".local/bin/node"),
		)
	case "windows":
		candidates = append(candidates,
			filepath.Join(home, "AppData", "Local", "Programs", "nodejs", "node.exe"),
		)
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil {
			if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
				return path, nil
			}
		}
	}

	return "", errors.New("node not found")
}

func splitCommandLine(input string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, errors.New("unfinished escape sequence in command")
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote in command")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}
