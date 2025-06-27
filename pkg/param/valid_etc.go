package param

import "regexp"

func init() {
	RegisterValidFunc(ID, ValidId)
	RegisterValidFunc(TITLE, ValidTitle)
}

var (
	regId    = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	regTitle = regexp.MustCompile(`^[가-힣a-zA-Z0-9 .,!?\[\]\(\)_\-@&/|$%*+~^:={}'"]+$`)
)

func ValidId(value string) (bool, error) {
	return regId.MatchString(value), nil
}

func ValidTitle(value string) (bool, error) {
	return regTitle.MatchString(value), nil
}
