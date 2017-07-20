package base

type Cashe interface {
	Get(key string) (string, error)
	Set(key, value string) error
}
