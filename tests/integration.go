// Integration test: Start collector server, open browser with extension, verify data collection

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

const extensionPath = "C:\\development\\aktis\\aktis-collector-jira\\bin\\aktis-chrome-extension"
const collectorPath = "C:\\development\\aktis\\aktis-collector-jira\\bin\\aktis-collector-jira.exe"
const targetUrl = "https://bobmcallan.atlassian.net/jira/projects"

func main() {
	log.SetFlags(log.Ltime)

	// Step 1: Start the collector server
	log.Printf("Starting Aktis collector server...")
	collectorCmd := exec.Command(collectorPath, "-config", "deployments/aktis-collector-jira.toml")
	collectorCmd.Stdout = os.Stdout
	collectorCmd.Stderr = os.Stderr

	if err := collectorCmd.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
	defer collectorCmd.Process.Kill()

	log.Printf("Collector server started (PID: %d)", collectorCmd.Process.Pid)
	log.Printf("Waiting 3 seconds for server to initialize...")
	time.Sleep(3 * time.Second)

	// Step 2: Install Playwright if needed
	if err := playwright.Install(); err != nil {
		log.Fatalf("Could not install playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Could not start playwright: %v", err)
	}
	defer pw.Stop()

	// Step 3: Launch browser with extension
	log.Printf("Launching Chrome with extension...")
	tempDir, err := os.MkdirTemp("", "integration-test-")
	if err != nil {
		log.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	browser, err := pw.Chromium.LaunchPersistentContext(tempDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--load-extension=" + extensionPath,
		},
		Timeout: playwright.Float(60000),
	})
	if err != nil {
		log.Fatalf("Could not launch browser: %v", err)
	}
	defer browser.Close()

	var page playwright.Page
	if len(browser.Pages()) > 0 {
		page = browser.Pages()[0]
	} else {
		page, err = browser.NewPage()
		if err != nil {
			log.Fatalf("Could not create page: %v", err)
		}
	}

	// Step 4: Login to Jira
	log.Printf("Navigating to Jira and logging in...")
	if _, err := page.Goto(targetUrl); err != nil {
		log.Fatalf("Could not navigate: %v", err)
	}

	// Click login
	if err := page.Locator("a[href*='login.jsp']").Click(); err != nil {
		log.Fatalf("Could not click login: %v", err)
	}

	// Enter credentials
	page.WaitForSelector("input[name='username']", playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	page.Fill("input[name='username']", "bobmcallan@gmail.com")
	page.Click("button[type='submit']")

	page.WaitForSelector("input[name='password']", playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	page.Fill("input[name='password']", "t!!7LL4#j%Wej4")
	page.Click("button[type='submit']")

	log.Printf("Waiting for login to complete...")
	page.WaitForTimeout(5000)

	log.Printf("Successfully logged in!")

	// Step 5: Navigate to projects page and wait for auto-collection
	log.Printf("Navigating to projects page...")
	if _, err := page.Goto(targetUrl); err != nil {
		log.Fatalf("Could not navigate to projects: %v", err)
	}

	log.Printf("Waiting 10 seconds for extension auto-collection to trigger...")
	page.WaitForTimeout(10000)

	// Step 6: Check console for extension activity
	log.Printf("Checking browser console for extension activity...")
	messages, _ := page.Evaluate(`() => {
		return window.console.messages || [];
	}`)
	log.Printf("Console messages: %v", messages)

	// Step 7: Keep browser open for manual verification
	log.Printf("\n========================================")
	log.Printf("VERIFICATION STEPS:")
	log.Printf("========================================")
	log.Printf("1. Browser is open with Jira projects page")
	log.Printf("2. Extension should be loaded (check toolbar)")
	log.Printf("3. Check collector logs for received data")
	log.Printf("4. Navigate to http://localhost:8080 to see collected data")
	log.Printf("5. Browser will close in 60 seconds...")
	log.Printf("========================================\n")

	time.Sleep(60 * time.Second)

	log.Printf("Test complete!")
}
