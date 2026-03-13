// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package main

import (
	"flag"
	"fmt"
	"os"

	"ok/internal/appinfo"
	"ok/internal/startup"
)

const (
	colorGreen = "\033[1;38;2;16;185;129m"
	banner     = "\r\n" +
		colorGreen + " ██████╗ ██╗  ██╗\n" +
		colorGreen + "██╔═══██╗██║ ██╔╝\n" +
		colorGreen + "██║   ██║█████╔╝ \n" +
		colorGreen + "██║   ██║██╔═██╗ \n" +
		colorGreen + "╚██████╔╝██║  ██╗\n" +
		colorGreen + " ╚═════╝ ╚═╝  ╚═╝\n" +
		"\033[0m\r\n"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	fmt.Print(banner)
	fmt.Printf("%s ok - Personal AI Assistant v%s\n\n", appinfo.Logo, appinfo.Version)

	if *showVersion {
		fmt.Println(appinfo.FormatVersion())
		build, goVer := appinfo.FormatBuildInfo()
		if build != "" {
			fmt.Printf("Built: %s\n", build)
		}
		fmt.Printf("Go: %s\n", goVer)
		return
	}

	startup.EnsureOnboarded()

	if err := startup.GatewayCmd(*debug); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
