// -----------------------------------------------------------------------
// Playwright-based browser test with extension loading
// -----------------------------------------------------------------------

package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

const extensionPath = "C:\\development\\aktis\\aktis-collector-jira\\bin\\aktis-chrome-extension"

// The target URL to automate
const targetUrl = "https://bobmcallan.atlassian.net/jira/projects"

func main() {
	// Install playwright browsers if needed
	err := playwright.Install()
	if err != nil {
		log.Fatalf("Could not install playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Could not start playwright: %v", err)
	}
	defer pw.Stop()

	log.Printf("Launching Chrome with extension from: %s", extensionPath)

	// Use a temporary directory for isolation
	tempDir, err := os.MkdirTemp("", "playwright-profile-")
	if err != nil {
		log.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Launch browser with extension
	browser, err := pw.Chromium.LaunchPersistentContext(tempDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(false),
		// Use the correct flags to load the unpacked extension
		Args: []string{
			"--load-extension=" + extensionPath,
		},
		Timeout: playwright.Float(60000), // 60 seconds timeout for launch
	})
	if err != nil {
		log.Fatalf("Could not launch browser: %v", err)
	}
	defer browser.Close()

	// Get the initial page (or create a new one if none exists)
	var page playwright.Page
	if len(browser.Pages()) > 0 {
		page = browser.Pages()[0]
	} else {
		page, err = browser.NewPage()
		if err != nil {
			log.Fatalf("Could not create new page: %v", err)
		}
	}

	// --- LOGIN SEQUENCE ---
	log.Printf("Navigating to Jira projects page...")
	if _, err := page.Goto(targetUrl, playwright.PageGotoOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		log.Fatalf("Could not navigate: %v", err)
	}

	// Wait for and click login link
	log.Printf("Waiting for login link...")
	if err := page.Locator("a[href*='login.jsp']").Click(); err != nil {
		log.Fatalf("Could not click login: %v", err)
	}

	// Enter username
	log.Printf("Entering username...")
	page.WaitForSelector("input[name='username']", playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err := page.Fill("input[name='username']", "bobmcallan@gmail.com"); err != nil {
		log.Fatalf("Could not enter username: %v", err)
	}

	// Click submit
	if err := page.Click("button[type='submit']"); err != nil {
		log.Fatalf("Could not click submit: %v", err)
	}

	// Enter password
	log.Printf("Entering password...")
	page.WaitForSelector("input[name='password']", playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err := page.Fill("input[name='password']", "t!!7LL4#j%Wej4"); err != nil {
		log.Fatalf("Could not enter password: %v", err)
	}

	// Click submit
	if err := page.Click("button[type='submit']"); err != nil {
		log.Fatalf("Could not click submit: %v", err)
	}

	// Wait for navigation to complete
	log.Printf("Waiting for login to complete and dashboard to load...")
	page.WaitForTimeout(5000)

	log.Printf("Successfully logged in!")

	// --- EXTENSION OPENING ---

	log.Printf("Accessing extension background page...")
	time.Sleep(2 * time.Second) // Give extension time to load

	// Method 1: Get background pages (most reliable for Playwright)
	backgrounds := browser.BackgroundPages()
	log.Printf("Found %d background pages", len(backgrounds))

	var extensionID string
	for _, bg := range backgrounds {
		url := bg.URL()
		log.Printf("Background page URL: %s", url)

		if strings.Contains(url, "chrome-extension://") {
			// Extract extension ID from URL
			parts := strings.Split(url, "chrome-extension://")
			if len(parts) > 1 {
				idParts := strings.Split(parts[1], "/")
				extensionID = idParts[0]
				log.Printf("Found extension ID: %s", extensionID)

				// Now trigger the side panel from the background page
				log.Printf("Opening side panel via background page...")
				result, err := bg.Evaluate(`() => {
					if (chrome.sidePanel && chrome.sidePanel.open) {
						return chrome.sidePanel.open().then(() => 'opened').catch(e => e.message);
					}
					return 'API not available';
				}`)

				if err != nil {
					log.Printf("Error opening side panel: %v", err)
				} else {
					log.Printf("Side panel open result: %v", result)
				}

				break
			}
		}
	}

	// Method 2: Monitor for side panel page to appear
	log.Printf("Monitoring for side panel to open...")
	time.Sleep(2 * time.Second)

	for _, p := range browser.Pages() {
		url := p.URL()
		if strings.Contains(url, extensionID) && strings.Contains(url, "sidepanel") {
			log.Printf("Side panel detected at: %s", url)

			// Now you can interact with the side panel
			title, _ := p.Title()
			log.Printf("Side panel title: %s", title)

			// Example: get content from side panel
			content, _ := p.Content()
			log.Printf("Side panel loaded with %d bytes of content", len(content))
		}
	}

	log.Printf("Browser will remain open for 3 minutes for manual inspection...")

	// Keep browser open for 3 minutes
	time.Sleep(180 * time.Second)
}
