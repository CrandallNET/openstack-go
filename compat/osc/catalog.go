package osc

import (
	"embed"
	"encoding/json"
	"path/filepath"
	"strings"
)

//go:embed 9.0.0/commands.json 9.0.0/completion.bash 9.0.0/global-help.txt 9.0.0/help
var files embed.FS

type CommandGroup struct {
	CommandGroup string   `json:"Command Group"`
	Commands     []string `json:"Commands"`
}

func Commands() ([]CommandGroup, error) {
	data, err := files.ReadFile("9.0.0/commands.json")
	if err != nil {
		return nil, err
	}

	var groups []CommandGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func Completion() (string, error) {
	data, err := files.ReadFile("9.0.0/completion.bash")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Help(command string) (string, bool, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		data, err := files.ReadFile("9.0.0/global-help.txt")
		if err != nil {
			return "", false, err
		}
		return string(data), true, nil
	}

	pathParts := append([]string{"9.0.0", "help"}, parts...)
	path := filepath.ToSlash(filepath.Join(pathParts...)) + ".txt"
	data, err := files.ReadFile(path)
	if err != nil {
		return "", false, nil
	}
	return string(data), true, nil
}
