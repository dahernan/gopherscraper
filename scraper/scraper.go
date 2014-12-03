package scraper

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dahernan/gopherscraper/model"
)

const (
	SelectorTypeList   = "list"
	SelectorTypeDetail = "detail"
	SelectorIdFromUrl  = "IdFromUrl"
	SelectorIdFromCSS  = "IdFromCSS"
	SelectorIdFromLink = "IdFromLink"

	bufferItemsSize = 100 // not sure if is good idea to make it configurable
)

var (
	ErrNoBaseSelector  = fmt.Errorf("No Base selector for the scraping")
	ErrInvalidSelector = fmt.Errorf("InvalidSelector it can not be Recursive for a Detail type")

	defaultHttpClient *http.Client
	defaultUserAgent  string

	// limit the number of concurrent connections
	semaphoreMaxConnections chan struct{}
)

func init() {
	// default config
	UseHttpClientWithTimeout(5 * time.Second)
	UseUserAgent("gopherscraper")
	UseMaxConnections(1000)
}

// GoQuery Seletor
type ScrapSelector struct {
	Url           string `json:"url"`
	Base          string `json:"base"`
	Stype         string `json:"stype,omitempty"`
	Recursive     bool   `json:"recursive,omitempty"`
	PageParam     string `json:"pageParam"`
	PageStart     int    `json:"pageStart"`
	PageIncr      int    `json:"pageIncr"`
	PageLimit     int    `json:"pageLimit"`
	IdFrom        string
	IdPrefix      string
	IdExtractor   ExtractId `json:"IdExtractor"`
	Id            Selector  `json:"id"`
	Link          Selector  `json:"link",omitempty`
	LinkPathLimit int       `json:"linkPathLimit",omitempty`
	Image         Selector  `json:"image,omitempty"`
	Title         Selector  `json:"title,omitempty"`
	Description   Selector  `json:"description,omitempty"`
	Price         Selector  `json:"price,omitempty"`
	Categories    Selector  `json:"categories,omitempty"`
	Stars         Selector  `json:"starts,omitempty"`

	// comma separated fixed tags
	ScrapTags string `json:"scrapTags,omitempty"`
}

type Selector struct {
	Exp  string `json:"exp"`
	Attr string `json:"attr,omitempty"`
}

type ExtractId struct {
	UrlPathIndex int    `json:"urlPathIndex"`
	SplitString  string `json:"splitString"`
	SplitIndex   int    `json:"splitIndex"`
}

type ItemResult struct {
	JobId string
	Item  model.Item
	Err   error
}

// Scrap a website looking for items based on the CSS selector
// and returns the jobId, a channel with the Items scrapperd, or an error
type ScrapperItems interface {
	Scrap(selector ScrapSelector) (string, chan ItemResult, error)
}

type DefaultScrapper struct {
}

func NewScrapper() ScrapperItems {
	return DefaultScrapper{}
}

// DefaultScrapper Scraps a Web looking for items, if the selector has multiple pages
// it does the scrap in all the pages concurrently
func (d DefaultScrapper) Scrap(selector ScrapSelector) (string, chan ItemResult, error) {
	wg := &sync.WaitGroup{}
	err := validateSelector(selector)
	if err != nil {
		return "", nil, err
	}

	items := make(chan ItemResult, bufferItemsSize)

	jobId := "D" + GenerateStringKey(selector)
	log.Printf("INFO: Scrap [%s] started\n", jobId)
	data := NewRedisScrapdata()
	data.StartJob(jobId, selector)

	pages := paginatedUrlSelector(selector)

	wg.Add(len(pages))
	for i, _ := range pages {
		go doScrapFromUrl(jobId, pages[i], items, wg)
	}

	go closeItemsChannel(jobId, items, wg)

	return jobId, items, err
}

func doScrapFromUrl(jobId string, s ScrapSelector, items chan ItemResult, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("INFO: Scrap [%s] GET from %s ", jobId, s.Url)

	doc, err := fromUrl(s)
	if err != nil {
		log.Printf("ERROR [%s] Scrapping %v with message %v", jobId, s.Url, err.Error())
		return
	}
	DocumentScrap(jobId, s, doc, items)
	log.Printf("INFO: Scrap [%s] FINISH SCRAP Request from %s ", jobId, s.Url)

}

func closeItemsChannel(jobId string, items chan ItemResult, wg *sync.WaitGroup) {
	wg.Wait()
	close(items)
	log.Printf("INFO: Scrap [%s] finished\n", jobId)
	data := NewRedisScrapdata()
	data.FinishJob(jobId)
}

// You can use a custom http.Client calling this function before doing any scrapping
func UseHttpClient(client *http.Client) {
	defaultHttpClient = client
}

