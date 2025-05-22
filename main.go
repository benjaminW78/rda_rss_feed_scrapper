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

const blogURL = "https://ruben.care/blog"

func main() {
	http.HandleFunc("/rss.xml", serveRSS)
	fmt.Println("Server started on :8080 (GET http://localhost:8080/rss.xml)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveRSS(w http.ResponseWriter, r *http.Request) {
	articles, err := fetchArticles()
	if err != nil {
		http.Error(w, "Unable to fetch articles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Le blog Ruben",
		Link:        &feeds.Link{Href: blogURL},
		Description: "Actualités et conseils pour les pets parents.",
		Created:     now,
	}

	for _, a := range articles {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       a.Title,
			Link:        &feeds.Link{Href: a.URL},
			Description: makeDescription(a.Category, a.DateTxt, a.ReadingTime),
			Created:     a.Date,
			Enclosure:   &feeds.Enclosure{Url: a.Image, Type: "image/png"},
		})
	}

	rss, err := feed.ToRss()
	if err != nil {
		http.Error(w, "Unable to create RSS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(rss))
}

// Article struct
type Article struct {
	URL         string
	Title       string
	Image       string
	Category    string
	DateTxt     string
	ReadingTime string
	Description string
	Date        time.Time
}

// fetchArticles scraps the blog for articles without Framer classes
func fetchArticles() ([]Article, error) {
	resp, err := http.Get(blogURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	articles := []Article{}
	seen := map[string]bool{}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, "blogs-articles") {
			return
		}

		// Build absolute URL
		fullURL := href
		if strings.HasPrefix(href, "./") {
			fullURL = "https://ruben.care/" + strings.TrimPrefix(href, "./")
		} else if strings.HasPrefix(href, "/") {
			fullURL = "https://ruben.care" + href
		} else if !strings.HasPrefix(href, "http") {
			fullURL = "https://ruben.care/" + href
		}
		if seen[fullURL] {
			return
		}
		seen[fullURL] = true

		// On ne garde que les liens qui ont un h3 (titre) et une image à l'intérieur
		title := s.Find("h3").Text()
		imgURL, _ := s.Find("img").Attr("src")
		if title == "" || imgURL == "" {
			return
		}

		// Optionnel : description courte avec la catégorie, la date, et le temps de lecture
		category := ""
		dateTxt := ""
		readingTime := ""
		// Ex: <p>Garde d'animaux</p> <p>|</p> <p>19 mai 2025</p> <p>|</p> <p>6 min</p>
		s.Find("p").Each(func(i int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text == "|" {
				return
			}
			if strings.Contains(text, "min") {
				readingTime = text
			} else if strings.Contains(text, "202") { // année
				dateTxt = text
			} else if category == "" {
				category = text
			}
		})

		// Parse la date (ex: "19 mai 2025"), sinon mets aujourd'hui
		dateParsed := time.Now()
		if dateTxt != "" {
			dateParsed, _ = parseFrenchDate(dateTxt)
		}

		articles = append(articles, Article{
			URL:         fullURL,
			Title:       title,
			Image:       imgURL,
			Category:    category,
			DateTxt:     dateTxt,
			ReadingTime: readingTime,
			Date:        dateParsed,
		})
	})

	return articles, nil
}

func makeDescription(category, dateTxt, readingTime string) string {
	// Format : "Garde d'animaux — 19 mai 2025 (6 min)"
	if category != "" && dateTxt != "" && readingTime != "" {
		return fmt.Sprintf("%s — %s (%s)", category, dateTxt, readingTime)
	}
	if category != "" && dateTxt != "" {
		return fmt.Sprintf("%s — %s", category, dateTxt)
	}
	if category != "" {
		return category
	}
	return dateTxt
}

// parseFrenchDate essaye de parser une date style "19 mai 2025"
func parseFrenchDate(dateStr string) (time.Time, error) {
	months := map[string]string{
		"janvier": "01", "février": "02", "mars": "03", "avril": "04",
		"mai": "05", "juin": "06", "juillet": "07", "août": "08",
		"septembre": "09", "octobre": "10", "novembre": "11", "décembre": "12",
	}
	parts := strings.Split(dateStr, " ")
	if len(parts) < 3 {
		return time.Now(), fmt.Errorf("cannot parse date: %s", dateStr)
	}
	day := parts[0]
	month := months[strings.ToLower(parts[1])]
	year := parts[2]
	iso := fmt.Sprintf("%s-%s-%s", year, month, day)
	return time.Parse("2006-01-02", iso)
}
