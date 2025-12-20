//go:build debug

package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var profileFile *os.File

func setupProfiling() {
	if *cpuprofile != "" {
		var err error
		profileFile, err = os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(profileFile)
	}
}

func stopProfiling() {
	if profileFile != nil {
		pprof.StopCPUProfile()
		profileFile.Close()
	}
}
