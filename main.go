package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	syscall "syscall"

	"v2ray.com/core"
	"v2ray.com/core/common/cmdarg"
	"v2ray.com/core/common/platform"
	_ "v2ray.com/core/main/confloader/external"
	_ "v2ray.com/core/main/distro/all"
)

var (
	configFiles cmdarg.Arg // config file(s) path
	configDir   string
	version     = flag.Bool("version", false, "Show current version of V2Ray.")
	testConfig  = flag.Bool("test", false, "Test the configuration and exit without starting.")
	// Changed default format to yaml since I prefer yaml configs over json
	format      = flag.String("format", "yaml", "Format of input file, supported formats: json, pb, toml, yaml")
)

func fileExists(file string) bool {
	info, err := os.Stat(file)
	return err == nil && !info.IsDir()
}

func dirExists(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func readConfDir(dirPath string) {
	confsDir, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config directory: %v\n", err)
		return
	}
	for _, f := range confsDir {
		ext := filepath.Ext(f.Name())
		if strings.EqualFold(ext, ".json") || strings.EqualFold(ext, ".yaml") ||
			strings.EqualFold(ext, ".toml") || strings.EqualFold(ext, ".pb") {
			configFiles = append(configFiles, filepath.Join(dirPath, f.Name()))
		}
	}
}

func getConfigFilePath() {
	if dirExists(configDir) {
		readConfDir(configDir)
	} else if configDir != "" {
		fmt.Fprintf(os.Stderr, "config directory %s not found\n", configDir)
	}

	if len(configFiles) > 0 {
		return
	}

	// Try default config locations
	if path := platform.GetConfigurationPath(); fileExists(path) {
		configFiles = append(configFiles, path)
	}
}

func startV2Ray() (core.Server, error) {
	getConfigFilePath()

	config, err := core.LoadConfig(*format, configFiles[0], configFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	server, err := core.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return server, nil
}

func main() {
	// Improve performance on multi-core systems
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Var(&configFiles, "config", "Config file(s) for V2Ray. Multiple assign is accepted (e.g., -config=a.json -config=b.json).")
	flag.Var(&configFiles, "c", "Short alias for -config.")
	flag.StringVar(&configDir, "confdir", "", "Directory for config files. All config files in the directory will be loaded.")
	flag.Parse()

	if *version {
		fmt.Fprintf(os.Stdout, "%s\n", core.VersionStatement())
		return
	}

	server, err := startV2Ray()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		// Configuration error, exit with code 23
		os.Exit(23)
	}

	if *testConfig {
		fmt.Println("Configuration OK.")
		return
	}

	if err := server.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start:", err)
		os.Exit(1)
	}
	defer server.Close()

	// Handle OS signals for graceful shutdown
	// Listening for SIGTERM and SIGINT so Ctrl+C and service stops work cleanly
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
	<-osSignals
	fmt.Println("V2Ray shutting down.")
}
