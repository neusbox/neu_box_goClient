package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	embeddedSkillRoot = "skill/neu-box"
	skillName         = "neu-box"
)

//go:embed skill/neu-box
var embeddedSkill embed.FS

func (a *app) runSkill(args []string) int {
	if len(args) != 2 || args[0] != "install" || strings.TrimSpace(args[1]) == "" {
		return a.usageError("用法: neu-sbox skill install <技能根目录>")
	}

	skillsRoot, err := expandUserPath(args[1])
	if err != nil {
		return a.internalError("skill_path_invalid", err)
	}
	skillsRoot, err = filepath.Abs(skillsRoot)
	if err != nil {
		return a.internalError("skill_path_invalid", fmt.Errorf("解析技能目录: %w", err))
	}
	destination := filepath.Join(skillsRoot, skillName)
	installed, err := installEmbeddedSkill(destination)
	if err != nil {
		return a.internalError("skill_install_failed", fmt.Errorf("安装 Neu Box skill: %w", err))
	}

	if a.jsonOutput {
		_ = printJSONValue(a.out, map[string]any{
			"files": installed,
			"name":  skillName,
			"path":  destination,
		})
		return 0
	}
	fmt.Fprintln(a.out, "[neu-sbox] Neu Box skill 已安装")
	fmt.Fprintf(a.out, "    path: %s\n", destination)
	fmt.Fprintf(a.out, "    files: %d\n", len(installed))
	return 0
}

func expandUserPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("读取用户主目录: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}

func installEmbeddedSkill(destination string) ([]string, error) {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录 %s: %w", destination, err)
	}

	installed := make([]string, 0, 3)
	err := fs.WalkDir(embeddedSkill, embeddedSkillRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(embeddedSkillRoot, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("内置 skill 包含越界路径")
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("内置 skill 包含不支持的文件类型: %s", path)
		}
		content, err := embeddedSkill.ReadFile(path)
		if err != nil {
			return err
		}
		if err := atomicWriteSkillFile(target, content); err != nil {
			return err
		}
		installed = append(installed, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return installed, nil
}

func atomicWriteSkillFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建目录 %s: %w", filepath.Dir(path), err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".neu-box-skill-*")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("写入 %s: %w", path, err)
	}
	return nil
}
