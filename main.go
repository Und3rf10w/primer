package main

import (
    "./constants"
    "flag"
    "fmt"
    "os"
    "time"
    "encoding/json"
    "strings"
)

type OutputFormat string

const (
    FormatText OutputFormat = "text"
    FormatJSON OutputFormat = "json"
    FormatCSV  OutputFormat = "csv"
)

type Options struct {
    ConfigPath    string
    OutputFormat  OutputFormat
    Verbose      bool
    BatchSize    int
    OutputFile   string
    QuickTest    bool
    CompareWith  string
}

func main() {
    opts := parseFlags()

    // Load configuration
    config, err := constants.LoadConfig(opts.ConfigPath)
    if err != nil {
        fmt.Printf("Error loading configuration: %v\n", err)
        os.Exit(1)
    }

    // Apply quick test modifications if requested
    if opts.QuickTest {
        config.NumCandidates = 10
        config.AvalancheTestCases = 100
        fmt.Println("Running in quick test mode with reduced parameters")
    }

    // Create generator
    generator := constants.NewGenerator(config)

    // Start timing
    start := time.Now()
    fmt.Println("Starting RC6 constant generation and analysis...")

    // Generate constants
    result, err := generator.Generate()
    if err != nil {
        fmt.Printf("Error generating constants: %v\n", err)
        os.Exit(1)
    }

    // Process and output results
    outputResults(result, opts)

    // Compare with existing constants if requested
    if opts.CompareWith != "" {
        compareWithExisting(result, opts.CompareWith)
    }
}

func parseFlags() Options {
    opts := Options{}

    flag.StringVar(&opts.ConfigPath, "config", "", "Path to configuration file")
    flag.Var((*outputFormatFlag)(&opts.OutputFormat), "format", "Output format (text, json, csv)")
    flag.BoolVar(&opts.Verbose, "verbose", false, "Enable verbose output")
    flag.IntVar(&opts.BatchSize, "batch", 100, "Batch size for processing")
    flag.StringVar(&opts.OutputFile, "output", "", "Output file path")
    flag.BoolVar(&opts.QuickTest, "quick", false, "Run quick test with reduced parameters")
    flag.StringVar(&opts.CompareWith, "compare", "", "Compare with existing constants file")

    flag.Parse()

    // Set defaults
    if opts.OutputFormat == "" {
        opts.OutputFormat = FormatText
    }

    return opts
}

func outputResults(result *constants.GenerationResult, opts Options) {
    switch opts.OutputFormat {
    case FormatJSON:
        outputJSON(result, opts)
    case FormatCSV:
        outputCSV(result, opts)
    default:
        outputText(result, opts)
    }
}

func outputText(result *constants.GenerationResult, opts Options) {
    fmt.Printf("\nGeneration completed in %v\n", result.Duration)
    fmt.Printf("\nSelected Constants:\n")
    fmt.Printf("P: 0x%X\n", result.SelectedP.Value)
    fmt.Printf("Q: 0x%X\n", result.SelectedQ.Value)

    if opts.Verbose {
        fmt.Printf("\nDetailed Analysis:\n")
        fmt.Printf("P Constant:\n")
        printConstantAnalysis(result.SelectedP)
        fmt.Printf("\nQ Constant:\n")
        printConstantAnalysis(result.SelectedQ)
    }

    fmt.Printf("\nStatistical Test Results:\n")
    printStatisticalSummary(result)

    if opts.OutputFile != "" {
        fmt.Printf("\nDetailed results saved to: %s\n", opts.OutputFile)
    }
}

func printConstantAnalysis(c constants.ConstantCandidate) {
    fmt.Printf("  Value: 0x%X\n", c.Value)
    fmt.Printf("  Bit Distribution: %.4f\n", c.BitDistribution)
    fmt.Printf("  Avalanche Score: %.4f\n", c.AvalancheScore)
    fmt.Printf("  Entropy Score: %.4f\n", c.EntropyScore)
    fmt.Printf("  Hamming Weight: %d\n", c.HammingWeight)
    
    if len(c.TestResults.StatisticalTests) > 0 {
        fmt.Printf("  Statistical Tests:\n")
        for _, test := range c.TestResults.StatisticalTests {
            fmt.Printf("    %s: %.4f (%v)\n", test.Name, test.Score, test.Passed)
            if test.Details != "" {
                fmt.Printf("      %s\n", test.Details)
            }
        }
    }
}