// You can set the timeout for the standard http.Client before doing any scrapping
func UseHttpClientWithTimeout(timeout time.Duration) {
	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	transport := http.Transport{
		Dial: dialTimeout,
	}
	client := http.Client{
		Transport: &transport,
	}

	defaultHttpClient = &client
}

// set a custom Agent for the scraper
func UseUserAgent(ua string) {
	defaultUserAgent = ua
}

func httpClient() *http.Client {
	return defaultHttpClient
}

func fromUrl(selector ScrapSelector) (*goquery.Document, error) {
	lockLimitConnections()
	defer unlockLimitConnections()

	req, err := http.NewRequest("GET", selector.Url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", defaultUserAgent)

	res, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	return goquery.NewDocumentFromResponse(res)
}

// acts as a lock to limit the number of concurrent connections
func lockLimitConnections() {
	semaphoreMaxConnections <- struct{}{}
}
func unlockLimitConnections() {
	<-semaphoreMaxConnections
}

// limit the number of maximun http conections used
func UseMaxConnections(max int) {
	semaphoreMaxConnections = make(chan struct{}, max)
}

func validateSelector(selector ScrapSelector) error {
	if selector.Base == "" {
		return ErrNoBaseSelector
	}

	if (selector.Stype == SelectorTypeDetail) && selector.Recursive {
		return ErrInvalidSelector
	}

	return nil

}

func paginatedUrlSelector(selector ScrapSelector) []ScrapSelector {
	var pages []ScrapSelector
	if selector.PageParam == "" {
		return []ScrapSelector{selector}
	}

	for i := selector.PageStart; i < selector.PageLimit; i = i + selector.PageIncr {
		dup := selector
		// change page parameter and re-encode
		url, _ := neturl.Parse(dup.Url)
		q := url.Query()
		q.Set(selector.PageParam, strconv.Itoa(i))
		url.RawQuery = q.Encode()
		dup.Url = url.String()

		pages = append(pages, dup)

	}

	return pages

}

func NewRecursiveScrapper() ScrapperItems {
	return RecursiveScrapper{
		baseScrapper: NewScrapper(),
	}
}

type RecursiveScrapper struct {
	baseScrapper ScrapperItems
}

// Recursive Scrapper can dig into detail pages, and do a recursive scrap
// the normal flow is
// 1) Scrap a List page -> multiple items in that page
// 2) For each item follow the link
// 3) Get the detail Selector from Redis related with the item scraped
// 4) Scrap the detail page
func (rs RecursiveScrapper) Scrap(selector ScrapSelector) (string, chan ItemResult, error) {
	wg := &sync.WaitGroup{}

	selector, err := rs.selectorFromRedis(selector)
	if err != nil {
		return "", nil, err
	}

	err = validateSelector(selector)
	if err != nil {
		return "", nil, err
	}

	baseJobId, itemsIn, err := rs.baseScrapper.Scrap(selector)
	if err != nil {
		return baseJobId, nil, err
	}

	if !selector.Recursive {
		return baseJobId, itemsIn, err
	}

	itemsOut := make(chan ItemResult, bufferItemsSize)

	recJobId := "R" + GenerateStringKey(selector)
	log.Printf("INFO: Scrap [%v] Recursive started\n", recJobId)
	data := NewRedisScrapdata()
	data.StartJob(recJobId, selector)

	wg.Add(1)
	go rs.ScrapAllRecursiveItems(recJobId, selector, itemsIn, itemsOut, wg)
	go closeItemsChannel(recJobId, itemsOut, wg)

	return recJobId, itemsOut, err

}

func (rs RecursiveScrapper) ScrapAllRecursiveItems(jobId string, selector ScrapSelector, inItems chan ItemResult, outItems chan ItemResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for it := range inItems {
		wg.Add(1)
		go rs.scrapItemRecursive(jobId, it, selector, outItems, wg)
	}

}

func (rs RecursiveScrapper) scrapItemRecursive(jobId string, it ItemResult, selector ScrapSelector, itemsChan chan ItemResult, wg *sync.WaitGroup) {
	defer wg.Done()
	rselector, err := rs.recursiveSelector(it, selector)
	if err != nil {
		log.Println("ERROR: RecursiveScrapper:scrapItemRecursive there is a problem with the Selector", err.Error())
		return
	}

	_, itemsRec, err := rs.Scrap(rselector)
	if err != nil {
		log.Println("ERROR: RecursiveScrapper:Scrap there is a problem with the Selector", err.Error())
		return
	}

	for i := range itemsRec {
		// overwrite the jobid to reflect the parent job
		i.JobId = jobId
		itemsChan <- i
	}

}

func (rs RecursiveScrapper) selectorFromRedis(s ScrapSelector) (ScrapSelector, error) {
	// if selector not empty just use it
	if s.Base != "" {
		return s, nil
	}

	redisData := NewRedisScrapdata()

	if s.Stype != "" {
		return redisData.Selector(s.Url, s.Stype)
	}

	rselector, err := redisData.Selector(s.Url, SelectorTypeList)
	if err == ErrSelectorNotFound {
		return redisData.Selector(s.Url, SelectorTypeDetail)
	}
	return rselector, err
}

func (rs RecursiveScrapper) recursiveSelector(it ItemResult, selector ScrapSelector) (ScrapSelector, error) {
	var rselector ScrapSelector

	redisData := NewRedisScrapdata()

	if it.Err != nil {
		log.Println("ERROR: RecursiveScrapper:Scrap the item for scrap recursive has an error", it.Err.Error())
		return rselector, it.Err
	}

	rselector, err := redisData.Selector(it.Item.Link, SelectorTypeDetail)
	if err != nil {
		log.Printf("ERROR: RecursiveScrapper:Scrap is not possible to get the Selector to scrap the recursive Item, %v,  Link: %v, Type: %v\n", err.Error(), it.Item.Link, SelectorTypeDetail)
		return rselector, ErrSelectorNotFound
	}

	// make sure data in the selector is right
	rselector.Url = it.Item.Link
	rselector.Recursive = false

	return rselector, nil
}

// Scrapper from reader useful for testing
type FromReaderScrapper struct {
	reader *io.Reader
}

func ScrapperFromReader(r io.Reader) ScrapperItems {
	return FromReaderScrapper{&r}
}

func (s FromReaderScrapper) Scrap(selector ScrapSelector) (string, chan ItemResult, error) {
	var wg sync.WaitGroup

	err := validateSelector(selector)
	if err != nil {
		return "", nil, err
	}

	jobId := "READER" + GenerateStringKey(selector)
	log.Printf("INFO: Scrap [%v] from Reader started\n", jobId)

	items := make(chan ItemResult, bufferItemsSize)
	wg.Add(1)

	go func() {
		doc, err := goquery.NewDocumentFromReader(*s.reader)
		if err != nil {
			log.Println("ERROR Scrapping ", selector.Url, " with message", err.Error())
			return
		}
		DocumentScrap(jobId, selector, doc, items)
		wg.Done()
	}()

	closeItemsChannel(jobId, items, &wg)

	return jobId, items, nil
}

// Scrapping logic from the document
func DocumentScrap(jobId string, selector ScrapSelector, doc *goquery.Document, items chan ItemResult) {
	rdata := NewRedisScrapdata()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERROR: DocumentScrap Panic applying selectors: '%v'", r)
			rdata.ScrapLogWrite("ERROR: bad CSS Selector, please review the syntax")
		}
	}()

	sel := doc.Find(selector.Base)
	for i := range sel.Nodes {
		s := sel.Eq(i)
		var err error
		item := model.Item{}
		item.ScrapUrl = selector.Url
		item.ScrapTags = selector.ScrapTags

		item.Link = SanitizeURL(item.ScrapUrl, extractText(s, selector.Link), selector.LinkPathLimit)
		item.Id, err = extractId(s, selector, item.Link)
		item.Image = SanitizeURL(item.ScrapUrl, extractText(s, selector.Image), 0)
		item.Title = extractText(s, selector.Title)
		item.Description = extractText(s, selector.Description)
		item.Price = extractFloat(s, selector.Price)
		item.Currency = extractCurrency(s, selector.Price)
		item.Stars = extractFloat(s, selector.Stars)
		item.Categories = extractText(s, selector.Categories)

		item.LastScrap = time.Now().Format(time.RFC3339)

		items <- ItemResult{
			JobId: jobId,
			Item:  item,
			Err:   err,
		}
	}

}

