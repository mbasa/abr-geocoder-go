// Package cli provides the command-line interface for abr-geocoder.
// Ported from TypeScript: src/interface/cli/cli.ts
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/mbasa/abr-geocoder-go/internal/config"
	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	"github.com/mbasa/abr-geocoder-go/internal/interface/format"
	"github.com/mbasa/abr-geocoder-go/internal/interface/server"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/download"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode"
)

// defaultDataDir returns the default data directory
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", config.CacheDirName)
	}
	return filepath.Join(home, config.CacheDirName)
}

// NewRootCommand creates the root cobra command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "abrg",
		Short:   "ABR Geocoder - Japanese address geocoding tool",
		Long:    `abrg is a command-line tool for geocoding Japanese addresses using the Address Base Registry (ABR).`,
		Version: config.AppVersion,
	}

	rootCmd.AddCommand(newGeocodeCommand())
	rootCmd.AddCommand(newDownloadCommand())
	rootCmd.AddCommand(newServeCommand())
	rootCmd.AddCommand(newUpdateCheckCommand())

	return rootCmd
}

// newGeocodeCommand creates the geocode subcommand
func newGeocodeCommand() *cobra.Command {
	var (
		abrgDir  string
		target   string
		outFormat string
		fuzzy    string
		debug    bool
		silent   bool
		output   string
	)

	cmd := &cobra.Command{
		Use:   "geocode [input-file] [output-file]",
		Short: "Geocode Japanese addresses",
		Long: `Geocode Japanese addresses from a file or stdin.

Input can be a file path or '-' for stdin.
Output can be a file path or '-' for stdout (default).

Example:
  abrg geocode input.txt
  abrg geocode input.txt output.json --format json
  echo "東京都千代田区1-1" | abrg geocode -`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abrgDir == "" {
				abrgDir = defaultDataDir()
			}

			// Parse search target
			searchTarget := types.SearchTarget(strings.ToLower(target))
			if !searchTarget.IsValid() {
				return fmt.Errorf("invalid target: %s (must be all, residential, or parcel)", target)
			}

			// Parse output format
			outputFormat := types.OutputFormat(strings.ToLower(outFormat))
			if !outputFormat.IsValid() {
				return fmt.Errorf("invalid format: %s", outFormat)
			}

			// Determine input source
			var inputFile string
			if len(args) > 0 {
				inputFile = args[0]
			}

			// Determine output destination
			var outputFile string
			if len(args) > 1 {
				outputFile = args[1]
			}
			if output != "" {
				outputFile = output
			}

			return runGeocode(abrgDir, inputFile, outputFile, searchTarget, outputFormat, fuzzy, debug, silent)
		},
	}

	cmd.Flags().StringVar(&abrgDir, "abrgDir", "", "Data directory (default: ~/.abr-geocoder)")
	cmd.Flags().StringVar(&target, "target", "all", "Search target: all, residential, parcel")
	cmd.Flags().StringVar(&outFormat, "format", "json", "Output format: json, csv, geojson, ndjson, ndgeojson, simplified")
	cmd.Flags().StringVar(&fuzzy, "fuzzy", config.DefaultFuzzyChar, "Fuzzy matching character")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")
	cmd.Flags().BoolVar(&silent, "silent", false, "Suppress progress output")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	return cmd
}

