package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func CreateDockerImage(ctx context.Context, repo model.RepoSpec, imageTag string, outputFile string, format string, inputName, inputMaintainer, inputVersion string) error {
	defer func() {
		fmt.Println("Cleaning up ./docker_context directory")
		if err := os.RemoveAll("./docker_context/"); err != nil {
			fmt.Printf("Warning: Failed to remove ./docker_context/: %v\n", err)
		}
	}()

	repos, _ := repos.ReadConfig()

	fmt.Println("Processing repositories")
	for name, repo := range repos {
		repoType, _ := repo["type"].(string)
		switch repoType {
		case "file":
			fmt.Printf("Copying local file/directory for repo: %s\n", name)
			if err := copyLocalRepo(repo["loc"].(string), "./docker_context/data/"+name); err != nil {
				return fmt.Errorf("failed to copy local repo: %w", err)
			}
			repo["type"] = "file"
			repo["loc"] = "/docker/repos/" + name
			repo["description"] = ""
		case "http":
			fmt.Printf("Downloading HTTP repo for: %s\n", name)
			if err := pullRemoteRepo(repo["loc"].(string), "./docker_context/data/"+name); err != nil {
				return fmt.Errorf("failed to pull remote repo: %w", err)
			}
			repo["type"] = "file"
			repo["loc"] = "/docker/repos/" + name
			repo["description"] = ""
		default:
			return fmt.Errorf("unknown repo type: %s", repoType)
		}
	}
	fmt.Println("Writing updated repos to config.json")
	if err := writeUpdatedConfig(repos, "./docker_context/config/config.json"); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}
	copyDirectory("./docker/", "./docker_context/")
	imageName := inputName
	if imageName == "" {
		imageName = "W3C Thing Model Catalog"
	}

	imageMaintainer := inputMaintainer
	if imageMaintainer == "" {
		imageMaintainer = "https://github.com/wot-oss"
	}

	imageVersion := inputVersion
	if imageVersion == "" {
		imageVersion = "latest"
	}

	buildArgs := []string{
		"--build-arg", fmt.Sprintf("NAME=\"%s\"", imageName),
		"--build-arg", fmt.Sprintf("MAINTAINER=\"%s\"", imageMaintainer),
		"--build-arg", fmt.Sprintf("VERSION=\"%s\"", imageVersion),
	}
	buildCmdArgs := []string{
		"build",
		"--progress=plain",
		"--no-cache",
		"-t", imageTag,
	}
	buildCmdArgs = append(buildCmdArgs, buildArgs...)
	buildCmdArgs = append(buildCmdArgs, "./docker_context", "-f", "./docker_context/Dockerfile_localdeployment")

	fmt.Println("Building Docker image")
	buildCmd := exec.Command("docker", buildCmdArgs...)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	saveCmd := exec.Command("docker", "save", "-o", outputFile, imageTag)
	saveCmd.Stdout = os.Stdout
	saveCmd.Stderr = os.Stderr
	if err := saveCmd.Run(); err != nil {
		return fmt.Errorf("failed to save Docker image: %w", err)
	}
	fmt.Println("\nSaved docker image to tarball " + outputFile)
	os.RemoveAll("/docker_context/")
	return nil
}

func writeUpdatedConfig(repos map[string]map[string]interface{}, configPath string) error {
	updatedConfig := map[string]interface{}{
		"repos": repos,
	}
	configPath = filepath.Clean(configPath)
	_, err := os.Stat(filepath.Dir(configPath))
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(configPath), 0770)
	}
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config.json for writing: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(updatedConfig); err != nil {
		return fmt.Errorf("failed to encode config.json: %w", err)
	}
	fmt.Println("Successfully updated config.json.")
	return nil
}

func copyLocalRepo(src, dest string) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source path error: %w", err)
	}
	if info.IsDir() {
		return copyDirectory(src, dest)
	} else {
		return copyFile(src, dest)
	}
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file '%s': %w", src, err)
	}
	convertedContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	err = os.WriteFile(dest, []byte(convertedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write converted content to destination file '%s': %w", dest, err)
	}
	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	return os.Chmod(dest, srcInfo.Mode())
}

func copyDirectory(src, dest string) error {
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy subdirectory: %w", err)
			}
		} else {
			if err := copyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
		}
	}
	return nil
}

func pullRemoteRepo(url, dest string) error {
	cmd := exec.Command("git", "clone", formatGitHubURL(url), dest)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Printf("Attempting to clone repository from '%s' into '%s'\n", formatGitHubURL(url), dest)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to clone repository '%s' into '%s': %w\nGit Stdout: %s\nGit Stderr: %s", formatGitHubURL(url), dest, err, stdout.String(), stderr.String())
	}

	fmt.Printf("Successfully cloned repository to '%s'.\n", dest)
	if stdout.Len() > 0 {
		fmt.Printf("Git output: %s\n", stdout.String())
	}

	return nil
}

func formatGitHubURL(url string) string {
	if strings.Contains(url, "raw.githubusercontent.com") {
		url = strings.Replace(url, "raw.githubusercontent.com", "github.com", 1)
		url = strings.Replace(url, "/refs/heads/main", ".git", 1)
	}
	return url
}
