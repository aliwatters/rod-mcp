package main

import (
	"fmt"
	"github.com/aliwatters/rod-mcp/banner"
	"github.com/aliwatters/rod-mcp/types"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"os"
)

type SubCfg struct {
	Headless        bool
	ConfigPath      string
	Mode            types.Mode
	CDPEndpoint     string
	ChromeDebugPort string
	CompactSnapshot bool
	OutputDir       string
	OmitImages      bool
}

func RunCmd() (*SubCfg, error) {
	subConfig := SubCfg{}
	cmd := &cli.App{
		Name:        "Rod MCP Server",
		Description: "Model Context Protocol Server of Rod",
		Usage:       "rod-mcp is a rod mcp server",
		Version:     banner.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "use to set Rod MCP Server's config file path, file name is `rod-mcp.yaml`",
				Destination: &subConfig.ConfigPath,
			}, &cli.StringFlag{
				Name:        "cdp-endpoint",
				Aliases:     []string{"cdp"},
				Usage:       "use to control running browser by cdp",
				Destination: &subConfig.CDPEndpoint,
			}, &cli.StringFlag{
				Name:        "chrome-debug-port",
				Usage:       "launch Chrome with --remote-debugging-port on the given port (e.g. 9222)",
				Destination: &subConfig.ChromeDebugPort,
			},
			&cli.BoolFlag{
				Name:        "headless",
				Aliases:     []string{"hl"},
				Value:       false,
				Usage:       "use to enable headless,if false browser will shown window",
				Destination: &subConfig.Headless,
			},
			&cli.BoolFlag{
				Name:    "no-banner",
				Aliases: []string{"nb"},
				Usage:   "use to disable show banner",
			},
			&cli.BoolFlag{
				Name:    "vision",
				Aliases: []string{"vs"},
				Usage:   "use to support vision LLM will load  vision tools",
			},
			&cli.BoolFlag{
				Name:    "compact-snapshot",
				Aliases: []string{"cs"},
				Usage:   "enable compact snapshot mode to reduce token usage by filtering non-interactive elements",
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Usage:       "directory for saving screenshots and PDFs (default: OS temp dir)",
				Destination: &subConfig.OutputDir,
			},
			&cli.BoolFlag{
				Name:    "omit-images",
				Usage:   "omit inline base64 image data from screenshot results (saves tokens)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.Bool("no-banner") {
				fmt.Println(banner.ShowBanner())
			}

			return nil
		},
		Action: func(c *cli.Context) error {
			if c.Bool("headless") {
				subConfig.Headless = true
			}

			if c.Bool("vision") {
				subConfig.Mode = types.Vision
			}
			if c.Bool("compact-snapshot") {
				subConfig.CompactSnapshot = true
			}
			if c.Bool("omit-images") {
				subConfig.OmitImages = true
			}
			return nil
		},
	}
	err := cmd.Run(os.Args)
	if err != nil {
		return nil, errors.Wrapf(err, "run cmd error")
	}
	return &subConfig, nil
}
