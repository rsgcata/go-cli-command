package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
)

const StatusOk = 0
const StatusErr = 1

// FlagDefinition represents a command-line flag definition
type FlagDefinition struct {
	name        string
	description string
	required    bool
	defaultVal  string
	setupFlag   func(*flag.FlagSet)
}

func NewFlagDefinition(
	name string,
	description string,
	required bool,
	defaultVal string,
	setupFlag func(*flag.FlagSet),
) FlagDefinition {
	return FlagDefinition{name, description, required, defaultVal, setupFlag}
}

// Name returns the name of the flag
func (def FlagDefinition) Name() string {
	return def.name
}

// Description returns the description of the flag
func (def FlagDefinition) Description() string {
	return def.description
}

// Required returns whether the flag is required
func (def FlagDefinition) Required() bool {
	return def.required
}

// DefaultValue returns the default value of the flag
func (def FlagDefinition) DefaultValue() string {
	return def.defaultVal
}

// FlagDefinitionMap is a map of flag names to their definitions
type FlagDefinitionMap map[string]FlagDefinition

// Command interface defines the methods that a command must implement
type Command interface {
	Id() string
	Description() string
	Exec(flagSet *flag.FlagSet, stdWriter io.Writer) error
	FlagDefinitions() FlagDefinitionMap
}

// setupFlagSet creates and configures a flag.FlagSet for the given command
func setupFlagSet(cmd Command, outputWriter io.Writer) *flag.FlagSet {
	flagSet := flag.NewFlagSet(cmd.Id(), flag.ContinueOnError)
	flagSet.Usage = func() {
		_, _ = fmt.Fprintf(outputWriter, "Usage of %s:\n", cmd.Id())
		flagSet.PrintDefaults()
	}

	// Add flags based on command's flag definitions
	for _, def := range cmd.FlagDefinitions() {
		def.setupFlag(flagSet)
	}

	return flagSet
}

// validateFlags checks if all required flags are provided
func validateFlags(flagSet *flag.FlagSet, cmd Command) []error {
	var flagErrors []error

	// Check for required flags
	for _, def := range cmd.FlagDefinitions() {
		if def.required {
			lookup := flagSet.Lookup(def.name)
			if lookup == nil || lookup.Value.String() == "" {
				flagErrors = append(
					flagErrors,
					fmt.Errorf("flag '%s' is required", def.name),
				)
			}
		}
	}

	return flagErrors
}

// runCommand runs the given command with the provided arguments
func runCommand(cmd Command, args []string, outputWriter io.Writer) (cmdErr error) {
	defer func() {
		if err := recover(); err != nil {
			cmdErr = err.(error)
		}
	}()

	// Setup flag set for the command
	flagSet := setupFlagSet(cmd, outputWriter)
	flagSet.SetOutput(outputWriter)

	// Parse flags
	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf(
			"Failed to execute command %s with error: %s\n",
			cmd.Id(),
			err.Error(),
		)
	}

	// Validate required flags
	if errs := validateFlags(flagSet, cmd); len(errs) > 0 {
		return fmt.Errorf(
			"Failed to execute command %s with error: %s\n",
			cmd.Id(),
			errors.Join(errs...).Error(),
		)
	}

	// Execute the command
	if cmdErr = cmd.Exec(flagSet, outputWriter); cmdErr != nil {
		return fmt.Errorf(
			"Failed to execute command %s with error: %s\n",
			cmd.Id(),
			cmdErr.Error(),
		)
	}

	return cmdErr
}

// parseCmdInput parses the command name and arguments from the input args
func parseCmdInput(args []string) (cmdName string, cmdArgs []string) {
	if len(args) > 1 {
		if args[0] == "--" {
			args = args[1:]
		}
	}

	if len(args) != 0 {
		cmdName = strings.TrimSpace(args[0])
		cmdArgs = args[1:]
	}

	return
}

// CommandsRegistry holds all registered commands
type CommandsRegistry struct {
	commands map[string]Command
}

// Register adds a command to the registry
func (registry *CommandsRegistry) Register(cmd Command) error {
	if _, exists := registry.commands[cmd.Id()]; exists {
		return fmt.Errorf("command '%s' is already registered", cmd.Id())
	}
	registry.commands[cmd.Id()] = cmd
	return nil
}

// Commands returns a copy of all registered commands
func (registry *CommandsRegistry) Commands() map[string]Command {
	cmdCopy := make(map[string]Command, len(registry.commands))
	for name, cmd := range registry.commands {
		cmdCopy[name] = cmd
	}
	return cmdCopy
}

// Command returns a command by its ID
func (registry *CommandsRegistry) Command(id string) (Command, bool) {
	cmd, ok := registry.commands[id]
	return cmd, ok
}

// Bootstrap Will bootstrap everything needed for the user CLI request. Will process the
// user input and run the requested command. By default, will output to os.Stdout if
// nil is provided for the io.Writer argument.
func Bootstrap(
	args []string,
	availableCommands CommandsRegistry,
	outputWriter io.Writer,
	processExit func(code int),
) {
	if outputWriter == nil {
		outputWriter = os.Stdout
	}

	if processExit == nil {
		processExit = os.Exit
	}

	_ = availableCommands.Register(
		&HelpCommand{slices.Collect(maps.Values(availableCommands.Commands()))},
	)
	cmdId, cmdArgs := parseCmdInput(args)
	if cmdId == "" {
		cmdId = (&HelpCommand{}).Id()
	}

	var cmdErr error
	cmd, exists := availableCommands.Command(cmdId)
	if !exists {
		cmdErr = fmt.Errorf("The command %s does not exist\n", cmdId)
	} else {
		cmdErr = runCommand(cmd, cmdArgs, outputWriter)
	}

	if cmdErr != nil {
		_, outputErr := outputWriter.Write(
			[]byte(
				fmt.Sprintf(
					"Failed to execute command %s with error: %s\n",
					cmdId,
					cmdErr.Error(),
				),
			),
		)
		if outputErr != nil {
			fmt.Printf(
				"Error writing to the provided output writer %s\n",
				reflect.TypeOf(outputWriter),
			)
		}
		processExit(StatusErr)
		return
	}

	processExit(StatusOk)
}