func printStatisticalSummary(result *constants.GenerationResult) {
    fmt.Printf("\nOverall Statistical Analysis:\n")
    fmt.Printf("Total Candidates Tested: %d\n", result.TotalCandidates)
    fmt.Printf("Generation Time: %v\n", result.Duration)
    
    // Calculate and display overall scores
    pScore := calculateOverallScore(result.SelectedP)
    qScore := calculateOverallScore(result.SelectedQ)
    
    fmt.Printf("P Constant Overall Score: %.4f\n", pScore)
    fmt.Printf("Q Constant Overall Score: %.4f\n", qScore)
}

func calculateOverallScore(c constants.ConstantCandidate) float64 {
    var total float64
    count := 0

    // Weight different scores
    total += c.AvalancheScore * 2.0
    count += 2
    
    total += c.BitDistribution
    count++
    
    total += c.EntropyScore / 2.0
    count++

    for _, test := range c.TestResults.StatisticalTests {
        total += test.Score
        count++
    }

    if count == 0 {
        return 0
    }
    return total / float64(count)
}

func outputJSON(result *constants.GenerationResult, opts Options) {
    output, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        fmt.Printf("Error generating JSON output: %v\n", err)
        return
    }

    if opts.OutputFile != "" {
        err = os.WriteFile(opts.OutputFile, output, 0644)
        if err != nil {
            fmt.Printf("Error writing to output file: %v\n", err)
            return
        }
    } else {
        fmt.Println(string(output))
    }
}

func outputCSV(result *constants.GenerationResult, opts Options) {
    var builder strings.Builder

    // Write header
    builder.WriteString("Constant,Value,BitDistribution,AvalancheScore,EntropyScore,HammingWeight\n")

    // Write P constant
    builder.WriteString(fmt.Sprintf("P,0x%X,%.4f,%.4f,%.4f,%d\n",
        result.SelectedP.Value,
        result.SelectedP.BitDistribution,
        result.SelectedP.AvalancheScore,
        result.SelectedP.EntropyScore,
        result.SelectedP.HammingWeight))

    // Write Q constant
    builder.WriteString(fmt.Sprintf("Q,0x%X,%.4f,%.4f,%.4f,%d\n",
        result.SelectedQ.Value,
        result.SelectedQ.BitDistribution,
        result.SelectedQ.AvalancheScore,
        result.SelectedQ.EntropyScore,
        result.SelectedQ.HammingWeight))

    output := builder.String()
    if opts.OutputFile != "" {
        err := os.WriteFile(opts.OutputFile, []byte(output), 0644)
        if err != nil {
            fmt.Printf("Error writing to output file: %v\n", err)
            return
        }
    } else {
        fmt.Print(output)
    }
}

func compareWithExisting(result *constants.GenerationResult, comparePath string) {
    fmt.Println("\nComparing with existing constants:")
    
    // Read existing constants file
    data, err := os.ReadFile(comparePath)
    if err != nil {
        fmt.Printf("Error reading comparison file: %v\n", err)
        return
    }

    var existing constants.GenerationResult
    if err := json.Unmarshal(data, &existing); err != nil {
        fmt.Printf("Error parsing comparison file: %v\n", err)
        return
    }

    // Compare and display results
    fmt.Printf("\nExisting Constants:\n")
    fmt.Printf("P: 0x%X\n", existing.SelectedP.Value)
    fmt.Printf("Q: 0x%X\n", existing.SelectedQ.Value)
    
    fmt.Printf("\nStatistical Comparison:\n")
    fmt.Printf("                    New         Existing\n")
    fmt.Printf("P Avalanche Score: %.4f vs %.4f\n",
        result.SelectedP.AvalancheScore,
        existing.SelectedP.AvalancheScore)
    fmt.Printf("Q Avalanche Score: %.4f vs %.4f\n",
        result.SelectedQ.AvalancheScore,
        existing.SelectedQ.AvalancheScore)
}

// Custom flag type for output format
type outputFormatFlag OutputFormat

func (f *outputFormatFlag) String() string {
    return string(*f)
}

func (f *outputFormatFlag) Set(value string) error {
    switch strings.ToLower(value) {
    case "text", "json", "csv":
        *f = outputFormatFlag(value)
        return nil
    default:
        return fmt.Errorf("invalid output format: %s", value)
    }
}