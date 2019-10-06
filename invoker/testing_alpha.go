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
	LastTest models.ReportDataTest
	TempDir  string
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
	problem, err := s.app.Problems.Get(c.ProblemID)
	if err != nil {
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
		c.Verdict = models.CompilationError
		return err
	}
	return bin.Wait()
}

func (s *Invoker) compileSolution(c *context) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		c.Verdict = models.CompilationError
		return err
	}
	bin := exec.Cmd{
		Path: dockerPath,
		Args: []string{
			"docker", "run", "--rm", "-t", "--stop-timeout", "10", "-v",
			fmt.Sprintf("%s:%s", c.TempDir, solutionHome),
			compileImage,
		},
	}
	if err := bin.Start(); err != nil {
		c.Verdict = models.CompilationError
		return err
	}
	if err := bin.Wait(); err != nil {
		c.Verdict = models.CompilationError
		return err
	}
	if _, err := os.Stat(path.Join(c.TempDir, executableFileName)); os.IsNotExist(err) {
		c.Verdict = models.CompilationError
		logs, err := ioutil.ReadFile(path.Join(c.TempDir, compilationLogFileName))
		if err != nil {
			return err
		}
		c.Data.CompileLogs.Stdout = string(logs)
		return fmt.Errorf("compilation error")
	}
	return nil
}

func (s *Invoker) runTests(c *context) error {
	files, err := ioutil.ReadDir(path.Join(c.TempDir, testsDir))
	if err != nil {
		return err
	}
	c.Verdict = models.Accepted
	for _, file := range files {
		if _, err := strconv.Atoi(file.Name()); err != nil {
			continue
		}
		input := path.Join(c.TempDir, testsDir, file.Name())
		answer := path.Join(c.TempDir, testsDir, fmt.Sprintf("%s.a", file.Name()))
		if err := s.runTest(c, input, answer); err != nil {
			c.Verdict = c.LastTest.Verdict
			return err
		}
	}
	return nil
}

func (s *Invoker) runTest(c *context, input, answer string) error {
	dir := path.Join(os.TempDir(), uuid.New().String())
	if err := os.Mkdir(dir, 0777); err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Println("Error:", err)
		}
	}()
	c.LastTest = models.ReportDataTest{
		Verdict: models.Accepted,
	}
	defer func() {
		if c.LastTest.Verdict != models.Accepted {
			c.Verdict = c.LastTest.Verdict
		}
		c.Data.Tests = append(c.Data.Tests, c.LastTest)
	}()
	if err := s.runSolution(c, dir, input); err != nil {
		return err
	}
	if err := s.runChecker(c, dir, input, answer); err != nil {
		return err
	}
	return nil
}

func (s *Invoker) runSolution(c *context, dir, input string) error {
	testInput := path.Join(dir, inputFileName)
	if err := copyFile(input, testInput); err != nil {
		c.LastTest.Verdict = models.RuntimeError
		log.Println("Error:", err)
		return err
	}
	binary := path.Join(c.TempDir, executableFileName)
	testBinary := path.Join(dir, executableFileName)
	if err := copyFile(binary, testBinary); err != nil {
		c.LastTest.Verdict = models.RuntimeError
		log.Println("Error:", err)
		return err
	}
	docker, err := exec.LookPath("docker")
	if err != nil {
		c.LastTest.Verdict = models.RuntimeError
		log.Println("Error:", err)
		return err
	}
	cmd := exec.Cmd{
		Path: docker,
		Args: []string{
			"docker", "run", "--rm", "-t", "--stop-timeout", "10", "-v",
			fmt.Sprintf("%s:%s", dir, solutionHome),
			executeImage,
		},
	}
	if err := cmd.Start(); err != nil {
		c.LastTest.Verdict = models.RuntimeError
		log.Println("Error:", err)
		return err
	}
	exited := make(chan error)
	go func() {
		exited <- cmd.Wait()
	}()
	select {
	case <-time.After(10 * time.Second):
		c.LastTest.Verdict = models.TimeLimitExceeded
		if err := cmd.Process.Kill(); err != nil {
			log.Println("Error:", err)
		}
		return fmt.Errorf("time limit exceeded")
	case err := <-exited:
		if err != nil {
			c.LastTest.Verdict = models.RuntimeError
			log.Println("Error:", err)
		}
		return err
	}
}

func (s *Invoker) runChecker(c *context, dir, input, answer string) error {
	testOutput := path.Join(dir, outputFileName)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	bin := exec.Cmd{
		Path: path.Join(c.TempDir, checkerName),
		Args: []string{
			checkerName, input, testOutput, answer,
		},
		Dir:    c.TempDir,
		Stdout: stdout,
		Stderr: stderr,
	}
	if err := bin.Start(); err != nil {
		c.LastTest.Verdict = models.WrongAnswer
		return err
	}
	if err := bin.Wait(); err != nil {
		c.LastTest.Verdict = models.WrongAnswer
		return err
	}
	c.LastTest.CheckLogs.Stdout = stdout.String()
	c.LastTest.CheckLogs.Stderr = stderr.String()
	if !bin.ProcessState.Success() {
		c.LastTest.Verdict = models.WrongAnswer
		return fmt.Errorf("wrong answer")
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
