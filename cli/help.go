package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type HelpCommand struct {
	availableCommands []Command
}

func (c *HelpCommand) Id() string {
	return "help"
}

func (c *HelpCommand) Description() string {
	return "Lists all available commands"
}

func (c *HelpCommand) InputDefinition() InputOptionDefinitionMap {
	return InputOptionDefinitionMap{}
}

func (c *HelpCommand) Exec(_ InputOptionsMap, baseWriter io.Writer) error {
	writer := tabwriter.NewWriter(baseWriter, 0, 0, 1, ' ', 0)
	_, _ = fmt.Fprintln(writer, c.Id()+"\tAvailable CLI Commands:")

	for _, command := range c.availableCommands {
		_, _ = fmt.Fprintln(writer, "_________\t")

		descChunks := chunkDescription(command.Description(), 80)
		_, _ = fmt.Fprintln(writer, command.Id()+"\t"+descChunks[0])
		if len(descChunks) > 1 {
			for _, descChunk := range descChunks[1:] {
				_, _ = fmt.Fprintln(writer, "\t"+descChunk)
			}
		}

		if len(command.InputDefinition()) > 0 {
			_, _ = fmt.Fprintln(writer, "\tOptions:")
			for _, def := range command.InputDefinition() {
				_, _ = fmt.Fprintf(
					writer,
					"\t--%s %s (default %s)\n",
					def.name,
					def.description,
					def.defaultVal,
				)
			}
		}
	}
	_ = writer.Flush()

	return nil
}

func chunkDescription(description string, size int) []string {
	if len(description) == 0 {
		return []string{""}
	}

	var chunks []string
	accumulator := ""
	for _, char := range description {
		accumulator += string(char)
		if (len(accumulator) >= size && string(char) == " ") || string(char) == "\n" {
			chunks = append(chunks, strings.TrimSpace(accumulator))
			accumulator = ""
		}
	}

	if len(accumulator) > 0 {
		chunks = append(chunks, accumulator)
	}

	return chunks
}
