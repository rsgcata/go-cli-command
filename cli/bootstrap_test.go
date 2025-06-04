package cli

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/suite"
	"io"
	"testing"
)

type BootstrapSuite struct {
	suite.Suite
}

func TestBootstrapSuite(t *testing.T) {
	suite.Run(t, new(BootstrapSuite))
}

// Mock command for testing bootstrap functionality
type bootstrapMockCommand struct {
	id          string
	description string
	inputDef    InputOptionDefinitionMap
	execFunc    func(options InputOptionsMap, writer io.Writer) error
}

func (m *bootstrapMockCommand) Id() string {
	return m.id
}

func (m *bootstrapMockCommand) Description() string {
	return m.description
}

func (m *bootstrapMockCommand) InputDefinition() InputOptionDefinitionMap {
	return m.inputDef
}

func (m *bootstrapMockCommand) Exec(options InputOptionsMap, writer io.Writer) error {
	if m.execFunc != nil {
		return m.execFunc(options, writer)
	}
	return nil
}

func (s *BootstrapSuite) TestItCanBuildOptionsFromRawOptions() {
	tests := []struct {
		name        string
		rawOptions  []string
		cmd         Command
		wantOptions InputOptionsMap
		wantErrors  bool
	}{
		{
			name:       "Empty options",
			rawOptions: []string{},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
			},
			wantOptions: InputOptionsMap{},
			wantErrors:  false,
		},
		{
			name:       "Valid option",
			rawOptions: []string{"--option1=value1"},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"option1": {
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
				},
			},
			wantOptions: InputOptionsMap{
				"option1": {
					InputOptionDefinition: InputOptionDefinition{
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
					rawVal: "value1",
				},
			},
			wantErrors: false,
		},
		{
			name:       "Option without value",
			rawOptions: []string{"--option1"},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"option1": {
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
				},
			},
			wantOptions: InputOptionsMap{
				"option1": {
					InputOptionDefinition: InputOptionDefinition{
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
					rawVal: "",
				},
			},
			wantErrors: false,
		},
		{
			name:       "Missing required option",
			rawOptions: []string{},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"option1": {
						name:        "option1",
						description: "Option 1",
						required:    true,
					},
				},
			},
			wantOptions: InputOptionsMap{},
			wantErrors:  true,
		},
		{
			name:       "Required option with empty value",
			rawOptions: []string{"--option1="},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"option1": {
						name:        "option1",
						description: "Option 1",
						required:    true,
					},
				},
			},
			wantOptions: InputOptionsMap{
				"option1": {
					InputOptionDefinition: InputOptionDefinition{
						name:        "option1",
						description: "Option 1",
						required:    true,
					},
					rawVal: "",
				},
			},
			wantErrors: true,
		},
		{
			name:       "Duplicate option",
			rawOptions: []string{"--option1=value1", "--option1=value2"},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"option1": {
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
				},
			},
			wantOptions: InputOptionsMap{
				"option1": {
					InputOptionDefinition: InputOptionDefinition{
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
					rawVal: "value2",
				},
			},
			wantErrors: true,
		},
		{
			name:       "Non-option argument",
			rawOptions: []string{"positional", "--option1=value1"},
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"option1": {
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
				},
			},
			wantOptions: InputOptionsMap{
				"option1": {
					InputOptionDefinition: InputOptionDefinition{
						name:        "option1",
						description: "Option 1",
						required:    false,
					},
					rawVal: "value1",
				},
			},
			wantErrors: false,
		},
	}

	for _, scenario := range tests {
		s.Run(
			scenario.name, func() {
				options, errs := BuildOptionsFrom(scenario.rawOptions, scenario.cmd)

				if scenario.wantErrors {
					s.NotEmpty(errs, "BuildOptionsFrom() should return errors")
				} else {
					s.Empty(errs, "BuildOptionsFrom() should not return errors")
				}

				s.Equal(
					len(scenario.wantOptions),
					len(options),
					"BuildOptionsFrom() returned incorrect number of options",
				)

				for name, wantOption := range scenario.wantOptions {
					gotOption, exists := options[name]
					s.True(exists, "BuildOptionsFrom() should return option %s", name)
					s.Equal(
						wantOption.rawVal,
						gotOption.rawVal,
						"BuildOptionsFrom() option %s has incorrect value",
						name,
					)
				}
			},
		)
	}
}

