package banner

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/chr1sbest/wiggum/internal/config"
)

var quoteRand = rand.New(rand.NewSource(time.Now().UnixNano()))

var ralphQuotes = []string{
	"Me fail English? That's unpossible!",
	"My cat's breath smells like cat food.",
	"They taste like... burning.",
	"I bent my Wookiee!",
	"When I grow up, I want to be a principal or a caterpillar.",
}

// ANSI color codes
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dim       = "\033[2m"
	cyan      = "\033[36m"
	blue      = "\033[34m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	magenta   = "\033[35m"
	white     = "\033[37m"
	boldCyan  = "\033[1;36m"
	boldGreen = "\033[1;32m"
)

// Box drawing characters
const (
	topLeft     = "╭"
	topRight    = "╮"
	bottomLeft  = "╰"
	bottomRight = "╯"
	horizontal  = "─"
	vertical    = "│"
	bullet      = "●"
	arrow       = "→"
)

// Banner handles pretty startup output
type Banner struct {
	writer io.Writer
	width  int
}

const ralphASCII = `⠀⠀⠀⠀⠀⠀⣀⣤⣶⡶⢛⠟⡿⠻⢻⢿⢶⢦⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⢀⣠⡾⡫⢊⠌⡐⢡⠊⢰⠁⡎⠘⡄⢢⠙⡛⡷⢤⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⢠⢪⢋⡞⢠⠃⡜⠀⠎⠀⠉⠀⠃⠀⠃⠀⠃⠙⠘⠊⢻⠦⠀⠀⠀⠀⠀⠀
⠀⠀⢇⡇⡜⠀⠜⠀⠁⠀⢀⠔⠉⠉⠑⠄⠀⠀⡰⠊⠉⠑⡄⡇⠀⠀⠀⠀⠀⠀
⠀⠀⡸⠧⠄⠀⠀⠀⠀⠀⠘⡀⠾⠀⠀⣸⠀⠀⢧⠀⠛⠀⠌⡇⠀⠀⠀⠀⠀⠀
⠀⠘⡇⠀⠀⠀⠀⠀⠀⠀⠀⠙⠒⠒⠚⠁⠈⠉⠲⡍⠒⠈⠀⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠈⠲⣆⠀⠀⠀⠀⠀⠀⠀⠀⣠⠖⠉⡹⠤⠶⠁⠀⠀⠀⠈⢦⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠈⣦⡀⠀⠀⠀⠀⠧⣴⠁⠀⠘⠓⢲⣄⣀⣀⣀⡤⠔⠃⠀⠀⠀⠀⠀
⠀⠀⠀⠀⣜⠀⠈⠓⠦⢄⣀⣀⣸⠀⠀⠀⠀⠁⢈⢇⣼⡁⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⢠⠒⠛⠲⣄⠀⠀⠀⣠⠏⠀⠉⠲⣤⠀⢸⠋⢻⣤⡛⣄⠀⠀⠀⠀⠀⠀⠀
⠀⠀⢡⠀⠀⠀⠀⠉⢲⠾⠁⠀⠀⠀⠀⠈⢳⡾⣤⠟⠁⠹⣿⢆⠀⠀⠀⠀⠀⠀
⠀⢀⠼⣆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣼⠃⠀⠀⠀⠀⠀⠈⣧⠀⠀⠀⠀⠀
⠀⡏⠀⠘⢦⡀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠞⠁⠀⠀⠀⠀⠀⠀⠀⢸⣧⠀⠀⠀⠀
⢰⣄⠀⠀⠀⠉⠳⠦⣤⣤⡤⠴⠖⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢯⣆⠀⠀⠀
⢸⣉⠉⠓⠲⢦⣤⣄⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣀⣀⣠⣼⢹⡄⠀⠀
⠘⡍⠙⠒⠶⢤⣄⣈⣉⡉⠉⠙⠛⠛⠛⠛⠛⠛⢻⠉⠉⠉⢙⣏⣁⣸⠇⡇⠀⠀
⠀⢣⠀⠀⠀⠀⠀⠀⠉⠉⠉⠙⠛⠛⠛⠛⠛⠛⠛⠒⠒⠒⠋⠉⠀⠸⠚⢇⠀⠀
⠀⠀⢧⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠇⢤⣨⠇⠀
⠀⠀⠀⢧⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣤⢻⡀⣸⠀⠀⠀
⠀⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⠛⠉⠁⠀⠀⠀
⠀⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠀⢠⢄⣀⣤⠤⠴⠒⠀⠀⠀⠀⢸⠀⠀⠀⠀⠀⠀
⠀⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠘⡆⠀⠀⠀⠀⠀
⠀⠀⠀⡎⠀⠀⠀⠀⠀⠀⠀⠀⢷⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠀
⠀⠀⢀⡷⢤⣤⣀⣀⣀⣀⣠⠤⠾⣤⣀⡘⠛⠶⠶⠶⠶⠖⠒⠋⠙⠓⠲⢤⣀⠀
⠀⠀⠘⠧⣀⡀⠈⠉⠉⠁⠀⠀⠀⠀⠈⠙⠳⣤⣄⣀⣀⣀⠀⠀⠀⠀⠀⢀⣈⡇
⠀⠀⠀⠀⠀⠉⠛⠲⠤⠤⢤⣤⣄⣀⣀⣀⣀⡸⠇⠀⠀⠀⠉⠉⠉⠉⠉⠉⠁⠀`

