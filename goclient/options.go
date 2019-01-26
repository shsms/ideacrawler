package goclient

import (
	"log"

	pb "github.com/ideas2it/ideacrawler/protofiles"
)

type Option func(*CrawlJob)

func SeedURL(vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.SeedUrl = vv
	}
}

func MinDelay(vv int32) Option {
	return func(args *CrawlJob) {
		args.dopt.MinDelay = vv
	}
}

func MaxDelay(vv int32) Option {
	return func(args *CrawlJob) {
		args.dopt.MaxDelay = vv
	}
}

func MaxIdleTime(vv int32) Option {
	return func(args *CrawlJob) {
		args.dopt.MaxIdleTime = vv
	}
}

func NoFollow() Option {
	return func(args *CrawlJob) {
		args.dopt.NoFollow = true
	}
}

func CallbackURLRegexp(vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.CallbackUrlRegexp = vv
	}
}

func FollowURLRegexp(vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.FollowUrlRegexp = vv
	}
}

func MaxConcurrentRequests(vv int32) Option {
	return func(args *CrawlJob) {
		args.dopt.MaxConcurrentRequests = vv
	}
}

func Useragent(vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.Useragent = vv
	}
}

func Impolite() Option {
	return func(args *CrawlJob) {
		args.dopt.Impolite = true
	}
}

func Depth(vv int32) Option {
	return func(args *CrawlJob) {
		args.dopt.Depth = vv
	}
}

func UnsafeNormalizeURL() Option {
	return func(args *CrawlJob) {
		args.dopt.UnsafeNormalizeURL = true
	}
}

func CheckLoginAfterEachPage() Option {
	return func(args *CrawlJob) {
		args.dopt.CheckLoginAfterEachPage = true
	}
}

func Chrome(uu bool, vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.Chrome = uu
		args.dopt.ChromeBinary = vv
	}
}

func DomLoadTime(vv int32) Option {
	return func(args *CrawlJob) {
		args.dopt.DomLoadTime = vv
	}
}

func NetworkIface(vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.NetworkIface = vv
	}
}

func CancelOnDisconnect() Option {
	return func(args *CrawlJob) {
		args.dopt.CancelOnDisconnect = true
	}
}

func CheckContent() Option {
	return func(args *CrawlJob) {
		args.dopt.CheckContent = true
	}
}

func Prefetch() Option {
	return func(args *CrawlJob) {
		args.dopt.Prefetch = true
	}
}

func CallbackAnchorTextRegexp(vv string) Option {
	return func(args *CrawlJob) {
		args.dopt.CallbackAnchorTextRegexp = vv
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
		args.dopt.CallbackSeedUrl = true
	}
}

func Login(loginUrl string, loginPayload, loginParseXpath KVMap, loginSuccessCheck KVMap) Option {
	return func(args *CrawlJob) {
		args.dopt.Login = true
		args.dopt.LoginUrl = loginUrl
		for k, v := range loginPayload {
			args.dopt.LoginPayload = append(args.dopt.LoginPayload, &pb.KVP{Key: k, Value: v})
		}
		if len(loginParseXpath) > 0 {
			args.dopt.LoginParseFields = true
		}
		for k, v := range loginParseXpath {
			args.dopt.LoginParseXpath = append(args.dopt.LoginParseXpath, &pb.KVP{Key: k, Value: v})
		}

		for k, v := range loginSuccessCheck {
			args.dopt.LoginSuccessCheck = &pb.KVP{Key: k, Value: v}
		}
	}
}

func LoginChrome(loginUrl string, loginJS string, loginSuccessCheck KVMap) Option {
	return func(args *CrawlJob) {
		args.dopt.Login = true
		args.dopt.LoginUrl = loginUrl
		args.dopt.LoginJS = loginJS
		for k, v := range loginSuccessCheck {
			args.dopt.LoginSuccessCheck = &pb.KVP{Key: k, Value: v}
		}
	}
}

func CallbackXpathMatch(mdata KVMap) Option {
	return func(args *CrawlJob) {
		for k, v := range mdata {
			args.dopt.CallbackXpathMatch = append(args.dopt.CallbackXpathMatch, &pb.KVP{Key: k, Value: v})
		}
	}
}

func CallbackXpathRegexp(mdata KVMap) Option {
	return func(args *CrawlJob) {
		for k, v := range mdata {
			args.dopt.CallbackXpathRegexp = append(args.dopt.CallbackXpathRegexp, &pb.KVP{Key: k, Value: v})
		}
	}
}
