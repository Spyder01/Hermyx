package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"hermyx/pkg/engine"
)

// --- HELP TEXT IMPROVEMENTS ---

func printRootHelp() {
	fmt.Println(`Hermyx - Blazing Fast Reverse Proxy with Smart Caching 🚀

Usage:
  hermyx <command> [options]

Available Commands:
  up        Start the Hermyx reverse proxy using a YAML configuration file.
  down      Gracefully stop the Hermyx proxy server defined in the config file.
  init      Generate (scaffold) a default Hermyx configuration YAML file.
  version   Display the current Hermyx version.
  help      Show detailed help for a specific command.

Examples:
  hermyx init --config ./hermyx.config.yaml
  hermyx up --config ./configs/prod.yaml
  hermyx down --config ./hermyx.config.yaml

Run 'hermyx help <command>' for details on a specific command.`)
}

func printInitHelp() {
	fmt.Println(`Hermyx Init Command 🧩

Usage:
  hermyx init [--config <path>]

Description:
  Creates (scaffolds) a default Hermyx configuration YAML file. 
  Use this command to quickly get started with a base configuration template.

Options:
  --config   Path to save the Hermyx config YAML file. 
             (default: ./hermyx.config.yaml)

Example:
  hermyx init --config ./configs/dev.yaml`)
}

func printUpHelp() {
	fmt.Println(`Hermyx Up Command 🚀

Usage:
  hermyx up [--config <path>]

Description:
  Starts the Hermyx reverse proxy server using the specified YAML configuration file.
  This will launch the proxy, load caching rules, and begin handling traffic.

Options:
  --config   Path to the Hermyx configuration YAML file. 
             (default: ./hermyx.config.yaml)

Example:
  hermyx up --config ./configs/prod.yaml`)
}

func printDownHelp() {
	fmt.Println(`Hermyx Down Command ⏹️

Usage:
  hermyx down [--config <path>]

Description:
  Gracefully shuts down the Hermyx proxy server defined by the given configuration.
  Ensures all active connections are safely closed before exiting.

Options:
  --config   Path to the Hermyx configuration YAML file. 
             (default: ./hermyx.config.yaml)

Example:
  hermyx down --config ./hermyx.config.yaml`)
}

func printVersionHelp() {
	fmt.Println(`Hermyx Version Command 🧾

Usage:
  hermyx version

Description:
  Displays the current version of Hermyx installed on your system.`)
}

// --- VERSION INFORMATION ---

const Version = "0.1.0"

func printVersion() {
	fmt.Printf("Hermyx version: %s\n", Version)
}

// --- MAIN ENTRYPOINT ---

func main() {
	if len(os.Args) < 2 {
		printRootHelp()
		os.Exit(1)
	}

	switch os.Args[1] {

	case "up":
		runCmd := flag.NewFlagSet("up", flag.ExitOnError)
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

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Config file not found: %s\n", absPath)
			os.Exit(1)
		}

		proxyEngine := engine.InstantiateHermyxEngine(absPath)
		proxyEngine.Run()

	case "down":
		runCmd := flag.NewFlagSet("down", flag.ExitOnError)
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

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Config file not found: %s\n", absPath)
			os.Exit(1)
		}

		err = engine.KillHermyx(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to stop Hermyx server at %s: %v\n", *configPath, err)
			os.Exit(1)
		}
		fmt.Printf("✅ Hermyx server shut down successfully using config: %s\n", *configPath)

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
			fmt.Fprintf(os.Stderr, "Unable to scaffold Hermyx config at %s: %v\n", *configPath, err)
			os.Exit(1)
		}

		fmt.Printf("✅ Hermyx configuration scaffolded successfully at: %s\n", absPath)

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
