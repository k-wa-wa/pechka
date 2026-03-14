package domain

type ShortIDGenerator interface {
	Generate() string
}