func extractId(s *goquery.Selection, selector ScrapSelector, link string) (string, error) {
	id, err := extractNakedId(s, selector, link)
	if err != nil {
		return "", err
	}
	return selector.IdPrefix + id, nil
}

func extractNakedId(s *goquery.Selection, selector ScrapSelector, link string) (string, error) {
	if selector.IdFrom == SelectorIdFromUrl {
		return ExtractIdFromURL(selector.Url, selector.IdExtractor.UrlPathIndex, selector.IdExtractor.SplitString, selector.IdExtractor.SplitIndex)
	}
	if selector.IdFrom == SelectorIdFromLink {
		return ExtractIdFromURL(link, selector.IdExtractor.UrlPathIndex, selector.IdExtractor.SplitString, selector.IdExtractor.SplitIndex)
	}

	return extractText(s, selector.Id), nil
}

func SnippetBase(selector ScrapSelector) (string, error) {
	doc, err := fromUrl(selector)
	if err != nil {
		return "", err
	}
	return baseSelectorSnip(selector, doc)

}

func SanitizeURL(scrapUrl, url string, linkLimit int) string {
	if url == "" {
		return scrapUrl
	}

	surl, err := neturl.Parse(scrapUrl)
	if err != nil {
		return url
	}

	purl, err := neturl.Parse(strings.Trim(url, " \t\r\n"))
	if err != nil {
		return url
	}

	// delete sid
	q := purl.Query()
	q.Del("sid")
	purl.RawQuery = q.Encode()
	purl.Path = strings.Replace(purl.Path, "sid=", "", -1)

	if linkLimit != 0 {
		path_parts := strings.Split(purl.Path, "/")
		index := len(path_parts) - linkLimit
		purl.Path = strings.Join(path_parts[0:index], "/")
	}

	if purl.IsAbs() {
		return purl.String()
	}

	// set absolute
	purl.Scheme = surl.Scheme
	purl.Host = surl.Host

	return purl.String()

}

