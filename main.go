package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
)

func main() {
	http.HandleFunc("/rss", rssHandler)
	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func rssHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := "https://ruben.care/blog"
	resp, err := http.Get(baseURL)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Failed to fetch source blog", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse HTML", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Le blog Ruben",
		Link:        &feeds.Link{Href: baseURL},
		Description: "Actualités et conseils pour les pets parents.",
		Created:     now,
	}

	var items []*feeds.Item

	doc.Find("a.framer-9x2ihz.framer-1lgb0lc").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.HasPrefix(href, "./") {
			href = "https://ruben.care/blogs-articles/" + strings.TrimPrefix(href, "./blogs-articles/")
		} else if strings.HasPrefix(href, "/") {
			href = "https://ruben.care" + href
		}
		imgSrc, _ := s.Find("img").Attr("src")
		if !strings.HasPrefix(imgSrc, "http") {
			imgSrc = "https://ruben.care" + imgSrc
		}
		meta := s.Find(".framer-lyerhg p, .framer-yb5v8v p, .framer-12ea1bz p")
		category := meta.Eq(0).Text()
		date := meta.Eq(1).Text()
		duration := meta.Eq(2).Text()
		title := s.Find("h3.framer-text").First().Text()

		createdTime := parseFrenchDate(date)

		description := fmt.Sprintf(`<img src="%s" style="max-width:300px"/><br><b>%s</b><br><i>%s | %s</i>`, imgSrc, category, date, duration)

		items = append(items, &feeds.Item{
			Title:       title,
			Link:        &feeds.Link{Href: href},
			Description: description,
			Created:     createdTime,
		})
	})

	feed.Items = items

	rss, err := feed.ToRss()
	if err != nil {
		http.Error(w, "Failed to create RSS", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(rss))
}

func parseFrenchDate(dateStr string) time.Time {
	mois := map[string]string{
		"janvier": "01", "février": "02", "mars": "03", "avril": "04",
		"mai": "05", "juin": "06", "juillet": "07", "août": "08",
		"septembre": "09", "octobre": "10", "novembre": "11", "décembre": "12",
	}
	parts := strings.Fields(dateStr)
	if len(parts) == 3 {
		day := parts[0]
		month := mois[strings.ToLower(parts[1])]
		year := parts[2]
		dateFmt := fmt.Sprintf("%s-%s-%s", year, month, day)
		t, _ := time.Parse("2006-01-02", dateFmt)
		return t
	}
	return time.Now()
}
