// -----------------------------------------------------------------------
// Last Modified: Wednesday, 1st October 2025 10:45:00 pm
// Modified By: Bob McAllan
// -----------------------------------------------------------------------

package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

const chromeExecutablePath = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
const extensionPath = "C:\\development\\aktis\\aktis-collector-jira\\bin\\aktis-chrome-extension"

func main() {
	log.Printf("Starting Chromedp without pre-loaded extension")
	log.Printf("Extension will need to be manually loaded from: %s", extensionPath)

	// Remove the load-extension flags - they don't work reliably with chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.Flag("disable-gpu", true),
		chromedp.ExecPath(chromeExecutablePath),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	log.Printf("Navigating to Jira projects page and attempting login...")

	loginActions := chromedp.Tasks{
		chromedp.Navigate(`https://bobmcallan.atlassian.net/jira/projects`),
		chromedp.Sleep(2 * time.Second),
		chromedp.WaitVisible(`a[href*="login"]`, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelector('a[href*="login"]').click()`, nil),
		chromedp.Sleep(3 * time.Second),
		chromedp.WaitVisible(`input[name="username"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="username"]`, "bobmcallan@gmail.com", chromedp.ByQuery),
		chromedp.Sleep(500 * time.Millisecond),
		chromedp.Evaluate(`document.querySelector('button[type="submit"]').click()`, nil),
		chromedp.Sleep(2 * time.Second),
		chromedp.WaitVisible(`input[name="password"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, "t!!7LL4#j%Wej4", chromedp.ByQuery),
		chromedp.Sleep(500 * time.Millisecond),
		chromedp.Evaluate(`document.querySelector('button[type="submit"]').click()`, nil),
		chromedp.Sleep(5 * time.Second),
	}

	err := chromedp.Run(ctx, loginActions)

	if err != nil {
		log.Fatalf("Failed to execute login sequence: %v", err)
	}

	log.Printf("Successfully completed login sequence!")

	// Navigate to chrome://extensions and enable developer mode
	err = chromedp.Run(ctx,
		chromedp.Navigate(`chrome://extensions/`),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
			(function() {
				const manager = document.querySelector('extensions-manager');
				if (manager && manager.shadowRoot) {
					const toggle = manager.shadowRoot.querySelector('#devMode');
					if (toggle && !toggle.checked) {
						toggle.click();
						return 'Developer mode enabled';
					}
					return 'Developer mode already enabled';
				}
				return 'Could not find developer mode toggle';
			})()
		`, nil),
		chromedp.Sleep(1*time.Second),
	)

	if err != nil {
		log.Printf("Warning: Could not enable developer mode: %v", err)
	} else {
		log.Printf("Developer mode toggle attempted")
	}

	log.Printf("\n========================================")
	log.Printf("MANUAL STEPS:")
	log.Printf("1. Click 'Load unpacked' button")
	log.Printf("2. Navigate to: %s", extensionPath)
	log.Printf("3. Click 'Select Folder'")
	log.Printf("4. Navigate to: https://bobmcallan.atlassian.net/jira/projects")
	log.Printf("5. Click the extension icon to test")
	log.Printf("========================================\n")

	log.Printf("Browser will remain open for 3 minutes...")
	time.Sleep(180 * time.Second)
}
