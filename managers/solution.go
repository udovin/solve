package managers

import (
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type SolutionManager struct {
	Solutions *models.SolutionStore
	Files     *FileManager
}

func NewSolutionManager(core *core.Core, files *FileManager) *SolutionManager {
	return &SolutionManager{
		Solutions: core.Solutions,
		Files:     files,
	}
}