func (s *BootstrapSuite) TestItCanParseCmdInput() {
	tests := []struct {
		name        string
		args        []string
		wantCmdName string
		wantOptions []string
	}{
		{
			name:        "Empty args",
			args:        []string{},
			wantCmdName: "",
			wantOptions: nil,
		},
		{
			name:        "Command only",
			args:        []string{"command"},
			wantCmdName: "command",
			wantOptions: []string{},
		},
		{
			name:        "Command with options",
			args:        []string{"command", "--option1=value1", "--option2=value2"},
			wantCmdName: "command",
			wantOptions: []string{"--option1=value1", "--option2=value2"},
		},
		{
			name:        "Command with -- prefix",
			args:        []string{"--", "command", "--option1=value1"},
			wantCmdName: "command",
			wantOptions: []string{"--option1=value1"},
		},
		{
			name:        "Command with whitespace",
			args:        []string{" command "},
			wantCmdName: "command",
			wantOptions: []string{},
		},
	}

	for _, scenario := range tests {
		s.Run(
			scenario.name, func() {
				cmdName, options := parseCmdInput(scenario.args)
				s.Equal(
					scenario.wantCmdName,
					cmdName,
					"parseCmdInput() returned incorrect command name",
				)
				s.Equal(scenario.wantOptions, options, "parseCmdInput() returned incorrect options")
			},
		)
	}
}

func (s *BootstrapSuite) TestItCanRegisterCommands() {
	s.Run(
		"Register adds command to registry", func() {
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd := &bootstrapMockCommand{id: "test", description: "Test command"}

			err := registry.Register(cmd)
			s.NoError(err, "Register() should not return an error")

			registeredCmd, exists := registry.commands["test"]
			s.True(exists, "Register() should add command to registry")
			s.Equal(cmd, registeredCmd, "Register() should store the correct command")
		},
	)

	s.Run(
		"Register returns error for duplicate command", func() {
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd1 := &bootstrapMockCommand{id: "test", description: "Test command 1"}
			cmd2 := &bootstrapMockCommand{id: "test", description: "Test command 2"}

			_ = registry.Register(cmd1)
			err := registry.Register(cmd2)

			s.Error(err, "Register() should return an error for duplicate command")
			s.Contains(
				err.Error(),
				"already registered",
				"Register() error should mention the command is already registered",
			)
		},
	)

	s.Run(
		"Commands returns copy of commands map", func() {
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd := &bootstrapMockCommand{id: "test", description: "Test command"}

			_ = registry.Register(cmd)
			commands := registry.Commands()

			s.Equal(1, len(commands), "Commands() should return all registered commands")
			s.Equal(cmd, commands["test"], "Commands() should return the correct command")

			// Verify it's a copy by modifying the returned map
			delete(commands, "test")
			s.Equal(
				1,
				len(registry.commands),
				"Commands() should return a copy of the commands map",
			)
		},
	)

	s.Run(
		"Command returns registered command", func() {
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd := &bootstrapMockCommand{id: "test", description: "Test command"}

			_ = registry.Register(cmd)
			gotCmd, exists := registry.Command("test")

			s.True(exists, "Command() should find registered command")
			s.Equal(cmd, gotCmd, "Command() should return the correct command")
		},
	)

	s.Run(
		"Command returns false for non-existent command", func() {
			registry := &CommandsRegistry{commands: make(map[string]Command)}

			_, exists := registry.Command("nonexistent")
			s.False(exists, "Command() should return false for non-existent command")
		},
	)
}

func (s *BootstrapSuite) TestItCanRunCommand() {
	tests := []struct {
		name           string
		cmd            Command
		rawOptions     []string
		setupCmd       func(cmd *bootstrapMockCommand)
		expectedOutput string
		expectError    bool
		errorContains  []string
	}{
		{
			name: "Successful command execution",
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
			},
			rawOptions: []string{},
			setupCmd: func(cmd *bootstrapMockCommand) {
				cmd.execFunc = func(options InputOptionsMap, writer io.Writer) error {
					_, _ = writer.Write([]byte("Command executed successfully"))
					return nil
				}
			},
			expectedOutput: "Command executed successfully",
			expectError:    false,
		},
		{
			name: "Command execution with option errors",
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef: InputOptionDefinitionMap{
					"required": {
						name:        "required",
						description: "Required option",
						required:    true,
					},
				},
			},
			rawOptions:     []string{},
			expectedOutput: "",
			expectError:    true,
			errorContains:  []string{"Failed to execute command", "required"},
		},
		{
			name: "Command execution with command error",
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
			},
			rawOptions: []string{},
			setupCmd: func(cmd *bootstrapMockCommand) {
				cmd.execFunc = func(options InputOptionsMap, writer io.Writer) error {
					return errors.New("command error")
				}
			},
			expectedOutput: "",
			expectError:    true,
			errorContains:  []string{"Failed to execute command", "command error"},
		},
		{
			name: "Command execution with panic",
			cmd: &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
			},
			rawOptions: []string{},
			setupCmd: func(cmd *bootstrapMockCommand) {
				cmd.execFunc = func(options InputOptionsMap, writer io.Writer) error {
					panic(errors.New("panic error"))
				}
			},
			expectedOutput: "",
			expectError:    true,
			errorContains:  []string{"panic error"},
		},
	}

	for _, scenario := range tests {
		s.Run(
			scenario.name, func() {
				// Set up the command if needed
				if scenario.setupCmd != nil {
					scenario.setupCmd(scenario.cmd.(*bootstrapMockCommand))
				}

				// Create a buffer to capture output
				var buf bytes.Buffer
				err := runCommand(scenario.cmd, scenario.rawOptions, &buf)

				// Check if error is expected
				if scenario.expectError {
					s.Error(err, "runCommand() should return an error")
					for _, errText := range scenario.errorContains {
						s.Contains(
							err.Error(),
							errText,
							"runCommand() error should contain expected text",
						)
					}
				} else {
					s.NoError(err, "runCommand() should not return an error")
				}

				// Check output
				s.Equal(
					scenario.expectedOutput,
					buf.String(),
					"runCommand() should write expected output to writer",
				)
			},
		)
	}
}

