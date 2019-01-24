package goclient

import (
	"log"

	pb "github.com/ideas2it/ideacrawler/protofiles"
)

type Option func(*CrawlJob)

func SeedURL(vv string) Option {
	return func(args *CrawlJob) {
		args.seedURL = vv
	}
}

func MinDelay(vv int32) Option {
	return func(args *CrawlJob) {
		args.minDelay = vv
	}
}

func MaxDelay(vv int32) Option {
	return func(args *CrawlJob) {
		args.maxDelay = vv
	}
}

func MaxIdleTime(vv int32) Option {
	return func(args *CrawlJob) {
		args.maxIdleTime = vv
	}
}

func NoFollow() Option {
	return func(args *CrawlJob) {
		args.follow = false
	}
}

func CallbackURLRegexp(vv string) Option {
	return func(args *CrawlJob) {
		args.callbackURLRegexp = vv
	}
}

func FollowURLRegexp(vv string) Option {
	return func(args *CrawlJob) {
		args.followURLRegexp = vv
	}
}

func MaxConcurrentRequests(vv int32) Option {
	return func(args *CrawlJob) {
		args.maxConcurrentRequests = vv
	}
}

func Useragent(vv string) Option {
	return func(args *CrawlJob) {
		args.useragent = vv
	}
}

func Impolite() Option {
	return func(args *CrawlJob) {
		args.impolite = true
	}
}

func Depth(vv int32) Option {
	return func(args *CrawlJob) {
		args.depth = vv
	}
}

func UnsafeNormalizeURL() Option {
	return func(args *CrawlJob) {
		args.unsafeNormalizeURL = true
	}
}

func CheckLoginAfterEachPage() Option {
	return func(args *CrawlJob) {
		args.checkLoginAfterEachPage = true
	}
}

func Chrome(uu bool, vv string) Option {
	return func(args *CrawlJob) {
		args.chrome = uu
		args.chromeBinary = vv
	}
}

func DomLoadTime(vv int32) Option {
	return func(args *CrawlJob) {
		args.domLoadTime = vv
	}
}

func NetworkIface(vv string) Option {
	return func(args *CrawlJob) {
		args.networkIface = vv
	}
}

func CancelOnDisconnect() Option {
	return func(args *CrawlJob) {
		args.cancelOnDisconnect = true
	}
}

func CheckContent() Option {
	return func(args *CrawlJob) {
		args.checkContent = true
	}
}

func Prefetch() Option {
	return func(args *CrawlJob) {
		args.prefetch = true
	}
}

func CallbackAnchorTextRegexp(vv string) Option {
	return func(args *CrawlJob) {
		args.callbackAnchorTextRegexp = vv
	}
}

func CleanUpFunc(vv func()) Option {
	return func(args *CrawlJob) {
		args.cleanUpFunc = vv
	}
}

func CallbackFunc(vv func(*PageHTML, *CrawlJob)) Option {
	return func(args *CrawlJob) {
		if args.usePageChan == true {
			log.Fatal("Can't use PageChan and CallbackFunc at the same time.")
		}
		args.callback = vv
	}
}

func PageChan(vv chan *pb.PageHTML) Option {
	return func(args *CrawlJob) {
		if args.callback != nil {
			log.Fatal("Can't use PageChan and CallbackFunc at the same time.")
		}
		args.usePageChan = true
		args.implPageChan = vv
		args.PageChan = vv
	}
}

func AnalyzedURLChan(vv chan *pb.UrlList) Option {
	return func(args *CrawlJob) {
		if args.callback != nil {
			log.Fatal("Can't use PageChan and CallbackFunc at the same time.")
		}
		args.useAnalyzedURLChan = true
		args.implAnalyzedURLChan = vv
		args.AnalyzedURLChan = vv
	}
}

func CallbackSeedURL() Option {
	return func(args *CrawlJob) {
		args.callbackSeedURL = true
	}
}

func Login(loginUrl string, loginPayload, loginParseXpath KVMap, loginSuccessCheck KVMap) Option {
	return func(args *CrawlJob) {
		args.login = true
		args.loginUrl = loginUrl
		for k, v := range loginPayload {
			args.loginPayload = append(args.loginPayload, &pb.KVP{Key: k, Value: v})
		}
		if len(loginParseXpath) > 0 {
			args.loginParseFields = true
		}
		for k, v := range loginParseXpath {
			args.loginParseXpath = append(args.loginParseXpath, &pb.KVP{Key: k, Value: v})
		}

		for k, v := range loginSuccessCheck {
			args.loginSuccessCheck = &pb.KVP{Key: k, Value: v}
		}
	}
}

func LoginChrome(loginUrl string, loginJS string, loginSuccessCheck KVMap) Option {
	return func(args *CrawlJob) {
		args.login = true
		args.loginUrl = loginUrl
		args.loginJS = loginJS
		for k, v := range loginSuccessCheck {
			args.loginSuccessCheck = &pb.KVP{Key: k, Value: v}
		}
	}
}

func CallbackXpathMatch(mdata KVMap) Option {
	return func(args *CrawlJob) {
		for k, v := range mdata {
			args.callbackXpathMatch = append(args.callbackXpathMatch, &pb.KVP{Key: k, Value: v})
		}
	}
}

func CallbackXpathRegexp(mdata KVMap) Option {
	return func(args *CrawlJob) {
		for k, v := range mdata {
			args.callbackXpathRegexp = append(args.callbackXpathRegexp, &pb.KVP{Key: k, Value: v})
		}
	}
}
