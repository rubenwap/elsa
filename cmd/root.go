package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/elsaland/elsa/core/options"
	"github.com/elsaland/elsa/module"
	"github.com/elsaland/elsa/util"
	"github.com/fatih/color"

	"github.com/elsaland/elsa/packager"
	"github.com/spf13/cobra"
)

// Elsa functions expected to be passed into cmd
type Elsa struct {
	Run    func(opt options.Options)
	Dev    func(og string, opt options.Options)
	Bundle func(file string, minify bool, config *module.Config) string
}

// Execute start the CLI
func Execute(elsa Elsa) {
	// TODO: need to come to a concrete conclusion
	// Load mod.toml (if exists)
	config, err := module.ConfigLoad()
	util.Check(err)

	color.NoColor = config.Options.NoColor

	var fsFlag bool
	var netFlag bool
	var minifyFlag bool
	var envFlag bool

	// Rool command
	var rootCmd = &cobra.Command{
		Use:   "elsa [file]",
		Short: "Elsa is a simple JavaScript and TypeScript runtime written in Go",
	}

	// Run subcommand
	var runCmd = &cobra.Command{
		Use:   "run [file]",
		Short: "Run a JavaScript and TypeScript source file",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) >= 0 {
				bundle := elsa.Bundle(args[0], true, config)
				env := options.Environment{
					NoColor:  config.Options.NoColor,
					Args:     args[1:],
					RunTests: false,
				}
				opt := options.Options{
					SourceFile: args[0],
					Source:     bundle,
					Perms:      &options.Perms{fsFlag, netFlag, envFlag},
					Env:        env,
				}
				elsa.Run(opt)
			}
		},
	}

	// --fs,--net, --env flags
	runCmd.Flags().BoolVar(&fsFlag, "fs", false, "Allow file system access")
	runCmd.Flags().BoolVar(&netFlag, "net", false, "Allow net access")
	runCmd.Flags().BoolVar(&envFlag, "env", false, "Allow Environment Variables access")

	// dev subcommand to run script in development mode
	var devCmd = &cobra.Command{
		Use:   "dev [file]",
		Short: "Run a script in development mode.",
		Long:  `Run a script in development mode. It enables type-checking using the inbuilt TypeScript compiler.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) >= 0 {
				bundle := elsa.Bundle(args[0], true, config)
				env := options.Environment{
					NoColor:  config.Options.NoColor,
					Args:     args[1:],
					RunTests: false,
				}
				opt := options.Options{
					SourceFile: args[0],
					Source:     bundle,
					Perms:      &options.Perms{fsFlag, netFlag, envFlag},
					Env:        env,
				}
				og, _ := ioutil.ReadFile(args[0])
				elsa.Dev(string(og), opt)
			}
		},
	}

	// bundle subcommand to bundle a source file
	var bundleCmd = &cobra.Command{
		Use:   "bundle [file]",
		Short: "Bundle your script to a single JavaScript file",
		Long:  `Bundle your script to a single JavaScript file. It utilizes esbuild for super fast bundling.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) >= 0 {
				out := elsa.Bundle(args[0], minifyFlag, config)
				fmt.Println(out)
			}
		},
	}

	// --minify flag for bundling
	bundleCmd.Flags().BoolVarP(&minifyFlag, "minify", "m", false, "Minify the output bundle")

	// pkg subcommand for trigger the packager
	var pkgCmd = &cobra.Command{
		Use:   "pkg [file]",
		Short: "Package your script to a standalone executable.",
		Long:  `Package your script to a standalone executable.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) >= 0 {
				packager.PkgSource(args[0])
			}
		},
	}

	// test subcommand to run test files
	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "Run tests for your Elsa scripts.",
		Long:  `Run tests for your Elsa scripts. All files matching *_test.js are run.`,
		Run: func(cmd *cobra.Command, args []string) {
			env := options.Environment{
				NoColor:  config.Options.NoColor,
				Args:     args,
				RunTests: true,
			}
			opt := options.Options{
				Perms: &options.Perms{fsFlag, netFlag, envFlag},
				Env:   env,
			}
			tests := CollectTests()
			for _, test := range tests {
				opt.SourceFile = test
				bundle := elsa.Bundle(test, true, config)
				opt.Source = bundle
				elsa.Run(opt)
			}
		},
	}

	// --net, --fs, --env perms
	testCmd.Flags().BoolVar(&fsFlag, "fs", false, "Allow file system access")
	testCmd.Flags().BoolVar(&netFlag, "net", false, "Allow net access")
	testCmd.Flags().BoolVar(&envFlag, "env", false, "Allow Environment Variables access")
	// Add subcommands to root command
	rootCmd.AddCommand(bundleCmd, runCmd, pkgCmd, devCmd, testCmd)

	// Execute! :)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func CollectTests() []string {
	var testFiles []string
	e := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err == nil {
			matchedJS, err := filepath.Match("*_test.js", info.Name())
			matchedTS, err := filepath.Match("*_test.ts", info.Name())
			if err != nil {
				return nil
			}
			if matchedJS || matchedTS {
				testFiles = append(testFiles, path)
			}
		}
		return nil
	})
	if e != nil {
		log.Fatal(e)
	}
	return testFiles
}
