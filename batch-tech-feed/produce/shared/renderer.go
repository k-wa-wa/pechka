package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	RemotionEntry       = "src/index.ts"
	RemotionComposition = "Digest"
)

func RunRender(manifest *Manifest, outDir, dst string, concurrency, crf int, quiet bool) (string, error) {
	ex, err := os.Executable()
	var baseDir string
	if err == nil {
		baseDir = filepath.Dir(ex)
	}
	projectDir, err := filepath.Abs(filepath.Join(baseDir, "remotion"))
	if err != nil || !isDir(projectDir) {
		projectDir, _ = filepath.Abs("remotion")
	}

	if !isDir(filepath.Join(projectDir, "node_modules")) {
		return "", fmt.Errorf("%s/node_modules not found; run `npm install` in that directory first", projectDir)
	}

	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return "", fmt.Errorf("npx not found on PATH (Node.js is required to render)")
	}

	propsPath := filepath.Join(outDir, "props.json")
	propsMap := map[string]interface{}{
		"manifest": manifest,
	}
	propsBytes, err := json.Marshal(propsMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal props JSON: %w", err)
	}
	if err := os.WriteFile(propsPath, propsBytes, 0o644); err != nil {
		return "", fmt.Errorf("failed to write props file: %w", err)
	}

	absDst, err := filepath.Abs(dst)
	if err != nil {
		absDst = dst
	}

	absProps, err := filepath.Abs(propsPath)
	if err != nil {
		absProps = propsPath
	}

	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		absOutDir = outDir
	}

	args := []string{
		"remotion", "render", RemotionEntry, RemotionComposition, absDst,
		fmt.Sprintf("--props=%s", absProps),
		fmt.Sprintf("--public-dir=%s", absOutDir),
		fmt.Sprintf("--crf=%d", crf),
	}

	if concurrency > 0 {
		args = append(args, fmt.Sprintf("--concurrency=%d", concurrency))
	}
	if quiet {
		args = append(args, "--log=error")
	}

	cmd := exec.Command(npxPath, args...)
	cmd.Dir = projectDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if !quiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("remotion render failed (%w): %s", err, stderr.String())
	}

	return absDst, nil
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