func baseSelectorSnip(selector ScrapSelector, doc *goquery.Document) (string, error) {
	if selector.Base == "" {
		return "", ErrNoBaseSelector
	}
	return doc.Find(selector.Base).Html()
}

func extractText(s *goquery.Selection, exp Selector) string {
	if exp.Exp == "" {
		return ""
	}
	if exp.Attr == "" {
		return s.Find(exp.Exp).Text()
	}
	value, ok := s.Find(exp.Exp).Attr(exp.Attr)
	if !ok {
		return ""
	}
	return value

}

func extractFloat(s *goquery.Selection, exp Selector) float64 {
	var value string
	if exp.Exp == "" {
		return 0
	}
	if exp.Attr == "" {
		value = s.Find(exp.Exp).Text()
		return extractFloatFromString(value)
	}
	value, ok := s.Find(exp.Exp).Attr(exp.Attr)
	if !ok {
		return 0
	}
	return extractFloatFromString(value)
}

func extractCurrency(s *goquery.Selection, exp Selector) string {
	return removeNumbers(extractText(s, exp))
}

func removeNumbers(value string) string {
	delete := func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return ' '
		case r == '.':
			return ' '
		default:
			return r
		}
	}
	s := strings.Map(delete, value)
	s = strings.Replace(s, " ", "", -1)
	return s

}

func extractFloatFromString(value string) float64 {

	delete := func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return r
		case r == '.':
			return r
		default:
			return ' '
		}

	}
	s := strings.Map(delete, value)
	s = strings.Replace(s, " ", "", -1)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func ExtractIdFromURL(u string, pathIndex int, split string, splitIndex int) (string, error) {
	parsed, err := neturl.Parse(u)
	if err != nil {
		return "", err
	}

	path_parts := strings.Split(parsed.Path, "/")

	if pathIndex < 0 {
		pathIndex = len(path_parts) + pathIndex
	}

	if !(pathIndex >= 0 && pathIndex < len(path_parts)) {
		return "", fmt.Errorf("The index [%v] to split the path [%s] of the url is out of bounds: %v", pathIndex, parsed.Path, prettyPrint(path_parts))
	}

	group := path_parts[pathIndex]
	if split == "" {
		return group, nil
	}

	splitted := strings.Split(group, split)

	if splitIndex < 0 {
		splitIndex = len(splitted) + splitIndex
	}

	if !(splitIndex >= 0 && splitIndex < len(splitted)) {
		return "", fmt.Errorf("The index [%v] to split [%s] is out of bounds: %s", splitIndex, group, prettyPrint(splitted))
	}
	id := splitted[splitIndex]
	cleaned := strings.Replace(id, ".html", "", -1)
	cleaned = strings.Replace(cleaned, ".htm", "", -1)
	return cleaned, nil

}

func GenerateStringKey(selector ScrapSelector) string {
	// TODO review this, if the number of urls is high it could have hash collision
	baseHash := selector.Url

	hash := fnv.New32()
	_, err := io.WriteString(hash, baseHash)
	if err != nil {
		log.Println(err)
	}
	return fmt.Sprintf("%v", hash.Sum32())

}

func prettyPrint(s []string) string {
	var buffer bytes.Buffer

	buffer.WriteString("{")
	for i, v := range s {
		buffer.WriteString(strconv.Itoa(i))
		buffer.WriteString(":")
		buffer.WriteString(v)
		buffer.WriteString(" ")
	}
	buffer.WriteString("}")
	return buffer.String()
}
