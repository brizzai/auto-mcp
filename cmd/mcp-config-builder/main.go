package main

import (
	"os"
	"runtime/debug"

	"github.com/pterm/pterm"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var (
	swaggerFile     string
	adjustmentsFile string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "mcp-config-builder",
	Short: "A tool to build MCP config from Swagger",
	Long: `MCP Config Builder is a CLI tool that helps you build MCP config from Swagger/OpenAPI definitions.
It allows you to filter out routes and adjust descriptions to optimize your API.`,
	Run: runTUI,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	// Place version check in PreRun to ensure flags are parsed first
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		versionFlag, _ := cmd.Flags().GetBool("version")
		if versionFlag {
			pterm.Info.Println(config.GetVersionInfo())
			os.Exit(0)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&swaggerFile, "swagger-file", "", "Path to the Swagger/OpenAPI file")
	rootCmd.PersistentFlags().StringVar(&adjustmentsFile, "adjustments-file", "", "Path to the MCP adjustments file")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Show version information")
}

// runTUI is the main function that runs the TUI
func runTUI(cmd *cobra.Command, args []string) {
	defer func() {
		if r := recover(); r != nil {
			pterm.Error.Printf("\nCaught panic: %v\n", r)
			pterm.Error.Printf("%s\n", debug.Stack())
			os.Exit(2)
		}
	}()
	// Create a new parser
	adjuster := parser.NewAdjuster()
	swaggerParser := parser.NewSwaggerParser(adjuster)

	if swaggerFile == "" {
		pterm.Error.Println("Swagger file is required, you must supply it with --swagger-file")
		os.Exit(1)
	}

	// Parse the swagger file
	err := swaggerParser.Init(swaggerFile, "") // no adjustments file for builder in edit mode
	if err != nil {
		pterm.Error.Printf("Error parsing swagger file: %v\n", err)
		os.Exit(1)
	}

	// Get the route tools
	routeTools := swaggerParser.GetRouteTools()
	err = adjuster.Load(adjustmentsFile)
	if err != nil {
		pterm.Error.Printf("Error loading adjustments file: %v\n", err)
		os.Exit(1)
	}

	// Create and run the TUI with the new AppModel
	p := tea.NewProgram(tui.NewAppModel(routeTools, adjuster), tea.WithAltScreen())

	// Run the program
	m, err := p.Run()
	if err != nil {
		pterm.Error.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	// Get the final model
	finalModel := m.(tui.AppModel)

	// Only display summary if the TUI completed successfully (user reached export page)
	if finalModel.IsFinished() {
		validRoutes := finalModel.GetRoutesUpdates()
		filteredRoutesCount := 0
		for _, route := range validRoutes {
			if !route.IsRemoved {
				filteredRoutesCount++
			}
		}
		pterm.Info.Printfln("Processing complete. Kept %s routes out of %s.",
			pterm.LightGreen(filteredRoutesCount),
			pterm.White(len(routeTools)))
	}
}
