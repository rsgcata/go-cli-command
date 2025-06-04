package cli

import (
	"bytes"
	"flag"
	"github.com/stretchr/testify/suite"
	"io"
	"testing"
)

type HelpSuite struct {
	suite.Suite
}

func TestHelpSuite(t *testing.T) {
	suite.Run(t, new(HelpSuite))
}

func (s *HelpSuite) TestChunkDescriptionSplitsStringBasedOnProvidedSize() {
	tests := []struct {
		name        string
		description string
		size        int
		want        []string
	}{
		{
			name:        "Empty description",
			description: "",
			size:        10,
			want:        []string{""},
		},
		{
			name:        "Description shorter than chunk size",
			description: "Short text",
			size:        20,
			want:        []string{"Short text"},
		},
		{
			name:        "Description exactly chunk size",
			description: "Exactly 10",
			size:        10,
			want:        []string{"Exactly 10"},
		},
		{
			name:        "Description with newline",
			description: "First line\nSecond line",
			size:        20,
			want:        []string{"First line", "Second line"},
		},
		{
			name:        "Description split on space",
			description: "This is a long description that should be split into multiple chunks",
			size:        20,
			want: []string{
				"This is a long description",
				"that should be split",
				"into multiple chunks",
			},
		},
	}

	for _, scenario := range tests {
		s.Run(
			scenario.name, func() {
				got := chunkDescription(scenario.description, scenario.size)
				s.Equal(
					len(scenario.want),
					len(got),
					"chunkDescription() returned incorrect number of chunks",
				)
				for i := range got {
					s.Equal(scenario.want[i], got[i], "chunkDescription() chunk[%d] incorrect", i)
				}
			},
		)
	}
}

// Mock command for testing
type mockCommand struct {
	id          string
	description string
	flagDefs    FlagDefinitionMap
}

func (m *mockCommand) Id() string {
	return m.id
}

func (m *mockCommand) Description() string {
	return m.description
}

func (m *mockCommand) FlagDefinitions() FlagDefinitionMap {
	return m.flagDefs
}

func (m *mockCommand) Exec(_ *flag.FlagSet, _ io.Writer) error {
	return nil
}

func (s *HelpSuite) TestHelpCommandExecutionCanShowAvailableCommandsInfo() {
	tests := []struct {
		name          string
		commands      []Command
		contentChecks []string
	}{
		{
			name:          "No commands",
			commands:      []Command{},
			contentChecks: []string{"help", "Available CLI Commands"},
		},
		{
			name: "Single command without options",
			commands: []Command{
				&mockCommand{
					id:          "test",
					description: "Test command",
					flagDefs:    FlagDefinitionMap{},
				},
			},
			contentChecks: []string{
				"help", "Available CLI Commands",
				"_________",
				"test", "Test command",
			},
		},
		{
			name: "Command with options",
			commands: []Command{
				&mockCommand{
					id:          "test",
					description: "Test command",
					flagDefs: FlagDefinitionMap{
						"option1": {
							name:        "option1",
							description: "First option",
							defaultVal:  "default1",
						},
					},
				},
			},
			contentChecks: []string{
				"help", "Available CLI Commands",
				"_________",
				"test", "Test command",
				"Options",
				"--option1", "First option", "default1",
			},
		},
	}

	for _, scenario := range tests {
		s.Run(
			scenario.name, func() {
				cmd := &HelpCommand{
					availableCommands: scenario.commands,
				}

				var buf bytes.Buffer
				flagSet := flag.NewFlagSet("help", flag.ContinueOnError)
				err := cmd.Exec(flagSet, &buf)
				s.NoError(err, "HelpCommand.Exec() should not return an error")

				output := buf.String()
				for _, check := range scenario.contentChecks {
					s.Contains(output, check, "HelpCommand.Exec() output should contain %q", check)
				}
			},
		)
	}
}
