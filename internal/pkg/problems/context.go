package problems

import (
	"context"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/cache"
	"github.com/udovin/solve/internal/pkg/compilers"
	ccache "github.com/udovin/solve/internal/pkg/compilers/cache"
)

type compileContext struct {
	compilers *models.CompilerStore
	cache     *ccache.CompilerImageManager
	images    map[int64]cache.Resource[ccache.CompilerImage]
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
		c.images = map[int64]cache.Resource[ccache.CompilerImage]{}
	}
	if image, ok := c.images[compiler.ImageID]; ok {
		return image.Get().Compiler(config), nil
	}
	image, err := c.cache.LoadSync(ctx, compiler.ImageID)
	if err != nil {
		return nil, err
	}
	c.images[compiler.ImageID] = image
	return image.Get().Compiler(config), nil
}

func (c *compileContext) Release() {
	for _, image := range c.images {
		image.Release()
	}
	c.images = nil
}

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