func (s *BootstrapSuite) TestItCanBootstrapCliRunner() {
	s.Run(
		"Successful command execution", func() {
			// Create a mock registry with a test command
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd := &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
				execFunc: func(options InputOptionsMap, writer io.Writer) error {
					_, _ = writer.Write([]byte("Command executed successfully"))
					return nil
				},
			}
			_ = registry.Register(cmd)

			// Mock the exit function to capture the exit code
			var exitCode int
			mockExit := func(code int) {
				exitCode = code
			}

			// Create a buffer to capture output
			var buf bytes.Buffer

			// Call Bootstrap with the test command
			Bootstrap([]string{"test"}, *registry, &buf, mockExit)

			// Verify the command was executed successfully
			s.Equal(StatusOk, exitCode, "Bootstrap should exit with StatusOk")
			s.Equal(
				"Command executed successfully",
				buf.String(),
				"Bootstrap should write command output to writer",
			)
		},
	)

	s.Run(
		"Command not found", func() {
			// Create an empty registry
			registry := &CommandsRegistry{commands: make(map[string]Command)}

			// Mock the exit function to capture the exit code
			var exitCode int
			mockExit := func(code int) {
				exitCode = code
			}

			// Create a buffer to capture output
			var buf bytes.Buffer

			// Call Bootstrap with a non-existent command
			Bootstrap([]string{"nonexistent"}, *registry, &buf, mockExit)

			// Verify the error was handled correctly
			s.Equal(StatusErr, exitCode, "Bootstrap should exit with StatusErr")
			s.Contains(
				buf.String(),
				"does not exist",
				"Bootstrap should write error message to writer",
			)
		},
	)

	s.Run(
		"Command execution with error", func() {
			// Create a mock registry with a test command that returns an error
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd := &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
				execFunc: func(options InputOptionsMap, writer io.Writer) error {
					return errors.New("command execution error")
				},
			}
			_ = registry.Register(cmd)

			// Mock the exit function to capture the exit code
			var exitCode int
			mockExit := func(code int) {
				exitCode = code
			}

			// Create a buffer to capture output
			var buf bytes.Buffer

			// Call Bootstrap with the test command
			Bootstrap([]string{"test"}, *registry, &buf, mockExit)

			// Verify the error was handled correctly
			s.Equal(StatusErr, exitCode, "Bootstrap should exit with StatusErr")
			s.Contains(
				buf.String(),
				"command execution error",
				"Bootstrap should write error message to writer",
			)
		},
	)

	s.Run(
		"Default to help command when no command specified", func() {
			// Create a mock registry
			registry := &CommandsRegistry{commands: make(map[string]Command)}

			// Mock the exit function to capture the exit code
			var exitCode int
			mockExit := func(code int) {
				exitCode = code
			}

			// Create a buffer to capture output
			var buf bytes.Buffer

			// Call Bootstrap with no command
			Bootstrap([]string{}, *registry, &buf, mockExit)

			// Verify help command was executed
			s.Equal(
				StatusOk,
				exitCode,
				"Bootstrap should exit with StatusOk when running help command",
			)
		},
	)

	s.Run(
		"Use default output writer when none provided", func() {
			// Create a mock registry with a test command
			registry := &CommandsRegistry{commands: make(map[string]Command)}
			cmd := &bootstrapMockCommand{
				id:          "test",
				description: "Test command",
				inputDef:    InputOptionDefinitionMap{},
				execFunc: func(options InputOptionsMap, writer io.Writer) error {
					return nil
				},
			}
			_ = registry.Register(cmd)

			// Mock the exit function to capture the exit code
			var exitCode int
			mockExit := func(code int) {
				exitCode = code
			}

			// Call Bootstrap with nil output writer
			Bootstrap([]string{"test"}, *registry, nil, mockExit)

			// Verify the command was executed successfully
			s.Equal(StatusOk, exitCode, "Bootstrap should exit with StatusOk")
		},
	)
}
