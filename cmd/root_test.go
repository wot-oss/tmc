package cmd

import (
	"log/slog"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
)

func TestLoggingOnSubCommands(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.DefaultConfigDir
	config.DefaultConfigDir = temp
	defer func() { config.DefaultConfigDir = orgDir }()

	config.InitViper()
	RootCmd.ResetCommands()

	var isDisabled bool

	// given: some sub-commands of the root command
	//        where the "serve" command is to be expected having logging enabled default
	runFunc := func(cmd *cobra.Command, args []string) {
		hdl := slog.Default().Handler()
		_, isDisabled = hdl.(*internal.DiscardLogHandler)
	}

	var listCmd = &cobra.Command{Use: "list", Run: runFunc}
	var serveCmd = &cobra.Command{Use: "serve", Run: runFunc}
	var pushCmd = &cobra.Command{Use: "push", Run: runFunc}

	RootCmd.AddCommand(listCmd, serveCmd, pushCmd)

	// when: executing the list command
	RootCmd.SetArgs([]string{"list"})
	_ = RootCmd.Execute()
	// then: logging is default DISABLED
	assert.True(t, isDisabled)

	// when: executing the serve command
	RootCmd.SetArgs([]string{"serve"})
	_ = RootCmd.Execute()
	// then: logging is default ENABLED
	assert.False(t, isDisabled)

	// when: executing the push command
	RootCmd.SetArgs([]string{"push"})
	_ = RootCmd.Execute()
	// then: logging is default DISABLED
	assert.True(t, isDisabled)
}

func TestLogFlagEnablesLogging(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.DefaultConfigDir
	config.DefaultConfigDir = temp
	defer func() { config.DefaultConfigDir = orgDir }()

	config.InitViper()
	RootCmd.ResetCommands()

	var isDisabled bool

	// given: a sub-command of the root command
	runFunc := func(cmd *cobra.Command, args []string) {
		hdl := slog.Default().Handler()
		_, isDisabled = hdl.(*internal.DiscardLogHandler)
	}

	var pushCmd = &cobra.Command{Use: "push", Run: runFunc}
	RootCmd.AddCommand(pushCmd)

	// when: executing the command with the --log flag
	RootCmd.SetArgs([]string{"push"})
	_ = RootCmd.ParseFlags([]string{"--log", ""})
	_ = RootCmd.Execute()
	// then: logging is ENABLED
	assert.False(t, isDisabled)
}
