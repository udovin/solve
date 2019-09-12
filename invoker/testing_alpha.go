package invoker

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/udovin/solve/models"
)

type context struct {
	*models.Solution
	*models.Compiler
	*models.Problem
	*models.Report
	TempDir string
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
	executableFileName       = "SOLUTION.EXE"
	compilationLogFileName   = "COMPLIE.LOG"
)

func (s *Invoker) processSolution(c *context) (err error) {
	if err = s.preparePackage(c); err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(c.TempDir); err != nil {
			log.Println("Error:", err)
		}
	}()
	defer func() {
		err = s.app.Reports.Update(c.Report)
	}()
	if err = s.preprocessSolution(c); err != nil {
		return
	}
	if err = s.compileSolution(c); err != nil {
		return
	}
	if err = s.runTests(c); err != nil {
		return
	}
	return
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
	c.TempDir = fmt.Sprintf("%s/%s", os.TempDir(), uuid.New().String())
	if err := os.Mkdir(c.TempDir, 0777); err != nil {
		return err
	}
	problemPath := fmt.Sprintf(
		"%s/%d.zip", s.app.Config.Invoker.ProblemsDir, problem.ID,
	)
	if err := unzip(problemPath, c.TempDir); err != nil {
		log.Println("Error:", err)
		return err
	}
	if err := ioutil.WriteFile(
		path.Join(c.TempDir, solutionName), []byte(c.SourceCode), 0777,
	); err != nil {
		return err
	}
	if err := os.Chmod(path.Join(c.TempDir, checkerName), 0777); err != nil {
		return err
	}
	return os.Chmod(path.Join(c.TempDir, preprocessName), 0777)
}

func (s *Invoker) preprocessSolution(c *context) error {
	bin := exec.Cmd{
		Dir:  c.TempDir,
		Path: path.Join(c.TempDir, preprocessName),
		Args: []string{
			preprocessName,
			solutionName,
			preprocessedSolutionName,
		},
	}
	err := bin.Start()
	if err != nil {
		return err
	}
	return bin.Wait()
}

func (s *Invoker) compileSolution(c *context) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	bin := exec.Cmd{
		Path: dockerPath,
		Args: []string{
			"docker", "run", "--rm", "-t", "-v",
			fmt.Sprintf("%s:%s", c.TempDir, solutionHome),
			compileImage,
		},
	}
	if err := bin.Start(); err != nil {
		return err
	}
	if err := bin.Wait(); err != nil {
		return err
	}
	if _, err := os.Stat(path.Join(c.TempDir, executableFileName)); os.IsNotExist(err) {
		logs, err := ioutil.ReadFile(path.Join(c.TempDir, compilationLogFileName))
		if err != nil {
			return err
		}
		c.Verdict = models.CompilationError
		c.Data.CompileLogs.Stdout = string(logs)
	}
	return nil
}

func (s *Invoker) runTests(c *context) error {
	tempDir := path.Join(os.TempDir(), uuid.New().String())
	if err := os.Mkdir(tempDir, 0777); err != nil {
		return err
	}
	files, err := ioutil.ReadDir(path.Join(c.TempDir, testsDir))
	if err != nil {
		return err
	}
	for _, file := range files {
		if _, err := strconv.Atoi(file.Name()); err != nil {
			continue
		}
		answerFile := path.Join(c.TempDir, testsDir, fmt.Sprintf("%s.a", file.Name()))
		inputFile := path.Join(c.TempDir, testsDir, file.Name())
		c.Data.Tests = append(c.Data.Tests, models.ReportDataTest{})
		test := len(c.Data.Tests) - 1
		c.Data.Tests[test].Verdict = models.Accepted
		if err := s.runTest(c, tempDir, inputFile); err != nil {
			c.Data.Tests[test].Verdict = c.Report.Verdict
			return err
		}
		if err := s.checkTest(c, tempDir, inputFile, answerFile); err != nil {
			c.Data.Tests[test].Verdict = c.Report.Verdict
			return err
		}
	}
	return nil
}

func copyFile(source, target string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Println("Error:", err)
		}
	}()
	w, err := os.Create(target)
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
	return nil
}

func (s *Invoker) runTest(c *context, tempDir, inputFile string) error {
	testInputFile := path.Join(tempDir, inputFileName)
	if err := copyFile(inputFile, testInputFile); err != nil {
		return err
	}
	binary := path.Join(c.TempDir, executableFileName)
	testBinary := path.Join(tempDir, executableFileName)
	if err := copyFile(binary, testBinary); err != nil {
		return err
	}
	// Run solution
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	bin := exec.Cmd{
		Path: dockerPath,
		Args: []string{
			"docker", "run", "--rm", "-t", "-v",
			fmt.Sprintf("%s:%s", tempDir, solutionHome),
			executeImage,
		},
	}
	if err := bin.Start(); err != nil {
		return err
	}
	exited := make(chan error)
	go func() {
		exited <- bin.Wait()
	}()
	select {
	case <-time.After(2 * time.Second):
		if err := bin.Process.Kill(); err != nil {
			log.Println("Error:", err)
		}
		c.Verdict = models.TimeLimitExceeded
		return nil
	case err := <-exited:
		return err
	}
}

func (s *Invoker) checkTest(c *context, tempDir, inputFile, answerFile string) error {
	solOutputFile := path.Join(tempDir, outputFileName)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	bin := exec.Cmd{
		Path: path.Join(c.TempDir, checkerName),
		Args: []string{
			checkerName,
			inputFile,
			solOutputFile,
			answerFile,
		},
		Dir:    c.TempDir,
		Stdout: stdout,
		Stderr: stderr,
	}
	if err := bin.Start(); err != nil {
		return err
	}
	defer func() {
		test := len(c.Data.Tests) - 1
		c.Data.Tests[test].CheckLogs.Stdout = stdout.String()
		c.Data.Tests[test].CheckLogs.Stderr = stderr.String()
	}()
	if err := bin.Wait(); err != nil {
		c.Verdict = models.WrongAnswer
		return err
	}
	if !bin.ProcessState.Success() {
		c.Verdict = models.WrongAnswer
		return errors.New("wrong answer")
	} else {
		c.Verdict = models.Accepted
		return nil
	}
}
