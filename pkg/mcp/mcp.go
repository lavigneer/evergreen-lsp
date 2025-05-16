package mcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/lavigneer/evergreen-lsp/pkg/project"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type Executor struct {
	workspace *project.Project
}

func New(workspace *project.Project) *Executor {
	return &Executor{workspace: workspace}
}

func (e *Executor) Register(server *mcp_golang.Server) error {
	var errs error
	err := e.RegisterTasks(server)
	errs = errors.Join(errs, err)
	err = e.RegisterFunctions(server)
	errs = errors.Join(errs, err)
	return errs
}

type FindTaskArgs struct {
	Name string `json:"name" jsonschema:"required,description=The name of the task to find"`
}

func (e *Executor) RegisterTasks(server *mcp_golang.Server) error {
	var errs error
	for _, t := range e.workspace.Data.Tasks {
		u := fmt.Sprintf("task://%s/%s", e.workspace.BasePath, t.Name)
		err := server.RegisterResource(u, t.Name, "", "text/plain", func() (*mcp_golang.ResourceResponse, error) {
			return mcp_golang.NewResourceResponse(mcp_golang.NewTextEmbeddedResource(u, t.Name, "text/plain")), nil
		})
		errs = errors.Join(errs, err)
	}
	server.RegisterTool("find_task_variants", "Finds all build variants that use a task in the evergreen configuration", func(arguments FindTaskArgs) (*mcp_golang.ToolResponse, error) {
		bvs := []string{}
		for _, bv := range e.workspace.Data.BuildVariants {
			for _, t := range bv.Tasks {
				if t.Name == arguments.Name {
					bvName := bv.DisplayName
					if bvName == "" {
						bvName = bv.Name
					}
					bvs = append(bvs, bvName)
				}
			}
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(strings.Join(bvs, ", "))), nil
	})
	return errs
}

func (e *Executor) RegisterFunctions(server *mcp_golang.Server) error {
	var errs error
	for name, f := range e.workspace.Data.Functions {
		u := fmt.Sprintf("function://%s/%s", e.workspace.BasePath, name)
		data, err := yaml.Marshal(f)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		err = server.RegisterResource(u, name, "", "text/plain", func() (*mcp_golang.ResourceResponse, error) {
			return mcp_golang.NewResourceResponse(mcp_golang.NewTextEmbeddedResource(u, string(data), "text/plain")), nil
		})
		errs = errors.Join(errs, err)
	}
	return errs
}
