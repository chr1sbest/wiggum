package eval

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// Compare compares evaluation results between ralph and oneshot approaches
// for a given suite and model, printing a formatted comparison table
func Compare(suite, model string) error {
	// Find latest result files for both approaches
	ralphFile, err := FindLatestResult(suite, "ralph", model)
	if err != nil {
		return fmt.Errorf("failed to find ralph result: %w", err)
	}

	oneshotFile, err := FindLatestResult(suite, "oneshot", model)
	if err != nil {
		return fmt.Errorf("failed to find oneshot result: %w", err)
	}

	// Load results
	ralph, err := LoadFromFile(ralphFile)
	if err != nil {
		return fmt.Errorf("failed to load ralph result: %w", err)
	}

	oneshot, err := LoadFromFile(oneshotFile)
	if err != nil {
		return fmt.Errorf("failed to load oneshot result: %w", err)
	}

	// Print comparison
	printComparison(ralph, oneshot, ralphFile, oneshotFile)
	return nil
}

// printComparison prints a formatted comparison table between two eval results
func printComparison(ralph, oneshot *EvalResult, ralphFile, oneshotFile string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    EVAL COMPARISON: Ralph vs One-Shot                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Ralph:   %s\n", filepath.Base(ralphFile))
	fmt.Printf("Oneshot: %s\n", filepath.Base(oneshotFile))
	fmt.Println()

	// Calculate winners for each metric
	durationWinner := calcWinner(ralph.DurationSeconds, oneshot.DurationSeconds, false)
	tokensWinner := calcWinner(ralph.TotalTokens, oneshot.TotalTokens, false)
	costWinner := calcWinnerFloat(ralph.CostUSD, oneshot.CostUSD, false)
	testsWinner := calcWinner(ralph.SharedTestsPassed, oneshot.SharedTestsPassed, true)

	// Print comparison table
	fmt.Println("┌──────────────────────┬───────────────────┬───────────────────┬─────────────────────┐")
	fmt.Printf("│ %-20s │ %-17s │ %-17s │ %-19s │\n", "Metric", "Ralph", "Oneshot", "Winner")
	fmt.Println("├──────────────────────┼───────────────────┼───────────────────┼─────────────────────┤")
	fmt.Printf("│ %-20s │ %15ds │ %15ds │ %-19s │\n", "Duration", ralph.DurationSeconds, oneshot.DurationSeconds, durationWinner)
	fmt.Printf("│ %-20s │ %17d │ %17d │ %-19s │\n", "Total Tokens", ralph.TotalTokens, oneshot.TotalTokens, tokensWinner)
	fmt.Printf("│ %-20s │ $%16.2f │ $%16.2f │ %-19s │\n", "Cost", ralph.CostUSD, oneshot.CostUSD, costWinner)
	fmt.Printf("│ %-20s │ %13d/%d │ %13d/%d │ %-19s │\n", "Shared Tests", ralph.SharedTestsPassed, ralph.SharedTestsTotal, oneshot.SharedTestsPassed, oneshot.SharedTestsTotal, testsWinner)
	fmt.Println("└──────────────────────┴───────────────────┴───────────────────┴─────────────────────┘")
	fmt.Println()

	// Calculate overall winner
	ralphWins := 0
	oneshotWins := 0
	ties := 0

	winners := []string{durationWinner, tokensWinner, costWinner, testsWinner}
	for _, winner := range winners {
		if strings.HasPrefix(winner, "Ralph") {
			ralphWins++
		} else if strings.HasPrefix(winner, "Oneshot") {
			oneshotWins++
		} else if winner == "Tie" {
			ties++
		}
	}

	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════")
	if ralphWins > oneshotWins {
		fmt.Printf("🏆 OVERALL WINNER: Ralph (%d metrics vs %d)\n", ralphWins, oneshotWins)
	} else if oneshotWins > ralphWins {
		fmt.Printf("🏆 OVERALL WINNER: Oneshot (%d metrics vs %d)\n", oneshotWins, ralphWins)
	} else {
		fmt.Printf("🤝 TIE: Both approaches won %d metrics each\n", ralphWins)
	}
	if ties > 0 {
		fmt.Printf("   (%d metric(s) tied)\n", ties)
	}
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println()
}

// calcWinner calculates the winner for integer metrics
// higherIsBetter indicates whether higher values are better (e.g., tests passed)
// or lower values are better (e.g., duration, tokens, cost)
func calcWinner(ralphVal, oneshotVal int, higherIsBetter bool) string {
	if oneshotVal == 0 {
		return "N/A"
	}

	// Calculate percentage difference
	diff := float64(ralphVal-oneshotVal) / float64(oneshotVal) * 100
	absDiff := math.Abs(diff)

	// Round to 2 decimal places
	absDiff = math.Round(absDiff*100) / 100

	if absDiff < 0.01 {
		return "Tie"
	}

	// Determine winner based on whether higher or lower is better
	if higherIsBetter {
		// For tests, higher is better
		if diff > 0 {
			return fmt.Sprintf("Ralph +%.2f%%", absDiff)
		}
		return fmt.Sprintf("Oneshot +%.2f%%", absDiff)
	}

	// For cost/time/tokens, lower is better
	if diff < 0 {
		return fmt.Sprintf("Ralph -%.2f%%", absDiff)
	}
	return fmt.Sprintf("Oneshot -%.2f%%", absDiff)
}

// calcWinnerFloat calculates the winner for float metrics (e.g., cost)
func calcWinnerFloat(ralphVal, oneshotVal float64, higherIsBetter bool) string {
	if oneshotVal == 0 {
		return "N/A"
	}

	// Calculate percentage difference
	diff := (ralphVal - oneshotVal) / oneshotVal * 100
	absDiff := math.Abs(diff)

	// Round to 2 decimal places
	absDiff = math.Round(absDiff*100) / 100

	if absDiff < 0.01 {
		return "Tie"
	}

	// Determine winner based on whether higher or lower is better
	if higherIsBetter {
		if diff > 0 {
			return fmt.Sprintf("Ralph +%.2f%%", absDiff)
		}
		return fmt.Sprintf("Oneshot +%.2f%%", absDiff)
	}

	// For cost/time/tokens, lower is better
	if diff < 0 {
		return fmt.Sprintf("Ralph -%.2f%%", absDiff)
	}
	return fmt.Sprintf("Oneshot -%.2f%%", absDiff)
}
