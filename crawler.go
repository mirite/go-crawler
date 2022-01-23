package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	s "strings"
)

var fileExtensionPattern, _ = regexp.Compile(`\.[a-zA-Z]+$`)
var urlPattern, _ = regexp.Compile(`href=['\"]([\w\.\:\?\-\%\/#]+)['\"]`)
var urlPreAnchorPattern, _ = regexp.Compile(`#[\w\-\_]+`)
var startingURL string = ""
var allValidURLsFound = make([]string, 1)
var allBadURLsFound = make([]string, 1)
var actualPagesFound = make([]string, 1)
var pagesRemianingToBeChecked = make([]string, 1)
var startingHost = ""
var extensionsAllowList = []string{"", ".html", ".htm", ".php", ".asp"}

type outputResults struct {
	Pages  []string
	Errors []string
}

func main() {
	initialize()
	for len(pagesRemianingToBeChecked) > 0 {
		checkNextPage()
	}
	writeResults()
}

func initialize() {
	arg := os.Args[1]
	startingURL = arg
	startingHost = getHost(startingURL)
	allValidURLsFound[0] = startingURL
	pagesRemianingToBeChecked[0] = startingURL
	actualPagesFound[0] = startingURL

}

func writeResults() {
	enc := json.NewEncoder(os.Stdout)
	output := &outputResults{
		Pages:  allValidURLsFound,
		Errors: allBadURLsFound,
	}
	enc.Encode(output)

}

func checkNextPage() {
	log.Println("Pages to check: ", len(pagesRemianingToBeChecked), " Pages found: ", len(allValidURLsFound))
	currentPageURL := getNextPageToCheck()
	removeFromPagesToCheck()
	currentPageSource := getPageBody(currentPageURL)
	if isHTMLFile(currentPageSource) {
		processHTMLPage(currentPageURL, currentPageSource)
	}
}

func processHTMLPage(currentPageURL string, currentPageSource string) {
	actualPagesFound = append(actualPagesFound, currentPageURL)
	currentPageOutboundURLs := getURLsFromPage(currentPageSource)

	for i := 0; i < len(currentPageOutboundURLs); i++ {
		processOutboundURL(currentPageURL, currentPageOutboundURLs[i])
	}
}

func processOutboundURL(currentPageURL string, rawOutboundURL string) {
	cleanedOutboundURL := resolveURL(currentPageURL, rawOutboundURL)
	if isPageInScope(startingHost, cleanedOutboundURL) {
		addFoundURL(cleanedOutboundURL)
	}
}
func addFoundURL(url string) {
	pagesRemianingToBeChecked = append(pagesRemianingToBeChecked, url)
	allValidURLsFound = append(allValidURLsFound, url)
}

func getNextPageToCheck() string {
	return pagesRemianingToBeChecked[0]
}

func isHTMLFile(body string) bool {
	return s.Contains(body, "<!DOCTYPE")
}

func removeFromPagesToCheck() {
	pagesRemianingToBeChecked = pagesRemianingToBeChecked[1:]
}

func isPageInScope(startingHost string, url string) bool {
	currentPageHost := getHost(url)
	if currentPageHost != startingHost {
		return false
	}

	if contains(allValidURLsFound, url) {
		return false
	}

	if contains(pagesRemianingToBeChecked, url) {
		return false
	}

	currentPageExtension := getExtension(url)
	if !contains(extensionsAllowList, currentPageExtension) {
		return false
	}

	return true
}

func getHost(urlToParse string) string {
	u, err := url.Parse(urlToParse)
	if err != nil {
		panic(err)
	}
	return u.Host
}

func getExtension(urlToParse string) string {
	return fileExtensionPattern.FindString(urlToParse)

}

func getPageBody(url string) string {
	log.Println("Requesting " + url)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode > 399 {
		allBadURLsFound = append(allBadURLsFound, url)
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	s := ""
	for scanner.Scan() {
		s += scanner.Text()
	}
	return s
}

func getURLsFromPage(s string) []string {

	results := urlPattern.FindAllStringSubmatch(s, -1)
	output := make([]string, len(results))
	for i := 0; i < len(results); i++ {
		output = append(output, results[i][1])
	}
	return output
}

func resolveURL(currentURL string, outboundUrl string) string {
	outboundUrlWithoutAnchor := urlPreAnchorPattern.ReplaceAllString(outboundUrl, "")
	u, err := url.Parse(outboundUrlWithoutAnchor)
	if err != nil {
		log.Fatal(err)
	}
	base, err := url.Parse(currentURL)
	if err != nil {
		log.Fatal(err)
	}
	return base.ResolveReference(u).String()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if isLooseMatch(a, e) {
			return true
		}
	}
	return false
}

func isLooseMatch(a string, b string) bool {
	existinVariant := a + "/"
	newVariant := b + "/"
	return (a == b) || (existinVariant == b) || (a == newVariant)
}
