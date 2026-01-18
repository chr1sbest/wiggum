package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type githubReleaseLatestResponse struct {
	TagName string `json:"tag_name"`
}

type githubTag struct {
	Name string `json:"name"`
}

type goListModuleVersions struct {
	Versions []string `json:"Versions"`
}

func upgradeCmd(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ExitOnError)
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.Parse(args)

	latest, err := fetchLatestVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check latest version: %v\n", err)
		return 1
	}
	if latest == "" {
		fmt.Fprintln(os.Stderr, "Failed to determine latest version.")
		return 1
	}
	if latest == version {
		fmt.Printf("ralph is up to date (%s)\n", version)
		return 0
	}
	if compareSemver(latest, version) <= 0 {
		fmt.Printf("ralph is up to date (%s)\n", version)
		return 0
	}

	fmt.Printf("Current version: %s\n", version)
	fmt.Printf("Latest version:  %s\n", latest)

	if !*yes {
		fmt.Print("Upgrade now? [y/N]: ")
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		if ans != "y" && ans != "yes" {
			fmt.Println("Canceled.")
			return 0
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to determine current executable: %v\n", err)
		printManualUpgradeInstructions()
		return 1
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	exeDir := filepath.Dir(exePath)

	if looksLikeHomebrewInstall(exePath) {
		fmt.Println("This ralph install looks like it was installed via Homebrew.")
		printBrewUpgradeInstructions()
		return 0
	}

	if !looksLikeGoInstall(exeDir) {
		fmt.Println("This ralph install does not look like it was installed via `go install`.")
		printManualUpgradeInstructions()
		return 0
	}
	if err := checkWritable(exePath); err != nil {
		fmt.Printf("Current binary is not writable (%v).\n", err)
		printManualUpgradeInstructions()
		return 0
	}
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Println("`go` not found in PATH.")
		printManualUpgradeInstructions()
		return 0
	}

	cmd := exec.Command("go", "install", "github.com/chr1sbest/wiggum/cmd/ralph@latest")
	cmd.Env = append(os.Environ(), "GOBIN="+exeDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Upgrade failed: %v\n", err)
		outText := strings.TrimSpace(string(out))
		if outText != "" {
			fmt.Fprintln(os.Stderr, outText)
			if strings.Contains(outText, "module declares its path as:") && strings.Contains(outText, "but was required as:") {
				fmt.Fprintln(os.Stderr, "\nThe latest published version appears to have an incorrect module path in go.mod.")
				fmt.Fprintln(os.Stderr, "Publish a new tag/release after fixing go.mod's `module` line to match the GitHub repo, then re-run: ralph upgrade")
			}
		}
		printManualUpgradeInstructions()
		return 1
	}
	if strings.TrimSpace(string(out)) != "" {
		fmt.Println(string(out))
	}
	fmt.Printf("Upgraded to latest. Run: ralph version\n")
	return 0
}

func fetchLatestVersion() (string, error) {
	url := "https://api.github.com/repos/chr1sbest/wiggum/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fetchLatestVersionFromTagsOrGo()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("unexpected response: %s (%s)", resp.Status, strings.TrimSpace(string(b)))
	}
	var r githubReleaseLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.TrimPrefix(r.TagName, "v")), nil
}

func fetchLatestVersionFromTagsOrGo() (string, error) {
	v, err := fetchLatestVersionFromTags()
	if err == nil && strings.TrimSpace(v) != "" {
		return v, nil
	}
	return fetchLatestVersionFromGo()
}

func fetchLatestVersionFromTags() (string, error) {
	url := "https://api.github.com/repos/chr1sbest/wiggum/tags?per_page=100"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("unexpected response: %s (%s)", resp.Status, strings.TrimSpace(string(b)))
	}
	var tags []githubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}
	best := ""
	for _, t := range tags {
		v := strings.TrimSpace(strings.TrimPrefix(t.Name, "v"))
		if v == "" {
			continue
		}
		if best == "" || compareSemver(v, best) > 0 {
			best = v
		}
	}
	return best, nil
}

func fetchLatestVersionFromGo() (string, error) {
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("go not found in PATH")
	}
	mod := "github.com/chr1sbest/wiggum"
	cmd := exec.Command("go", "list", "-mod=mod", "-m", "-json", "-versions", mod)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return "", fmt.Errorf("go list failed: %w (%s)", err, msg)
		}
		return "", fmt.Errorf("go list failed: %w", err)
	}
	var m goListModuleVersions
	if err := json.Unmarshal(out, &m); err != nil {
		return "", err
	}
	best := ""
	for _, raw := range m.Versions {
		v := strings.TrimSpace(strings.TrimPrefix(raw, "v"))
		if v == "" {
			continue
		}
		if best == "" || compareSemver(v, best) > 0 {
			best = v
		}
	}
	return best, nil
}

func printManualUpgradeInstructions() {
	fmt.Println("To upgrade manually, run:")
	fmt.Println("  go install github.com/chr1sbest/wiggum/cmd/ralph@latest")
}

func printBrewUpgradeInstructions() {
	fmt.Println("To upgrade via Homebrew, run:")
	fmt.Println("  brew update")
	fmt.Println("  brew upgrade ralph")
}

func looksLikeHomebrewInstall(exePath string) bool {
	p := filepath.Clean(exePath)
	if rp, err := filepath.EvalSymlinks(p); err == nil {
		p = rp
	}
	// Homebrew installs versioned packages under the Cellar.
	if strings.Contains(p, string(filepath.Separator)+"Cellar"+string(filepath.Separator)) {
		return true
	}
	// Heuristic: some setups might not include Cellar in the resolved path.
	if strings.Contains(p, string(filepath.Separator)+"Homebrew"+string(filepath.Separator)) {
		return true
	}
	return false
}

func looksLikeGoInstall(exeDir string) bool {
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		if samePath(exeDir, gobin) {
			return true
		}
	}
	gopath := strings.TrimSpace(os.Getenv("GOPATH"))
	if gopath == "" {
		return false
	}
	for _, p := range strings.Split(gopath, string(os.PathListSeparator)) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		binDir := filepath.Join(p, "bin")
		if samePath(exeDir, binDir) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	aa := filepath.Clean(a)
	bb := filepath.Clean(b)
	if aa == bb {
		return true
	}
	if ra, err := filepath.EvalSymlinks(aa); err == nil {
		aa = ra
	}
	if rb, err := filepath.EvalSymlinks(bb); err == nil {
		bb = rb
	}
	return aa == bb
}

func checkWritable(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
}
