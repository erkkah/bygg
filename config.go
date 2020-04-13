package main

import "flag"

type config struct {
	verbose     bool
	veryVerbose bool
	dryRun      bool
	baseDir     string
	byggFil     string
	target      string
}

func parseConfig(args []string) (cfg config, err error) {
	var fs flag.FlagSet

	fs.StringVar(&cfg.byggFil, "f", "byggfil", "Bygg file")
	fs.BoolVar(&cfg.dryRun, "n", false, "Performs a dry run")
	fs.BoolVar(&cfg.verbose, "v", false, "Verbose")
	fs.BoolVar(&cfg.veryVerbose, "vv", false, "Very verbose")
	fs.StringVar(&cfg.baseDir, "C", ".", "Base dir")
	err = fs.Parse(args)

	if cfg.veryVerbose {
		cfg.verbose = true
	}

	targets := fs.Args()
	if len(targets) > 0 {
		cfg.target = targets[0]
	} else {
		cfg.target = "all"
	}

	return
}
