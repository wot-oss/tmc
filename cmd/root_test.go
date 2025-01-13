package cmd

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
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

	config.ReadInConfig()
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

	config.ReadInConfig()
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

func resetSearchFlags(flags *FilterFlags) {
	flags.FilterAuthor = ""
	flags.FilterManufacturer = ""
	flags.FilterMpn = ""
	flags.FilterProtocol = ""
}

func TestConvertFilters(t *testing.T) {

	// given: no filter params set via CLI flags
	flags := FilterFlags{}
	// when: converting to Filters
	params := CreateFiltersFromCLI(flags, "")
	// then: Filters are undefined
	assert.Nil(t, params)

	// given: filter params are set with single values
	resetSearchFlags(&flags)
	flags.FilterAuthor = "some author"
	flags.FilterManufacturer = "some manufacturer"
	flags.FilterMpn = "some mpn"
	flags.FilterProtocol = "http"
	name := "omni-corp/omni"
	// when: converting to Filters
	params = CreateFiltersFromCLI(flags, name)
	// then: the filter values are converted correctly
	assert.NotNil(t, params)
	assert.Equal(t, []string{flags.FilterAuthor}, params.Author)
	assert.Equal(t, []string{flags.FilterManufacturer}, params.Manufacturer)
	assert.Equal(t, []string{flags.FilterMpn}, params.Mpn)
	assert.Equal(t, name, params.Name)
	assert.Equal(t, model.PrefixMatch, params.Options.NameFilterType)
	assert.Equal(t, []string{flags.FilterProtocol}, params.Protocol)

	// given: filter params are set with multiple comma-separated values
	resetSearchFlags(&flags)
	flags.FilterAuthor = "some author 1,some author 2"
	flags.FilterManufacturer = "some manufacturer 1,some manufacturer 2"
	flags.FilterMpn = "some mpn 1,some mpn 2,some mpn 3"
	flags.FilterProtocol = "http,https"
	// when: converting to Filters
	params = CreateFiltersFromCLI(flags, "")
	// then: the multiple filter values are converted correctly
	assert.NotNil(t, params)
	assert.Equal(t, strings.Split(flags.FilterAuthor, ","), params.Author)
	assert.Equal(t, strings.Split(flags.FilterManufacturer, ","), params.Manufacturer)
	assert.Equal(t, strings.Split(flags.FilterMpn, ","), params.Mpn)
	assert.Equal(t, strings.Split(flags.FilterProtocol, ","), params.Protocol)
}
