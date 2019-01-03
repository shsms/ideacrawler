package statuscodes

type StatusCode = int32

const (
	LoginSuccess     StatusCode = 1500
	LoginFailed      StatusCode = 1501
	NolongerLoggedIn StatusCode = 1502
	FetchbotError    StatusCode = 1520
	ResponseError    StatusCode = 1530
	Subscription     StatusCode = 1540
	FollowParseError StatusCode = 1550
)

func ToString(sc StatusCode) string {
	switch sc {
	case LoginSuccess:
		return "LoginSuccess"
	case LoginFailed:
		return "LoginFailed"
	case NolongerLoggedIn:
		return "NolongerLoggedIn"
	case FetchbotError:
		return "FetchbotError"
	case ResponseError:
		return "ResponseError"
	case Subscription:
		return "Subscription"
	case FollowParseError:
		return "FollowParseError"
	default:
		return "Unknown"
	}
}
