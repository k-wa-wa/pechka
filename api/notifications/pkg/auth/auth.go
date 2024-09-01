package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

var (
	apiKey        = os.Getenv("API_KEY")
	protectedURLs = []*regexp.Regexp{
		regexp.MustCompile("^/api/auth/.*$"),
	}
	AuthConfig = keyauth.New(keyauth.Config{
		Next:      AuthFilter,
		KeyLookup: "header:Authorization",
		Validator: ValidateAPIKey,
	})
)

func ValidateAPIKey(c *fiber.Ctx, key string) (bool, error) {
	hashedAPIKey := sha256.Sum256([]byte(apiKey))
	hashedKey := sha256.Sum256([]byte(key))

	if subtle.ConstantTimeCompare(hashedAPIKey[:], hashedKey[:]) == 1 {
		return true, nil
	}
	return false, keyauth.ErrMissingOrMalformedAPIKey

}

func AuthFilter(c *fiber.Ctx) bool {
	originalURL := strings.ToLower(c.OriginalURL())
	for _, pattern := range protectedURLs {
		if pattern.MatchString(originalURL) {
			return false
		}
	}
	return true
}