// runGeocode executes the geocoding process
func runGeocode(dataDir, inputFile, outputFile string, target types.SearchTarget, outputFormat types.OutputFormat, fuzzyChar string, debug, silent bool) error {
	// Initialize geocoder
	g, err := geocode.New(geocode.GeocoderOptions{
		DataDir:      dataDir,
		FuzzyChar:    fuzzyChar,
		SearchTarget: target,
		Debug:        debug,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize geocoder: %w", err)
	}
	defer g.Close()

	// Set up input reader
	var scanner *bufio.Scanner
	if inputFile == "" || inputFile == "-" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(inputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	}

	// Set up output writer
	var out *os.File
	if outputFile == "" || outputFile == "-" {
		out = os.Stdout
	} else {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	// Create formatter
	formatter, err := format.NewFormatter(outputFormat, debug)
	if err != nil {
		return err
	}

	// Write header
	if err := formatter.WriteHeader(out); err != nil {
		return err
	}

	// Process each line
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		result, err := g.Geocode(line)
		if err != nil {
			if !silent {
				fmt.Fprintf(os.Stderr, "Warning: failed to geocode %q: %v\n", line, err)
			}
			continue
		}

		if err := formatter.WriteResult(out, result); err != nil {
			return err
		}

		lineNum++
		if !silent && lineNum%100 == 0 {
			fmt.Fprintf(os.Stderr, "\rProcessed %d addresses...", lineNum)
		}
	}

	if !silent && lineNum > 0 {
		fmt.Fprintf(os.Stderr, "\rProcessed %d addresses\n", lineNum)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	// Write footer
	return formatter.WriteFooter(out)
}

// newDownloadCommand creates the download subcommand
func newDownloadCommand() *cobra.Command {
	var (
		abrgDir string
		lgCodes []string
		threads int
		keep    bool
		debug   bool
		silent  bool
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download ABR dataset",
		Long: `Download the Address Base Registry dataset from the Digital Agency.

You must specify at least one local government code (lgCode) using the --lgCode flag.
LG codes are 6-digit numbers identifying municipalities in Japan.

Example:
  abrg download --lgCode 131016
  abrg download --lgCode 131016 --lgCode 131024`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if abrgDir == "" {
				abrgDir = defaultDataDir()
			}

			if len(lgCodes) == 0 {
				return fmt.Errorf("at least one --lgCode is required")
			}

			opts := download.DownloadOptions{
				DataDir: abrgDir,
				LGCodes: lgCodes,
				Threads: threads,
				KeepData: keep,
			}

			if !silent {
				opts.ProgressCallback = func(downloaded, total int64, current string) {
					fmt.Fprintf(os.Stderr, "\rDownloading: %s (%d/%d)", current, downloaded, total)
				}
			}

			d, err := download.NewDownloader(opts)
			if err != nil {
				return fmt.Errorf("failed to initialize downloader: %w", err)
			}
			defer d.Close()

			if err := d.Download(); err != nil {
				return fmt.Errorf("download failed: %w", err)
			}

			if !silent {
				fmt.Fprintln(os.Stderr, "\nDownload completed successfully")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&abrgDir, "abrgDir", "", "Data directory (default: ~/.abr-geocoder)")
	cmd.Flags().StringArrayVar(&lgCodes, "lgCode", nil, "Local government code (can be specified multiple times)")
	cmd.Flags().IntVar(&threads, "threads", 5, "Number of concurrent downloads")
	cmd.Flags().BoolVar(&keep, "keep", false, "Keep existing data (don't overwrite)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")
	cmd.Flags().BoolVar(&silent, "silent", false, "Suppress progress output")

	return cmd
}

// newServeCommand creates the serve subcommand
func newServeCommand() *cobra.Command {
	var (
		abrgDir string
		port    int
		fuzzy   string
		debug   bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the geocoding API server",
		Long: `Start a REST API server for geocoding addresses.

The server provides a GET /geocode endpoint that accepts:
  - address: the address to geocode (required)
  - format: output format (json, csv, geojson, etc.)
  - debug: enable debug output

Example:
  abrg serve
  abrg serve --port 8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if abrgDir == "" {
				abrgDir = defaultDataDir()
			}

			srv, err := server.New(server.ServerOptions{
				Port:      port,
				DataDir:   abrgDir,
				FuzzyChar: fuzzy,
				Debug:     debug,
			})
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}

			// Handle shutdown gracefully
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-quit
				fmt.Fprintln(os.Stderr, "\nShutting down server...")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				srv.Shutdown(ctx)
			}()

			fmt.Fprintf(os.Stderr, "Server starting on port %d\n", port)
			return srv.Start()
		},
	}

	cmd.Flags().StringVar(&abrgDir, "abrgDir", "", "Data directory (default: ~/.abr-geocoder)")
	cmd.Flags().IntVar(&port, "port", config.CLIServerPort, "Port to listen on")
	cmd.Flags().StringVar(&fuzzy, "fuzzy", config.DefaultFuzzyChar, "Fuzzy matching character")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")

	return cmd
}

// newUpdateCheckCommand creates the update-check subcommand
func newUpdateCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-check",
		Short: "Check for dataset updates",
		Long:  `Check if newer versions of the ABR dataset are available.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("ABR Geocoder version: %s\n", config.AppVersion)
			fmt.Println("Check https://github.com/mbasa/abr-geocoder-go for updates.")
			return nil
		},
	}
	return cmd
}
