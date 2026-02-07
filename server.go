package main

import (
	"context"
	"github.com/charmbracelet/log"
	"github.com/aliwatters/rod-mcp/tools"
	"github.com/aliwatters/rod-mcp/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	ctx       *types.Context
	mcpServer *server.MCPServer
}

func NewServer(stdCtx context.Context, cfg types.Config) *Server {
	ctx := types.NewContext(stdCtx, cfg)
	mcpServer := server.NewMCPServer(cfg.ServerName, cfg.ServerVersion)
	ser := &Server{
		ctx:       ctx,
		mcpServer: mcpServer,
	}
	switch ctx.CurrentMode() {
	case types.Text:
		ser.registerTools(tools.TextTools, tools.TextToolHandlers)
	case types.Vision:
		log.Warn("Vision mode is not yet implemented, falling back to Text mode")
		ser.registerTools(tools.TextTools, tools.TextToolHandlers)
	}
	return ser

}

func (s *Server) registerTools(mcpTools []mcp.Tool, handlers map[string]tools.ToolHandler) *Server {
	for _, mt := range mcpTools {
		if handlerFunc, ok := handlers[mt.Name]; ok {
			log.Debugf("register tool: %s", mt.Name)
			s.mcpServer.AddTool(mt, handlerFunc(s.ctx))
		}

	}
	return s

}

func (s *Server) Start() error {
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return err
	}
	return nil
}

func (s *Server) Close() error {
	return s.ctx.Close()
}
