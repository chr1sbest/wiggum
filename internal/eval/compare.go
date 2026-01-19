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
	fmt.Println("┌────────────────┬─────────────┬─────────────┬────────────────────┐")
	fmt.Println("│         Metric │       Ralph │     Oneshot │             Winner │")
	fmt.Println("├────────────────┼─────────────┼─────────────┼────────────────────┤")
	fmt.Printf("│       Duration │ %10ds │ %10ds │ %18s │\n", ralph.DurationSeconds, oneshot.DurationSeconds, durationWinner)
	fmt.Printf("│   Total Tokens │ %11d │ %11d │ %18s │\n", ralph.TotalTokens, oneshot.TotalTokens, tokensWinner)
	fmt.Printf("│           Cost │ %11s │ %11s │ %18s │\n", fmt.Sprintf("$%.2f", ralph.CostUSD), fmt.Sprintf("$%.2f", oneshot.CostUSD), costWinner)
	fmt.Printf("│   Shared Tests │ %11s │ %11s │ %18s │\n", fmt.Sprintf("%d/%d", ralph.SharedTestsPassed, ralph.SharedTestsTotal), fmt.Sprintf("%d/%d", oneshot.SharedTestsPassed, oneshot.SharedTestsTotal), testsWinner)
	fmt.Println("└────────────────┴─────────────┴─────────────┴────────────────────┘")
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
	if oneshotVal == 0 && ralphVal == 0 {
		return "Tie"
	}
	if oneshotVal == 0 {
		return "Ralph"
	}
	if ralphVal == oneshotVal {
		return "Tie"
	}

	// Determine winner based on whether higher or lower is better
	if higherIsBetter {
		if ralphVal > oneshotVal {
			return fmt.Sprintf("Ralph (+%d)", ralphVal-oneshotVal)
		}
		return fmt.Sprintf("Oneshot (+%d)", oneshotVal-ralphVal)
	}

	// For cost/time/tokens, lower is better
	if ralphVal < oneshotVal {
		return fmt.Sprintf("Ralph (%.1fx)", float64(oneshotVal)/float64(ralphVal))
	}
	return fmt.Sprintf("Oneshot (%.1fx)", float64(ralphVal)/float64(oneshotVal))
}

// calcWinnerFloat calculates the winner for float metrics (e.g., cost)
func calcWinnerFloat(ralphVal, oneshotVal float64, higherIsBetter bool) string {
	if oneshotVal == 0 && ralphVal == 0 {
		return "Tie"
	}
	if oneshotVal == 0 {
		return "Ralph"
	}

	// Check for effective tie (within 1%)
	diff := math.Abs(ralphVal-oneshotVal) / oneshotVal
	if diff < 0.01 {
		return "Tie"
	}

	// Determine winner based on whether higher or lower is better
	if higherIsBetter {
		if ralphVal > oneshotVal {
			return fmt.Sprintf("Ralph (%.1fx)", ralphVal/oneshotVal)
		}
		return fmt.Sprintf("Oneshot (%.1fx)", oneshotVal/ralphVal)
	}

	// For cost/time/tokens, lower is better
	if ralphVal < oneshotVal {
		return fmt.Sprintf("Ralph (%.1fx)", oneshotVal/ralphVal)
	}
	return fmt.Sprintf("Oneshot (%.1fx)", ralphVal/oneshotVal)
}
