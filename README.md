# go-cli-command

A lightweight, flexible framework for building command-line applications in Go. This package provides a simple way to define, register, and execute CLI commands with support for flags and help documentation.

## Features

- Simple command registration and execution
- Flag-based command-line argument parsing
- Built-in help command for displaying available commands
- Support for required and optional flags
- Customizable output handling
- Error handling and exit code management

## Installation

```bash
go get github.com/rsgcata/go-cli-command
```

## Quick Start

Here's a simple example of how to use the package:

```go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rsgcata/go-cli-command/cli"
)

// Define a custom command
type GreetCommand struct{}

func (c *GreetCommand) Id() string {
	return "greet"
}

func (c *GreetCommand) Description() string {
	return "Greets a person with a custom message"
}

func (c *GreetCommand) FlagDefinitions() cli.FlagDefinitionMap {
	return cli.FlagDefinitionMap{
		"name": cli.NewFlagDefinition(
			"name",
			"Name of the person to greet",
			true,  // required
			func(fs *flag.FlagSet) {
				fs.String("name", "", "Name of the person to greet")
			},
		),
		"message": cli.NewFlagDefinition(
			"message",
			"Custom greeting message",
			false, // optional
			func(fs *flag.FlagSet) {
				fs.String("message", "Hello", "Custom greeting message")
			},
		),
		"enabled": cli.NewFlagDefinition(
			"enabled",
			"Is enabled",
			false, // optional
			func(fs *flag.FlagSet) {
				fs.Bool("message", true, "Custom greeting message")
			},
		),
	}
}

func (c *GreetCommand) Exec(flagSet *flag.FlagSet, writer io.Writer) error {
	name := flagSet.Lookup("name").Value.String()
	message := flagSet.Lookup("message").Value.String()

	_, err := fmt.Fprintf(writer, "%s, %s!\n", message, name)
	return err
}

func main() {
	// Create a registry for commands
	registry := &cli.CommandsRegistry{}

	// Register our custom command
	_ = registry.Register(&GreetCommand{})

	// Bootstrap the CLI application
	cli.Bootstrap(os.Args[1:], registry, os.Stdout, os.Exit)
}
```

To run the command:

```bash
go run main.go greet --name John --message Hi
```

Output:
```
Hi, John!
```

## API Documentation

### Core Components

#### Command Interface

The `Command` interface defines the methods that a command must implement:

```go
type Command interface {
	Id() string                                           // Unique identifier for the command
	Description() string                                  // Human-readable description
	FlagDefinitions() FlagDefinitionMap                   // Defines the flags the command accepts
	Exec(flagSet *flag.FlagSet, stdWriter io.Writer) error // Executes the command
}
```

#### FlagDefinition

The `FlagDefinition` struct represents a command-line flag:

```go
type FlagDefinition struct {
	name        string
	description string
	required    bool
	defaultVal  string
	setupFlag   func(*flag.FlagSet)
}
```

Use `NewFlagDefinition` to create a new flag definition:

```go
func NewFlagDefinition(
	name string,
	description string,
	required bool,
	defaultVal string,
	setupFlag func(*flag.FlagSet),
) FlagDefinition
```

#### CommandsRegistry

The `CommandsRegistry` struct manages command registration:

```go
type CommandsRegistry struct {
	commands map[string]Command
}
```

Methods:
- `Register(cmd Command) error`: Registers a new command
- `Commands() map[string]Command`: Returns a copy of all registered commands
- `Command(id string) (Command, bool)`: Returns a command by its ID

#### Bootstrap Function

The `Bootstrap` function is the main entry point for running commands:

```go
func Bootstrap(
	args []string,
	availableCommands CommandsRegistry,
	outputWriter io.Writer,
	processExit func(code int),
)
```

## Advanced Usage

### Custom Output Writers

You can provide a custom output writer to the `Bootstrap` function:

```go
var buf bytes.Buffer
cli.Bootstrap(os.Args[1:], registry, &buf, os.Exit)
```

### Custom Exit Handlers

You can provide a custom exit handler to the `Bootstrap` function:

```go
customExit := func(code int) {
	fmt.Printf("Exiting with code: %d\n", code)
	os.Exit(code)
}

cli.Bootstrap(os.Args[1:], registry, os.Stdout, customExit)
```
