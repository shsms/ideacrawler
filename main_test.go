package main

import (
	"log"
	"math/rand"
	"regexp"
	"testing"
	"time"

	gc "github.com/ideas2it/ideacrawler/goclient"
)

var serverRunning bool

func startTestServer() string {
	if serverRunning == true {
		return "2345"
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rand.Seed(time.Now().UTC().UnixNano())
	cliParams.ListenAddress = "127.0.0.1"
	cliParams.ListenPort = "2345"
	go startServer()
	serverRunning = true
	time.Sleep(1 * time.Second)
	return "2345"
}

type TestData struct {
	name              string
	seedURL           string
	chrome            bool
	maxIdleTime       int32
	depth             int32
	followUrlRegexp   string
	callbackUrlRegexp string
	respStatusCode    int32
	respBodyRegexp    string
	respBodyMinLength int
	respPageCount     int
}

func TestSinglePage(t *testing.T) {
	port := startTestServer()

	var dataPoints = []TestData{
		TestData{
			name:              "chrome200",
			seedURL:           "http://quotes.toscrape.com/tag/simile/",
			chrome:            true,
			respStatusCode:    200,
			respBodyRegexp:    "Life.is.like.riding.a.bicycle",
			respBodyMinLength: 100,
		},
		TestData{
			name:              "chrome404",
			seedURL:           "http://quotes.toscrape.com/tag/simile/1",
			chrome:            true,
			respStatusCode:    404,
			respBodyRegexp:    "",
			respBodyMinLength: 0,
		},
		TestData{
			name:              "simple200",
			seedURL:           "http://quotes.toscrape.com/tag/simile/",
			chrome:            false,
			respStatusCode:    200,
			respBodyRegexp:    "Life.is.like.riding.a.bicycle",
			respBodyMinLength: 100,
		},
		TestData{
			name:              "simple404",
			seedURL:           "http://quotes.toscrape.com/tag/simile/1",
			chrome:            false,
			respStatusCode:    404,
			respBodyRegexp:    "",
			respBodyMinLength: 0,
		},
	}

	for _, dp := range dataPoints {
		t.Run(dp.name, func(t *testing.T) {
			z := gc.NewCrawlJob("127.0.0.1", port)

			z.SetPageChan(gc.NewPageChan())
			z.SeedURL = dp.seedURL
			z.Chrome = dp.chrome
			z.ChromeBinary = "/usr/bin/chromium"
			z.Follow = false

			z.Start()

			ph := <-z.PageChan

			if ph.Httpstatuscode != dp.respStatusCode {
				t.Errorf("expected status code: %d, got %d.", dp.respStatusCode, ph.Httpstatuscode)
			}
			if dp.respBodyMinLength > 0 && len(string(ph.Content)) < dp.respBodyMinLength {
				t.Errorf("expected respBodyMinLength of %d, got %d.", dp.respBodyMinLength, len(string(ph.Content)))
			}
			if dp.respBodyRegexp != "" {
				re := regexp.MustCompile(dp.respBodyRegexp)
				if re.Match(ph.Content) == false {
					t.Errorf("regexp match failed for: %s", dp.respBodyRegexp)
				}
			}
			z.Stop()
		})
	}
}

func TestMultiPage(t *testing.T) {
	port := startTestServer()

	var dataPoints = []TestData{
		TestData{
			name:              "depth1",
			seedURL:           "http://quotes.toscrape.com",
			followUrlRegexp:   "com$|author",
			callbackUrlRegexp: "author",
			chrome:            false,
			respStatusCode:    200,
			respBodyMinLength: 100,
			depth:             1,
			maxIdleTime:       10,
			respPageCount:     8,
		},
	}

	for _, dp := range dataPoints {
		t.Run(dp.name, func(t *testing.T) {
			z := gc.NewCrawlJob("127.0.0.1", port)

			z.SetPageChan(gc.NewPageChan())
			z.SeedURL = dp.seedURL
			z.Chrome = dp.chrome
			z.ChromeBinary = "/usr/bin/chromium"
			z.Follow = true
			z.Depth = dp.depth
			z.MinDelay = 1
			z.MaxIdleTime = dp.maxIdleTime
			z.CallbackUrlRegexp = dp.callbackUrlRegexp
			z.FollowUrlRegexp = dp.followUrlRegexp

			z.Start()
			pageCount := 0
			for z.IsAlive() == true { // TODO:  or add timeout.
				select {
				case ph := <-z.PageChan:
					if ph.Httpstatuscode != dp.respStatusCode {
						t.Errorf("expected status code: %d, got %d.", dp.respStatusCode, ph.Httpstatuscode)
					}
					pageCount++
				default:
					time.Sleep(1 * time.Second)
				}
			}
			if pageCount != dp.respPageCount {
				t.Errorf("expected %d response pages, got %d", dp.respPageCount, pageCount)
			}
		})
	}
}
