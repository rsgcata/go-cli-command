package cli

import (
	"errors"
	"fmt"
	"github.com/rsgcata/go-params/params"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
)

const StatusOk = 0
const StatusErr = 1

type InputOptionDefinition struct {
	name        string
	description string
	required    bool
	defaultVal  string
}

func (def InputOptionDefinition) Name() string {
	return def.name
}

func (def InputOptionDefinition) Description() string {
	return def.description
}

func (def InputOptionDefinition) Required() bool {
	return def.required
}

func (def InputOptionDefinition) DefaultValue() string {
	return def.defaultVal
}

type InputOption struct {
	InputOptionDefinition
	rawVal string
}

func (opt InputOption) RawVal() params.RawVal {
	return params.RawVal(opt.rawVal)
}

type InputOptionDefinitionMap map[string]InputOptionDefinition
type InputOptionsMap map[string]InputOption

type Command interface {
	Id() string
	Description() string
	Exec(options InputOptionsMap, stdWriter io.Writer) error
	InputDefinition() InputOptionDefinitionMap
}

func BuildOptionsFrom(
	rawOptions []string,
	cmd Command,
) (InputOptionsMap, []error) {
	options := InputOptionsMap{}
	var optionErrors []error
	for _, arg := range rawOptions {
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		parts := strings.SplitN(strings.TrimLeft(arg, "--"), "=", 2)
		if len(parts) < 1 {
			continue
		}

		optionName := strings.TrimSpace(parts[0])
		if _, exists := options[optionName]; exists {
			optionErrors = append(
				optionErrors,
				fmt.Errorf("option '%s' is defined twice", optionName),
			)
		}

		optionValue := ""
		if len(parts) == 2 {
			optionValue = strings.TrimSpace(parts[1])
		}

		options[optionName] = InputOption{
			InputOptionDefinition: cmd.InputDefinition()[optionName],
			rawVal:                optionValue,
		}
	}

	for _, optionDef := range cmd.InputDefinition() {
		option, optionSet := options[optionDef.name]
		if optionDef.required && (!optionSet || option.rawVal == "") {
			optionErrors = append(
				optionErrors,
				fmt.Errorf("option '%s' is required", optionDef.name),
			)
		}
	}

	return options, optionErrors
}

func runCommand(cmd Command, rawOptions []string, outputWriter io.Writer) (cmdErr error) {
	defer func() {
		if err := recover(); err != nil {
			cmdErr = err.(error)
		}
	}()

	optionsMap, errs := BuildOptionsFrom(rawOptions, cmd)
	if len(errs) > 0 {
		return fmt.Errorf(
			"Failed to execute command %s with error: %s\n",
			cmd.Id(),
			errors.Join(errs...).Error(),
		)
	}

	if cmdErr = cmd.Exec(optionsMap, outputWriter); cmdErr != nil {
		return fmt.Errorf(
			"Failed to execute command %s with error: %s\n",
			cmd.Id(),
			cmdErr.Error(),
		)
	}

	return cmdErr
}

func parseCmdInput(args []string) (cmdName string, rawOptions []string) {
	if len(args) > 1 {
		if args[0] == "--" {
			args = args[1:]
		}
	}

	if len(args) != 0 {
		cmdName = strings.TrimSpace(args[0])
		rawOptions = args[1:]
	}

	return
}

type CommandsRegistry struct {
	commands map[string]Command
}

func (registry *CommandsRegistry) Register(cmd Command) error {
	if _, exists := registry.commands[cmd.Id()]; exists {
		return fmt.Errorf("command '%s' is already registered", cmd.Id())
	}
	registry.commands[cmd.Id()] = cmd
	return nil
}

func (registry *CommandsRegistry) Commands() map[string]Command {
	cmdCopy := make(map[string]Command, len(registry.commands))
	for name, cmd := range registry.commands {
		cmdCopy[name] = cmd
	}
	return cmdCopy
}

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
	cmdId, rawOptions := parseCmdInput(args)
	if cmdId == "" {
		cmdId = (&HelpCommand{}).Id()
	}

	var cmdErr error
	cmd, exists := availableCommands.Command(cmdId)
	if !exists {
		cmdErr = fmt.Errorf("The command %s does not exist\n", cmdId)
	} else {
		cmdErr = runCommand(cmd, rawOptions, outputWriter)
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
