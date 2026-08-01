package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestSkillsIndexLinksExistingFiles(t *testing.T) {
	body, err := os.ReadFile("skills/README.md")
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile(`\[[^]]+\]\(([^):#]+\.md)\)`)
	for _, match := range links.FindAllStringSubmatch(string(body), -1) {
		if _, err := os.Stat(filepath.Join("skills", filepath.Clean(match[1]))); err != nil {
			t.Errorf("skills index link %q: %v", match[1], err)
		}
	}
}

func TestRCCGuideCoversEveryToolkitTask(t *testing.T) {
	toolkit, err := os.ReadFile("../developer/toolkit.yaml")
	if err != nil {
		t.Fatal(err)
	}
	guide, err := os.ReadFile("skills/rcc-development.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range taskNames(string(toolkit), "tasks") {
		command := "rcc run -r developer/toolkit.yaml -t " + task
		if !strings.Contains(string(guide), command) {
			t.Errorf("regular RCC task lacks documented command %q", command)
		}
	}
	for _, task := range taskNames(string(toolkit), "devTasks") {
		command := "rcc run -r developer/toolkit.yaml --dev -t " + task
		if !strings.Contains(string(guide), command) {
			t.Errorf("development RCC task lacks documented command %q", command)
		}
	}
}

func TestRCCGuideKeepsEvidenceBoundariesExplicit(t *testing.T) {
	body, err := os.ReadFile("skills/rcc-development.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{
		"`test` is Docker-free",
		"`integration` is explicit",
		"host-mutating commands are excluded",
		"a passing Robot run is not release or deployment proof",
	} {
		if !strings.Contains(string(body), phrase) {
			t.Errorf("RCC guide lacks evidence boundary %q", phrase)
		}
	}
}

func taskNames(document, section string) []string {
	header := section + ":"
	inSection := false
	name := regexp.MustCompile(`^  ([A-Za-z][A-Za-z0-9]*):$`)
	var names []string
	for _, line := range strings.Split(document, "\n") {
		if line == header {
			inSection = true
			continue
		}
		if inSection && line != "" && line[0] != ' ' {
			break
		}
		if inSection {
			if match := name.FindStringSubmatch(line); match != nil {
				names = append(names, match[1])
			}
		}
	}
	sort.Strings(names)
	return names
}
