package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alexcb/earthlymacmon/slack"
)

func getCurrentVersion() (string, error) {
	url := "https://formulae.brew.sh/api/formula/earthly.json"

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "earthlymacmon")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var data struct {
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	}
	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		return "", err
	}
	return data.Versions.Stable, nil

}

func getLastRunVersion() (string, error) {
	data, err := ioutil.ReadFile("/tmp/last-earth-check")
	return string(data), err
}

func setLastRunVersion(version string) error {
	return ioutil.WriteFile("/tmp/last-earth-check", []byte(version), 0644)
}

func doPreInstall(version string) (string, bool) {
	cmd := exec.Command("sh", "-c", "echo running on `hostname -f`")
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err == nil
}

func doInstall(version string) (string, bool) {
	cmd := exec.Command("sh", "-c", "brew upgrade earthly")
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err == nil
}

func doChecVersion(version string) (string, bool) {
	cmd := exec.Command("sh", "-c", "earth --version")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return string(stdoutStderr), false
	}

	output := string(stdoutStderr)
	if !strings.Contains(output, version) {
		output = fmt.Sprintf("expected to find version string %q in %q", version, output)
		return output, false
	}

	return output, true
}

func doTestRun(version string) (string, bool) {
	testCmd := "earth github.com/earthly/earthly/examples/go+docker"
	cmd := exec.Command("sh", "-c", testCmd)
	stdoutStderr, err := cmd.CombinedOutput()
	return "$ " + testCmd + "\n" + string(stdoutStderr), err == nil
}

func doTest(alerter slack.Alerter, version string) {
	fmt.Printf("detected new version %q; running test\n", version)

	ok := true
	status := []slack.SubAlert{}

	for _, test := range []struct {
		title string
		fun   func(string) (string, bool)
	}{
		{"pre-installation", doPreInstall},
		{"installation", doInstall},
		{"check version", doChecVersion},
		{"test run", doTestRun},
	} {
		var output string
		output, ok = test.fun(version)
		status = append(status, slack.SubAlert{
			Title:  test.title,
			Output: output,
		})
		if !ok {
			break
		}
	}
	title := fmt.Sprintf("tests for version %q OK", version)
	if !ok {
		title = fmt.Sprintf("tests for version %q failed", version)
	}
	alerter.Alert(title, status, ok)
}

func maybeDoTest(alerter slack.Alerter) {
	ver, err := getCurrentVersion()
	if err != nil {
		panic(err)
	}

	lastRunVersion, err := getLastRunVersion()
	if err != nil {
		lastRunVersion = "unknown"
	}

	if lastRunVersion == ver {
		fmt.Printf("current homebrew version is %v and is up to date with testing\n", ver)
		return
	}

	doTest(alerter, ver)

	err = setLastRunVersion(ver)
	if err != nil {
		panic(err)
	}

}

func main() {

	webHookURL := os.Getenv("EARTHLY_ALERT_WEBHOOK")
	if webHookURL == "" {
		panic(fmt.Sprintf("failed to get EARTHLY_ALERT_WEBHOOK"))
	}

	alerter := slack.NewSlackAlerter(webHookURL)

	for {
		maybeDoTest(alerter)
		time.Sleep(time.Minute * 5)
	}
}
