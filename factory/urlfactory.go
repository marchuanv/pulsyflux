package factory

import (
	"net/url"
)

const (
	urlFactory factory[*url.URL] = "4fa8fa98-dee0-4f3a-bdf8-a3eed61b42f9"
)

func URLFactory() factory[*url.URL] {
	urlFactory.ctor(func(args ...*Arg) *url.URL {
		isString, value := argValue[string](args[0])
		if isString {
			_url, err := url.Parse(value)
			if err != nil {
				panic(err)
			}
			return _url
		}
		return nil
	})
	return urlFactory
}
