package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

func AttachmentList(spec model.RepoSpec, tmNameOrId string) error {
	list, err := commands.AttachmentList(context.Background(), spec, tmNameOrId)
	if err != nil {
		Stderrf("Could not list attachments for %s: %v", tmNameOrId, err)
		return err
	}

	for _, v := range list {
		fmt.Println(v)
	}
	return nil
}

func AttachmentPush(spec model.RepoSpec, tmNameOrId, filename string) error {
	abs, err := filepath.Abs(filename)
	if err != nil {
		Stderrf("Error expanding file name %s: %v", filename, err)
		return err
	}

	stat, err := os.Stat(abs)
	if err != nil || stat.IsDir() {
		Stderrf("Cannot read file %s: %v", filename, err)
		return err
	}
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("Couldn't read file %s: %v", filename, err)
	}
	err = commands.AttachmentPush(context.Background(), spec, tmNameOrId, filepath.Base(filename), raw)
	if err != nil {
		Stderrf("Failed to put attachment %s to %s: %v", filename, tmNameOrId, err)
	}

	return err
}
func AttachmentDelete(spec model.RepoSpec, tmNameOrId, attachmentName string) error {
	err := commands.AttachmentDelete(context.Background(), spec, tmNameOrId, attachmentName)
	if err != nil {
		Stderrf("Failed to delete attachment %s to %s: %v", attachmentName, tmNameOrId, err)
	}

	return err
}
func AttachmentFetch(spec model.RepoSpec, tmNameOrId, attachmentName string) error {
	content, err := commands.AttachmentFetch(context.Background(), spec, tmNameOrId, attachmentName)
	if err != nil {
		Stderrf("Failed to delete attachment %s to %s: %v", attachmentName, tmNameOrId, err)
	}

	fmt.Print(string(content))
	return nil
}
