package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"

	"github.com/aliwatters/rod-mcp/types"
)

func main() {
	subCfg, err := RunCmd()
	if err != nil {
		log.Error(err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := types.LoadConfig(subCfg.ConfigPath)
	if err != nil {
		log.Errorf("Load config error: %s", err)
		return
	}
	// init logger
	types.InitLogger(cfg.LoggerConfig)

	if subCfg.Headless {
		cfg.Headless = true
	}

	if subCfg.Mode != "" {
		cfg.Mode = subCfg.Mode
	}

	if subCfg.CDPEndpoint != "" {
		cfg.CDPEndpoint = subCfg.CDPEndpoint
	}

	if subCfg.ChromeDebugPort != "" {
		cfg.ChromeDebugPort = subCfg.ChromeDebugPort
	}

	if subCfg.UserDataDir != "" {
		cfg.UserDataDir = subCfg.UserDataDir
	}

	if domains := parseCloneDomains(subCfg.CloneDomains); len(domains) > 0 {
		cfg.CloneDomains = domains
	}

	if subCfg.NoClone {
		cfg.NoClone = true
	}

	if subCfg.CloneAll {
		cfg.CloneAll = true
	}

	if subCfg.CompactSnapshot {
		cfg.CompactSnapshot = true
	}

	if subCfg.OutputDir != "" {
		cfg.OutputDir = subCfg.OutputDir
	}

	if subCfg.OmitImages {
		cfg.ImageResponses = types.ImageResponsesOmit
	}

	cfg.ServerVersion = Version

	runner := NewRunner(ctx, *cfg)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
		defer signal.Stop(c)

		for {
			select {
			case <-c:
				log.Info("Received signal, exiting...")
				cancel()
				return
			}
		}
	}()
	runner.Run()

	defer func() {
		err := runner.Close()
		if err != nil {
			log.Errorf("Server close error: %s", err)
		}
	}()
	return
}
