package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillInstallCreatesStandardSkillTree(t *testing.T) {
	application, out, errOut := testApplication("http://127.0.0.1:1", t.TempDir())
	skillsRoot := filepath.Join(t.TempDir(), "nested", "skills")

	code := application.run([]string{"skill", "install", skillsRoot})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}

	destination := filepath.Join(skillsRoot, "neu-box")
	for _, relative := range []string{
		"SKILL.md",
		filepath.Join("agents", "openai.yaml"),
		filepath.Join("references", "cli.md"),
		filepath.Join("references", "http-api.md"),
	} {
		info, err := os.Stat(filepath.Join(destination, relative))
		if err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
			t.Fatalf("unexpected mode for %s: %s", relative, info.Mode())
		}
	}

	skill, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skill), "name: neu-box") ||
		!strings.Contains(string(skill), "Worker API v2 with `curl`") ||
		!strings.Contains(string(skill), "neu-sbox wait") {
		t.Fatalf("unexpected SKILL.md: %s", skill)
	}
	httpReference, err := os.ReadFile(filepath.Join(destination, "references", "http-api.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(httpReference), "POST") ||
		!strings.Contains(string(httpReference), "/tasks") ||
		!strings.Contains(string(httpReference), "api_version") {
		t.Fatalf("unexpected HTTP reference: %s", httpReference)
	}
	if !strings.Contains(out.String(), destination) {
		t.Fatalf("missing destination in output: %s", out.String())
	}
}

func TestSkillInstallRequiresSkillsRoot(t *testing.T) {
	application, _, errOut := testApplication("http://127.0.0.1:1", t.TempDir())
	code := application.run([]string{"skill", "install"})
	if code != 2 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "技能根目录") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}