// New creates a new Banner that writes to stdout
func New() *Banner {
	return &Banner{
		writer: os.Stdout,
		width:  60,
	}
}

// NewWithWriter creates a Banner with a custom writer (for testing)
func NewWithWriter(w io.Writer) *Banner {
	return &Banner{
		writer: w,
		width:  60,
	}
}

// Print displays the startup banner with config information
func (b *Banner) Print(cfg *config.Config) {
	_ = cfg
	b.printHeader()
	fmt.Fprintf(b.writer, "%s%s%s\n", cyan, ralphASCII, reset)
	b.printFooter()
}

func (b *Banner) printHeader() {
	// Top border
	fmt.Fprintf(b.writer, "\n%s%s%s%s%s\n", dim, topLeft, strings.Repeat(horizontal, b.width-2), topRight, reset)

	// Title line
	titleText := "Ralph"
	title := fmt.Sprintf("  %s%s%s%s", bold, blue, titleText, reset)
	padding := b.width - visualLen(titleText) - 4
	fmt.Fprintf(b.writer, "%s%s%s%s%s%s\n", dim, vertical, reset, title, strings.Repeat(" ", padding), dim+vertical+reset)

	subtitleText := randomRalphQuote()
	maxSubtitleLen := b.width - 4
	if maxSubtitleLen < 0 {
		maxSubtitleLen = 0
	}
	if visualLen(subtitleText) > maxSubtitleLen {
		if maxSubtitleLen <= 3 {
			subtitleText = subtitleText[:maxSubtitleLen]
		} else {
			subtitleText = subtitleText[:maxSubtitleLen-3] + "..."
		}
	}
	sub := fmt.Sprintf("  %s%s%s", dim, subtitleText, reset)
	subPadding := b.width - visualLen(subtitleText) - 4
	if subPadding < 0 {
		subPadding = 0
	}
	fmt.Fprintf(b.writer, "%s%s%s%s%s%s\n", dim, vertical, reset, sub, strings.Repeat(" ", subPadding), dim+vertical+reset)

	// Separator
	fmt.Fprintf(b.writer, "%s%s%s%s%s\n", dim, vertical, strings.Repeat(horizontal, b.width-2), vertical, reset)
}

func (b *Banner) printFooter() {
	// Bottom border with start indicator
	fmt.Fprintf(b.writer, "%s%s%s%s%s\n", dim, bottomLeft, strings.Repeat(horizontal, b.width-2), bottomRight, reset)
	fmt.Fprintf(b.writer, "\n")
}

// visualLen returns the visual length of a string (excluding ANSI codes)
func visualLen(s string) int {
	return len(s)
}

func randomRalphQuote() string {
	if len(ralphQuotes) == 0 {
		return "Automation loop with Claude agent"
	}
	return ralphQuotes[quoteRand.Intn(len(ralphQuotes))]
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
