package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func jsonRows(jsonText string) []map[string]any {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonText), &rows); err != nil {
		return nil
	}
	return rows
}

func jsonRowString(row map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := row[name]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func jsonRowInt(row map[string]any, names ...string) int {
	text := jsonRowString(row, names...)
	if text == "" {
		return 0
	}
	value, _ := strconv.Atoi(text)
	return value
}

func jsonStringField(jsonText string, names ...string) string {
	var values map[string]any
	if err := json.Unmarshal([]byte(jsonText), &values); err != nil {
		return ""
	}
	for _, name := range names {
		if value, ok := values[name]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "command is unavailable on this cloud"
}

func looksDeleted(result stepResult) bool {
	text := strings.ToLower(result.Error + "\n" + result.Stderr + "\n" + result.Stdout)
	return strings.Contains(text, "not found") ||
		strings.Contains(text, "could not find") ||
		strings.Contains(text, "no ") ||
		strings.Contains(text, "404")
}

func envKeys(extraEnv map[string]string) []string {
	var env []string
	for key, value := range extraEnv {
		env = append(env, key+"="+value)
	}
	sort.Strings(env)
	return env
}

func envMapValues(extraEnv map[string]string) []string {
	return envKeys(extraEnv)
}

func firstActiveImageID(jsonText string) string {
	fallback := ""
	for _, row := range jsonRows(jsonText) {
		id := jsonRowString(row, "ID", "id")
		name := strings.ToLower(jsonRowString(row, "Name", "name"))
		status := strings.ToLower(jsonRowString(row, "Status", "status"))
		if id == "" || status != "active" {
			continue
		}
		if strings.Contains(name, "cirros") {
			return id
		}
		if fallback == "" {
			fallback = id
		}
	}
	return fallback
}

func smallestFlavorID(jsonText string) string {
	rows := jsonRows(jsonText)
	bestID := ""
	bestRAM := 0
	bestDisk := 0
	for _, row := range rows {
		id := jsonRowString(row, "ID", "id")
		if id == "" {
			continue
		}
		ram := jsonRowInt(row, "RAM", "ram")
		disk := jsonRowInt(row, "Disk", "disk")
		if bestID == "" || ram < bestRAM || (ram == bestRAM && disk < bestDisk) {
			bestID = id
			bestRAM = ram
			bestDisk = disk
		}
	}
	return bestID
}

func firstNetworkID(jsonText string) string {
	for _, row := range jsonRows(jsonText) {
		id := jsonRowString(row, "ID", "id")
		if id != "" {
			return id
		}
	}
	return ""
}
