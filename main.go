package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"gopkg.in/ini.v1"
)

const (
	ExitCodeOk             = 0
	ExitCodeParseFlagError = 1
	ExitCodeError          = 2
	Name                   = "awsp"
	Version                = "0.1.0"
)

const usage = `awsp is a tool to switch aws profile.
Usage: 
    %s [arguments]
Args:
    -h      Print Help message
    -v      Print the version of this tool
`

type cli struct {
	outStream, errStream io.Writer
}

func homedir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}

	return os.Getenv("HOME")
}

func configFileName() string {
	return filepath.Join(homedir(), ".aws", "config")
}

func loadConfig() (*ini.File, error) {
	cfg, err := ini.Load(configFileName())
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *cli) Run(args []string) int {
	var help, version bool

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(c.errStream)
	flags.Usage = func() {
		fmt.Fprintf(c.errStream, usage, Name)
	}
	flags.BoolVar(&help, "h", false, "display help message")
	flags.BoolVar(&version, "v", false, "display the version")

	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeParseFlagError
	}

	if help {
		flags.Usage()
		return ExitCodeOk
	}

	if version {
		fmt.Fprintf(c.errStream, "%s v%s\n", Name, Version)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(c.errStream, "aws config load error: %v\n", err)
		return ExitCodeError
	}
	sections := cfg.SectionStrings()
	for i, s := range sections {
		if strings.HasPrefix(s, "profile ") {
			splitprofile := strings.Split(s, " ")
			sections[i] = splitprofile[1]
		}
	}
	idx, err := fuzzyfinder.FindMulti(
		sections,
		func(i int) string {
			return sections[i]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return sections[i]
		}))
	if err != nil {
		fmt.Fprintf(c.errStream, "fuzzyfinder error: %v", err)
		return ExitCodeError
	}
	if err := os.Setenv("AWS_PROFILE", sections[idx[0]]); err != nil {
		fmt.Fprintf(c.errStream, "set enviroment variable error: %v", err)
		return ExitCodeError
	}
	fmt.Fprintf(c.outStream, "current profile %s\n", sections[idx[0]])
	fmt.Fprintf(c.outStream, "AWS_PROFILE: %s\n", os.Getenv("AWS_PROFILE"))

	return ExitCodeOk
}

func main() {
	c := &cli{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(c.Run(os.Args))
}
