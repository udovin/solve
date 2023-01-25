package managers

import (
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type SolutionManager struct {
	solutions *models.SolutionStore
	files     *FileManager
}

func NewSolutionManager(core *core.Core, files *FileManager) *SolutionManager {
	return &SolutionManager{
		solutions: core.Solutions,
		files:     files,
	}
}
