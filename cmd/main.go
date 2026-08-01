package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"hermyx/pkg/engine"
)

func printRootHelp() {
	fmt.Println(`hermyx - blazing fast reverse proxy with smart caching

Usage:
  hermyx <command> [options]

Available Commands:
  up        Start the Hermyx reverse proxy
  down      Close the Hermyx reverse proxy
  init      Scaffold hermyx config yaml.
  help      Show help for a command
  version   Display hermyx version

Examples:

  hermyx up --config ./hermyx.config.yaml
  hermyx down --config ./hermyx.config.yaml
  hermyx init
  hermyx help up

Run 'hermyx help <command>' for details on a specific command.`)
}

func printInitHelp() {
	fmt.Println(`hermyx init - scaffold a default Hermyx config YAML file

Creates a starter hermyx.config.yaml (or the path given via --config) with
default logging, server, storage, and cache sections pre-filled so you can
start editing right away instead of writing a config from scratch.

Usage:
  hermyx init [--config <path>]

Options:
  --config   Path to Hermyx config YAML file to create (default: ./hermyx.config.yaml)

Examples:
  # Scaffold ./hermyx.config.yaml in the current directory
  hermyx init

  # Scaffold a config at a custom path
  hermyx init --config ./configs/prod.yaml`)
}

func printUpHelp() {
	fmt.Println(`hermyx up - start the Hermyx reverse proxy

Reads the given config file, starts the proxy server on the configured port,
and begins routing and caching requests according to the configured routes.
Writes a PID file under the configured storage path so 'hermyx down' can
later find and stop this instance. Runs in the foreground until it receives
a shutdown signal (Ctrl+C) or is stopped via 'hermyx down'.

Usage:
  hermyx up [--config <path>]

Options:
  --config   Path to Hermyx config YAML file (default: ./hermyx.config.yaml)

Examples:
  # Start using ./hermyx.config.yaml in the current directory
  hermyx up

  # Start using a specific config file
  hermyx up --config ./configs/prod.yaml`)
}

func printDownHelp() {
	fmt.Println(`hermyx down - stop a running Hermyx reverse proxy

Reads the PID file recorded by a running 'hermyx up' instance (found via the
storage path in the given config file) and gracefully stops that process.

Usage:
  hermyx down [--config <path>]

Options:
  --config   Path to Hermyx config YAML file (default: ./hermyx.config.yaml)

Examples:
  # Stop the instance started with ./hermyx.config.yaml
  hermyx down

  # Stop the instance started with a specific config file
  hermyx down --config ./configs/prod.yaml`)
}

func printVersionHelp() {
	fmt.Println(`hermyx version - display the installed Hermyx version

Usage:
  hermyx version

Examples:
  hermyx version`)
}

const Version = "0.1.0"

func printVersion() {
	fmt.Printf("Hermyx version: %s", Version)
}

func main() {
	if len(os.Args) < 2 {
		printRootHelp()
		os.Exit(1)
	}

	switch os.Args[1] {

	case "up":
		runCmd := flag.NewFlagSet("up", flag.ExitOnError)
		configPath := runCmd.String("config", "hermyx.config.yaml", "Path to configuration YAML file")

		// Parse flags after "up"
		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse flags: %v\n", err)
			os.Exit(1)
		}

		absPath, err := filepath.Abs(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to resolve config path: %v\n", err)
			os.Exit(1)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Config file not found: %s\n", absPath)
			os.Exit(1)
		}

		proxyEngine := engine.InstantiateHermyxEngine(absPath)
		proxyEngine.Run()

	case "down":
		runCmd := flag.NewFlagSet("up", flag.ExitOnError)
		configPath := runCmd.String("config", "hermyx.config.yaml", "Path to configuration YAML file")

		// Parse flags after "up"
		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse flags: %v\n", err)
			os.Exit(1)
		}

		absPath, err := filepath.Abs(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to resolve config path: %v\n", err)
			os.Exit(1)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Config file not found: %s\n", absPath)
			os.Exit(1)
		}

		err = engine.KillHermyx(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to kill the hermyx server at %s: %v\n", *configPath, err)
			os.Exit(1)
		}
		fmt.Printf("Shut down hermyx server at %s \n", *configPath)

	case "init":
		runCmd := flag.NewFlagSet("init", flag.ExitOnError)
		configPath := runCmd.String("config", "hermyx.config.yaml", "Path to configuration YAML file")

		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse flags: %v\n", err)
			os.Exit(1)
		}

		absPath, err := filepath.Abs(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to resolve config path: %v\n", err)
			os.Exit(1)
		}

		err = engine.InitConfig(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to scaffold hermyx config at %s: %v\n", *configPath, err)
			os.Exit(1)
		}

	case "version":
		printVersion()

	case "help":
		if len(os.Args) == 2 {
			printRootHelp()
		} else {
			switch os.Args[2] {
			case "up":
				printUpHelp()
			case "down":
				printDownHelp()
			case "init":
				printInitHelp()
			case "version":
				printVersionHelp()
			default:
				fmt.Printf("Unknown help topic: %s\n", os.Args[2])
				printRootHelp()
				os.Exit(1)
			}
		}

	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printRootHelp()
		os.Exit(1)
	}
}
