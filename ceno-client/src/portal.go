package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/nicksnyder/go-i18n/i18n"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
)

type PortalPath struct {
	PageName string
	Href     string
}

// Location of the JSON file containing the merged translated strings
var allJSONPath string = path.Join(".", "locale", "all.json")

/**
 * ceno-client/locale/all.json contains data formatted like
{
 	"en": {
		"string1": "content content content"
	},
	"fr": {
		"string1": "french french french"
	}
}
*/
type IdentifiedString struct {
	Identifier string
	Content    string
}

type LanguageStrings struct {
	Name    string
	Locale  string
	Strings []IdentifiedString
}

type LanguageStringJSON map[string]map[string]string

/**
 * JSON files containing information about articles stored in the distributed cache (Freenet)
 * are named like `json-files/<base64(feed's url)>.json`
 * @param feedUrl - The URL of the RSS/Atom feed to retrieve information about articles from
 * @return the path to the article of interest's respective JSON file on disk
 */
func articlesFilename(feedUrl string) string {
	b64FeedUrl := base64.StdEncoding.EncodeToString([]byte(feedUrl))
	return path.Join(".", "json-files", b64FeedUrl+".json")
}

/**
 * Convert a URL like /cenosite/<base64(url)> to just the contained url.
 * @param feedUrl - A portal-internal URL for a feed
 * @return the original feed's URL and any error that occurs parsing it out
 */
func getFeedUrl(feedUrl string) (string, error) {
	parts := strings.Split(feedUrl, "/")
	b64FeedUrl := parts[len(parts)-1]
	decoded, decodeErr := base64.StdEncoding.DecodeString(b64FeedUrl)
	if decodeErr != nil {
		return "", decodeErr
	}
	return string(decoded), nil
}

/**
 * Get information about feeds to be injected into the portal page.
 * @return a map with a "feeds" key and corresponding array of Feed structs and an optional error
 */
