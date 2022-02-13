package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func scrapeWords(letter string, index int, wordLen int, c chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	url := fmt.Sprintf("https://www.dictionary.com/list/%s/%d", letter, index)
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	regexString := fmt.Sprintf("^([a-zA-Z]{%d})$", wordLen)
	regex := regexp.MustCompile(regexString)

	doc.Find("ul[data-testid=\"list-az-results\"] li a").Each(func(i int, s *goquery.Selection) {
		text := s.Text()

		if match := regex.MatchString(text); match {
			c <- strings.ToLower(text)
		}
	})

	fmt.Printf("done scraping for letter %s, page %d\n", letter, index)
}

func getWordsForLetter(letter string, wordLen int, c chan string, wg *sync.WaitGroup) {
	fmt.Printf("Fetching for letter %s\n", letter)
	defer wg.Done()
	url := fmt.Sprintf("https://www.dictionary.com/list/%s", letter)

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	goLastBtn := doc.Find("ul[data-testid=\"list-az-results\"]+div ol li a").Last()
	lastHref, exists := goLastBtn.Attr("href")
	lastValue := 1
	if exists {
		lastValue, _ = strconv.Atoi(strings.Replace(lastHref, "/list/"+letter+"/", "", -1))
	}

	var childWg sync.WaitGroup
	for i := 1; i <= lastValue; i++ {
		childWg.Add(1)
		go scrapeWords(letter, i, wordLen, c, &childWg)
	}

	childWg.Wait()
}

func main() {
	c := make(chan string)
	var wg sync.WaitGroup
	for letter := 'a'; letter <= 'z'; letter++ {
		wg.Add(1)
		go getWordsForLetter(string(letter), 5, c, &wg)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	wordMap := make(map[string]bool)
	for word := range c {
		wordMap[word] = true
	}

	words := make([]string, 0, len(wordMap))
	for word := range wordMap {
		words = append(words, word)
	}

	fmt.Println(len(words))

	jsonContent, _ := json.Marshal(words)

	_ = ioutil.WriteFile("5-letter-words.json", jsonContent, 0777)

}
