package invoker

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/udovin/solve/models"
)

type context struct {
	*models.Solution
	*models.Compiler
	*models.Problem
	*models.Report
	TemDir string
}

const (
	solutionName             = "solution.raw.asm"
	preprocessedSolutionName = "solution.asm"
	preprocessName           = "preprocess"
	checkerName              = "checker"
	solutionHome             = "/home/solution"
	compileImage             = "tasm/compile"
	executeImage             = "tasm/execute"
	testsDir                 = "tests"
	inputFileName            = "INPUT.TXT"
	outputFileName           = "OUTPUT.TXT"
)

func (s *Invoker) processSolution(c *context) error {
	if err := s.preparePackage(c); err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(c.TemDir); err != nil {
			log.Println("Error:", err)
		}
	}()
	if err := s.preprocessSolution(c); err != nil {
		c.Verdict = models.CompilationError
		return s.app.Reports.Update(c.Report)
	}
	if err := s.compileSolution(c); err != nil {
		c.Verdict = models.CompilationError
		return s.app.Reports.Update(c.Report)
	}
	if err := s.runTests(c); err != nil {
		return nil
	}
	c.Verdict = models.Accepted
	return s.app.Reports.Update(c.Report)
}

func (s *Invoker) preparePackage(c *context) error {
	if err := s.app.Problems.Manager.Sync(); err != nil {
		log.Println("Error:", err)
	}
	problem, ok := s.app.Problems.Get(c.ProblemID)
	if !ok {
		return fmt.Errorf("unknown problem with id = %d", c.ProblemID)
	}
	c.Problem = &problem
	c.TemDir = fmt.Sprintf("%s/%s", os.TempDir(), uuid.New().String())
	if err := os.Mkdir(c.TemDir, 0777); err != nil {
		return err
	}
	problemPath := fmt.Sprintf("%s/%d.zip", s.app.Config.Invoker.ProblemsDir, problem.ID)
	if err := unzip(problemPath, c.TemDir); err != nil {
		return err
	}
	if err := ioutil.WriteFile(
		path.Join(c.TemDir, solutionName), []byte(c.SourceCode), 0777,
	); err != nil {
		return err
	}
	return nil
}

func (s *Invoker) preprocessSolution(c *context) error {
	ps := exec.Cmd{
		Dir:  c.TemDir,
		Path: path.Join(c.TemDir, preprocessName),
		Args: []string{
			solutionName,
			preprocessedSolutionName,
		},
	}
	err := ps.Start()
	if err != nil {
		return err
	}
	return ps.Wait()
}

func (s *Invoker) compileSolution(c *context) error {
	cs := exec.Cmd{
		Path: "docker",
		Args: []string{
			"run",
			"-v",
			fmt.Sprintf("%q:%q", c.TemDir, solutionHome),
			compileImage,
		},
	}
	err := cs.Start()
	if err != nil {
		return err
	}
	return cs.Wait()
}

func (s *Invoker) runTests(c *context) error {
	files, err := ioutil.ReadDir(path.Join(c.TemDir, testsDir))
	if err != nil {
		return err
	}
	for _, file := range files {
		if _, err := strconv.Atoi(file.Name()); err != nil {
			continue
		}
		answerFile := fmt.Sprintf("%s.a", file.Name())
		if err := s.runTest(c, file.Name(), answerFile); err != nil {
			return err
		}
	}
	return nil
}

func (s *Invoker) runTest(c *context, inputFile, answerFile string) error {
	tempDir := path.Join(os.TempDir(), uuid.New().String())
	if err := os.Mkdir(tempDir, 0777); err != nil {
		return err
	}
	r, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Println("Error:", err)
		}
	}()
	tempInputFile := path.Join(tempDir, inputFileName)
	w, err := os.Create(tempInputFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Println("Error:", err)
		}
	}()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	// Run solution
	rt := exec.Cmd{
		Path: "docker",
		Args: []string{
			"run",
			"-v",
			fmt.Sprintf("%q:%q", tempDir, solutionHome),
			executeImage,
		},
	}
	if err := rt.Start(); err != nil {
		return err
	}
	waiter := time.After(2 * time.Second)
	go func() {
		<-waiter
	}()
	return rt.Wait()
}
