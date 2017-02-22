package main

import (
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
)

var gumtreePage = "https://www.gumtree.pl"

func main() {

	mainCatLinksChan := make(chan string)
	allCategoriesLinksChan := make(chan []string)
	result := make(chan string)

	runtime.GOMAXPROCS(runtime.NumCPU())

	mainCategoriesLinks := getLinksFromMainPage()

	for i := 0; i < 1000; i++ {
		go getAllPostLinksForCategory(allCategoriesLinksChan, result)
	}

	go getAllCategoriesLinks(mainCatLinksChan, allCategoriesLinksChan)

	go produce(mainCatLinksChan, mainCategoriesLinks)

	consume(result)
}

func produce(c chan string, links []string) {
	for _, item := range links {
		//	fmt.Println("Adding main link: ", item)
		c <- item
	}

	close(c)
}

func consume(c chan string) {
	for {
		data, more := <-c
		if more {
			fmt.Println(data)

		} else {
			return
		}
	}
}

func getLinksFromMainPage() []string {

	root := parse(gumtreePage)

	allLinks := func(n *html.Node) bool {
		links := scrape.Attr(n, "href")
		return strings.HasPrefix(links, "/s-")
	}
	articles := scrape.FindAll(root, allLinks)

	var result []string
	for _, article := range articles {
		result = append(result, gumtreePage+scrape.Attr(article, "href"))
	}
	return result
}

func getAllCategoriesLinks(mainCatLinksChannel chan string, allCategoriesLinksChannel chan []string) {

	pageRegex := regexp.MustCompile("page-(\\d+)")

	for {

		link, more := <-mainCatLinksChannel

		if more {

			allLinks := []string{}

			root := parse(link)
			lastPageNumberEl, ok := scrape.Find(root, scrape.ByClass("last"))
			if ok {
				text := scrape.Attr(lastPageNumberEl, "href")
				lastPageNumber := pageRegex.FindStringSubmatch(text)[1]
				parsed, _ := strconv.Atoi(lastPageNumber)

				for index := 1; index < parsed; index++ {
					newEl := gumtreePage + strings.Replace(text, lastPageNumber, strconv.Itoa(index), 2)
					allLinks = append(allLinks, newEl)
					//	fmt.Println(newEl)
				}
				allCategoriesLinksChannel <- allLinks

			}
		} else {
			return
		}
	}
}

func getAllPostLinksForCategory(allCategoriesLinksChannel chan []string, result chan string) {

	for {

		links, more := <-allCategoriesLinksChannel
		if more {
			for _, link := range links {
				root := parse(link)

				allLinks := scrape.FindAll(root, scrape.ByClass("href-link"))
				//fmt.Println(allLinks)
				for _, link := range allLinks {
					href := scrape.Attr(link, "href")

					newEl := gumtreePage + href

					//fmt.Println(href)
					result <- newEl

				}
			}

		} else {
			return
		}

	}
}

func parse(link string) *html.Node {
	resp, err := http.Get(link)
	if err != nil {
		return parse(link)
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		return parse(link)
	}
	defer resp.Body.Close()

	return root
}