func InitModuleWithFeeds() (map[string]interface{}, error) {
	feedInfo := FeedInfo{}
	var decodeErr error
	// Download the latest feeds list from the LCS
	result := Lookup(FeedsJsonFile) // Defined in data.go
	if result.Complete && result.Found {
		// Serve whatever the LCS gave us as the most recent feeds file.
		// After the first complete lookup, others will be served from the LCS's cache.
		decoder := json.NewDecoder(bytes.NewReader([]byte(result.Bundle)))
		decodeErr = decoder.Decode(&feedInfo)
	} else {
		// Before the first complete lookup, serve from the files distributed with
		// the client to keep the user experience fast.
		feedInfoFile, openErr := os.Open(FEED_LIST_FILENAME)
		if openErr != nil {
			return nil, openErr
		}
		defer feedInfoFile.Close()
		decoder := json.NewDecoder(feedInfoFile)
		decodeErr = decoder.Decode(&feedInfo)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	// Convert the URLs of feeds to the form that the CENO Client can handle directly, when clicked
	for i, feed := range feedInfo.Feeds {
		url := feed.Url
		feedInfo.Feeds[i].Url = "cenosite/" + base64.StdEncoding.EncodeToString([]byte(url))
	}
	var err error = nil
	mapping := make(map[string]interface{})
	mapping["Feeds"] = feedInfo.Feeds
	mapping["Version"] = feedInfo.Version
	return mapping, err
}

/**
 * Get information about articles from a given feed to be injected into the portal page.
 * @param {string} feedUrl - The URL of the feed to fetch articles from
 * @return a map with a "feeds" key and corresponding array of Feed structs and an optional error
 */
func InitModuleWithArticles(feedUrl string) (map[string]interface{}, error) {
	T, _ := i18n.Tfunc(os.Getenv(LANG_ENVVAR), DEFAULT_LANG)
	articleInfo := ArticleInfo{}
	var decodeErr error
	result := Lookup(feedUrl)
	if result.Complete && result.Found {
		fmt.Println("Lookup is complete")
		// Serve whatever the LCS gave us as the most recent articles list for
		// the feed we want to see.
		decoder := json.NewDecoder(bytes.NewReader([]byte(result.Bundle)))
		decodeErr = decoder.Decode(&articleInfo)
	} else {
		fmt.Println("Lookup is not complete")
		// Before the first complete lookup, serve from the files
		// distributed with the client.
		articleInfoFile, openErr := os.Open(articlesFilename(feedUrl))
		if openErr != nil {
			fmt.Println("Got file open error", openErr.Error())
			return nil, openErr
		}
		defer articleInfoFile.Close()
		decoder := json.NewDecoder(articleInfoFile)
		decodeErr = decoder.Decode(&articleInfo)
	}
	if decodeErr != nil {
		fmt.Println("Got decode error", decodeErr.Error())
		return nil, decodeErr
	}
	mapping := make(map[string]interface{})
	// We want to get the feed's title to display on the articles page, however we cannot simply
	// scan through the feeds.json file on disk, because we might be serving from what the LCS is giving us.
	feedsModule, feedErr := InitModuleWithFeeds()
	if feedErr != nil {
		return nil, feedErr
	}
	mapping["Title"] = T("feed_not_found", map[string]string{"FeedUrl": feedUrl})
	fmt.Println("Trying to find title for feed with url", feedUrl)
	for _, feed := range feedsModule["Feeds"].([]Feed) {
		actualFeedUrl, urlErr := getFeedUrl(feed.Url)
		if urlErr != nil {
			continue
		}
		if actualFeedUrl == feedUrl {
			// We will always find a title eventually unless the user messed up and accidentally changed the
			// feed url in the address bar.
			fmt.Println("Found feed with title", feed.Title)
			mapping["Title"] = feed.Title
			break
		}
	}
	mapping["Articles"] = articleInfo.Items
	mapping["Version"] = articleInfo.Version
	return mapping, nil
}

func stringifyLanguages(langStrings LanguageStringJSON) string {
	asBytes, _ := json.Marshal(langStrings)
	return string(asBytes)
}

func loadLanguageStrings() ([]LanguageStrings, LanguageStringJSON, error) {
	// Dear Glob.
	langStrings := make(LanguageStringJSON)
	decodedStrings := []LanguageStrings{}
	file, err := os.Open(allJSONPath)
	if err != nil {
		return decodedStrings, langStrings, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decodeErr := decoder.Decode(&langStrings)
	if decodeErr != nil {
		return decodedStrings, langStrings, decodeErr
	}
	// Use the configuration as a guide to explore the merged languages json file and construct
	// structures containing all the information relevant to the portal page about text.
	for _, availableLanguage := range Configuration.PortalLanguages {
		stringPairs, found := langStrings[availableLanguage.Locale]
		if !found {
			continue
		}
		languageStrings := LanguageStrings{}
		languageStrings.Name = availableLanguage.Name
		languageStrings.Locale = availableLanguage.Locale
		for identifier, content := range stringPairs {
			languageStrings.Strings = append(languageStrings.Strings, IdentifiedString{identifier, content})
		}
		decodedStrings = append(decodedStrings, languageStrings)
	}
	return decodedStrings, langStrings, nil
}

func PortalIndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got request for test portal page")
	t, _ := template.ParseFiles("./views/index.html", "./views/nav.html", "./views/resources.html", "./views/scripts.html")
	module := map[string]interface{}{}
	languageStrings, langStringsJson, readErr := loadLanguageStrings()
	if readErr != nil {
		fmt.Println(readErr)
	} else {
		// For the language selection menu
		module["LanguageStrings"] = languageStrings
		// For the javascript code that applies strings
		module["LanguageStringsAsJSON"] = stringifyLanguages(langStringsJson)
	}
	t.Execute(w, module)
}

func PortalChannelsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got request for test channels page")
	T, _ := i18n.Tfunc(os.Getenv(LANG_ENVVAR), DEFAULT_LANG)
	t, _ := template.ParseFiles("./views/channels.html", "./views/nav.html", "./views/resources.html", "./views/breadcrumbs.html", "./views/scripts.html")
	module, err := InitModuleWithFeeds()
	if err != nil {
		t.Execute(w, nil)
	} else {
		module["Breadcrumbs"] = []PortalPath{
			{"CeNO", "/portal"},
			{T("channel_selector"), "/channels"},
		}
		languageStrings, langStringsJson, readErr := loadLanguageStrings()
		if readErr != nil {
		} else {
			// For the language selection menu
			module["LanguageStrings"] = languageStrings
			// For the javascript code that applies strings
			module["LanguageStringsAsJSON"] = stringifyLanguages(langStringsJson)
		}
		t.Execute(w, module)
	}
}

func PortalArticlesHandler(w http.ResponseWriter, r *http.Request) {
	T, _ := i18n.Tfunc(os.Getenv(LANG_ENVVAR), DEFAULT_LANG)
	t, _ := template.ParseFiles("./views/articles.html", "./views/nav.html", "./views/resources.html", "./views/breadcrumbs.html", "./views/scripts.html")
	pathComponents := strings.Split(r.URL.Path, "/")
	b64FeedUrl := pathComponents[len(pathComponents)-1]
	feedUrlBytes, _ := base64.StdEncoding.DecodeString(b64FeedUrl)
	feedUrl := string(feedUrlBytes)
	module, err := InitModuleWithArticles(feedUrl)
	if err != nil {
		t.Execute(w, nil)
	} else {
		module["PublishedWord"] = T("published_word")
		module["AuthorWord"] = T("authors_word")
		module["Breadcrumbs"] = []PortalPath{
			{"CeNO", "/portal"},
			{T("channel_selector"), "/channels"},
			{module["Title"].(string), r.URL.String()},
		}
		languageStrings, langStringsJson, readErr := loadLanguageStrings()
		if readErr != nil {
		} else {
			// For the language selection menu
			module["LanguageStrings"] = languageStrings
			// For the javascript code that applies strings
			module["LanguageStringsAsJSON"] = stringifyLanguages(langStringsJson)
		}
		t.Execute(w, module)
	}
}
