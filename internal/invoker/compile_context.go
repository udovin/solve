package invoker

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/cache"
	"github.com/udovin/solve/internal/pkg/compilers"
	compilerCache "github.com/udovin/solve/internal/pkg/compilers/cache"
	"github.com/udovin/solve/internal/pkg/logs"
	"github.com/udovin/solve/internal/pkg/problems"
)

type CompileContext interface {
	problems.CompileContext
	GetCompilerByID(context.Context, int64) (compilers.Compiler, error)
	Release()
}

type compileContext struct {
	compilers *models.CompilerStore
	cache     *compilerCache.CompilerImageManager
	images    map[int64]cache.Resource[compilerCache.CompilerImage]
	logger    *logs.Logger
}

func (c *compileContext) Logger() *logs.Logger {
	return c.logger
}

func (c *compileContext) GetCompilerByID(ctx context.Context, id int64) (compilers.Compiler, error) {
	compiler, err := c.compilers.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.getCompiler(ctx, compiler)
}

func (c *compileContext) GetCompiler(ctx context.Context, name string) (compilers.Compiler, error) {
	compiler, err := c.compilers.GetByName(name)
	if err != nil {
		return nil, err
	}
	return c.getCompiler(ctx, compiler)
}

func (c *compileContext) getCompiler(ctx context.Context, compiler models.Compiler) (compilers.Compiler, error) {
	config, err := compiler.GetConfig()
	if err != nil {
		return nil, err
	}
	if c.images == nil {
		c.images = map[int64]cache.Resource[compilerCache.CompilerImage]{}
	}
	if image, ok := c.images[compiler.ImageID]; ok {
		return image.Get().Compiler(compiler.Name, config), nil
	}
	image, err := c.cache.LoadSync(ctx, compiler.ImageID)
	if err != nil {
		return nil, err
	}
	c.images[compiler.ImageID] = image
	return image.Get().Compiler(compiler.Name, config), nil
}

func (c *compileContext) Release() {
	for _, image := range c.images {
		image.Release()
	}
	c.images = nil
}

var _ CompileContext = (*compileContext)(nil)

// pinnedCompileContext represents compile context with pinned compiler names.
type pinnedCompileContext struct {
	*compileContext
	pinnedCompilers map[string]int64
}

func (c *pinnedCompileContext) GetCompiler(ctx context.Context, name string) (compilers.Compiler, error) {
	if id, ok := c.pinnedCompilers[name]; ok {
		compiler, err := c.compilers.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return c.getCompiler(ctx, compiler)
	}
	compiler, err := c.compilers.GetByName(name)
	if err != nil {
		return nil, err
	}
	return c.getCompiler(ctx, compiler)
}

var _ CompileContext = (*pinnedCompileContext)(nil)

type polygonCompileContext struct {
	ctx      CompileContext
	settings *models.SettingStore
}

func (c *polygonCompileContext) GetCompilerByID(ctx context.Context, id int64) (compilers.Compiler, error) {
	return c.ctx.GetCompilerByID(ctx, id)
}

func (c *polygonCompileContext) GetCompiler(ctx context.Context, polygonType string) (compilers.Compiler, error) {
	name, err := c.getCompilerName("polygon." + polygonType)
	if err != nil {
		return nil, err
	}
	return c.ctx.GetCompiler(ctx, name)
}

func (c *polygonCompileContext) getCompilerName(name string) (string, error) {
	setting, err := c.settings.GetByKey("invoker.compilers." + name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("cannot get compiler %q", name)
		}
		return "", err
	}
	return setting.Value, nil
}

func (c *polygonCompileContext) Logger() *logs.Logger {
	return c.ctx.Logger()
}

func (c *polygonCompileContext) Release() {
	c.ctx.Release()
}

func (c *polygonCompileContext) GetCompilerName(name string) (string, error) {
	panic("not implemented")
}
