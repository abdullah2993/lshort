package lshort

type LinkShortner interface {
	Shrink(url string) (string, error)
	Expand(key string) (string, error)
}
