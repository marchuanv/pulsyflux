package factory

import (
	"net/url"
)

const (
	URLFactory Factory[*url.URL] = "4fa8fa98-dee0-4f3a-bdf8-a3eed61b42f9"
)

func init() {
	URLFactory.Ctor(func(args []*Arg) *url.URL {
		isString, value := ArgValue[string](args[0])
		if isString {
			_url, err := url.Parse(value)
			if err != nil {
				panic(err)
			}
			return _url
		}
		return nil
	})
}
