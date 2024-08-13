package cmd

import (
	"log/slog"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal"
	"github.com/wot-oss/tmc/internal/config"
)

func TestLoggingOnSubCommands(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.ConfigDir
	config.ConfigDir = temp
	defer func() { config.ConfigDir = orgDir }()

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
	var importCmd = &cobra.Command{Use: "import", Run: runFunc}

	RootCmd.AddCommand(listCmd, serveCmd, importCmd)

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

	// when: executing the import command
	RootCmd.SetArgs([]string{"import"})
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
	orgDir := config.ConfigDir
	config.ConfigDir = temp
	defer func() { config.ConfigDir = orgDir }()

	config.InitViper()
	RootCmd.ResetCommands()

	var isDisabled bool

	// given: a sub-command of the root command
	runFunc := func(cmd *cobra.Command, args []string) {
		hdl := slog.Default().Handler()
		_, isDisabled = hdl.(*internal.DiscardLogHandler)
	}

	var importCmd = &cobra.Command{Use: "import", Run: runFunc}
	RootCmd.AddCommand(importCmd)

	// when: executing the command with the --loglevel flag
	RootCmd.SetArgs([]string{"import"})
	_ = RootCmd.ParseFlags([]string{"--loglevel", "info"})
	_ = RootCmd.Execute()
	// then: logging is ENABLED
	assert.False(t, isDisabled)
}

func TestEnvVarOverridesDefaultConfigDir(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.ConfigDir
	defer func() { config.ConfigDir = orgDir }()
	const envKey = "TMC_CONFIG"
	orgEnv, unset := os.LookupEnv(envKey)
	err = os.Setenv(envKey, temp)
	assert.NoError(t, err)
	defer func() {
		if unset {
			_ = os.Unsetenv(envKey)
		} else {
			_ = os.Setenv(envKey, orgEnv)
		}
	}()

	RootCmd.ResetCommands()

	// given: a sub-command of the root command
	runFunc := func(cmd *cobra.Command, args []string) {}

	var importCmd = &cobra.Command{Use: "import", Run: runFunc}
	RootCmd.AddCommand(importCmd)

	// when: executing the command with the --loglevel flag
	RootCmd.SetArgs([]string{"import"})
	_ = RootCmd.Execute()
	// then: ConfigDir is set to temp
	assert.Equal(t, temp, config.ConfigDir)
}
func TestConfigFlagOverridesDefaultConfigDir(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.ConfigDir
	defer func() { config.ConfigDir = orgDir }()

	RootCmd.ResetCommands()

	// given: a sub-command of the root command
	runFunc := func(cmd *cobra.Command, args []string) {}

	var importCmd = &cobra.Command{Use: "import", Run: runFunc}
	RootCmd.AddCommand(importCmd)

	// when: executing the command with the --loglevel flag
	RootCmd.SetArgs([]string{"import"})
	_ = RootCmd.ParseFlags([]string{"--config", temp})
	_ = RootCmd.Execute()
	// then: ConfigDir is set to temp
	assert.Equal(t, temp, config.ConfigDir)
}
