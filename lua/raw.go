package lua

type RawSource interface{}

func Raw(src string) RawSource {
	return src
}
