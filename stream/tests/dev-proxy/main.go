package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	cookieName = "cf-access-token-mock"
	kid        = "mock-key-id-1"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     = os.Getenv("CLOUDFLARE_JWT_ISSUER")
	audience   = os.Getenv("CLOUDFLARE_JWT_AUDIENCE")
)

func init() {
	if issuer == "" {
		log.Fatal("CLOUDFLARE_JWT_ISSUER environment variable is required")
	}
	if audience == "" {
		log.Fatal("CLOUDFLARE_JWT_AUDIENCE environment variable is required")
	}

	// Fixed RSA private key for local testing consistency
	const testPrivKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDCiT7kOkG4B4ho
YrskHftiDOSI7R2b06h+2zR04sr6A/0HB6UaVy6sLP50YsNgzSi+s8NCkjM94/Xe
eumA3rCF4EelihMCZ6pYB69JjDGm4AyEZHkYWcMAzakW4BHc3l0ryMLMmHIN8oXp
syLLhBFgH8psln4KwH1RqRUlgvPjaqS14OqeYXh+a7X4VraLDwFWNxVPxW0BNwgM
h+WadQMhpMVXeN2KE3VXUDl+xbt6LnCoi6KiJewhs8OeRyB1TgU8GzebWVUTle7u
j0j17Ph+/14Xk9VUB5oBOuWP+XnDewIfN8cGB7qBsgY6Zcvvr6fIrVTNmKz8iI6p
QaFJwGh9AgMBAAECggEAV7aFPmeMCTuQRCy8H4lLNscEZj6venrBPs18hfVaOseA
l2JZjZpgp240HusHGAb5B59K+6Gq7A10Zyd5UEtYQUzCUUAD2TI/qqhwXxOQsaLU
0f7xYMrcM2kHhBJsy28RiHPhbVmRF3vR6HEGT8gRA4vh+/sRAq0O9DpuF/dHGzLJ
pgMKKKQwJ3Q29U+RO+NYd7O9UgH+1G8PJfahVGkNTkiNk7qdOC0veiJWGFSvkbWa
HkmhP0TNoaBb0YMFQ3BkeYkT3lXsmnPLUEh9mKJfqBx1UB0tcj8baaY3Kc6SpHSt
i/xc8VKuVQ5VdOWYGBxPRkofba1v1LIXLAWhNs5t2wKBgQDiCAjwEszKBEJSXYKg
djqI760gF+N28wuxfBy/ZzLyIgbAcEGMSwcAqMtffwEa00ovwJ1NhhL3HN8MfOCI
oXB9ub8Crbbub7kgr4yLTH48zCZdOuIRra6Am7QF7e9h4MKJLT3XrxXpTNeIzJof
tR4257fLcQKqwAKgY++H+tCjawKBgQDcVDJ760LSlK+XN4p/qtPwUtqSJluvCZms
w9rxo4u5r4DgZ/La5aFA0LlKED05LUTByLtZDoYlnNbrsU90ey0tZJIOHM3g/qkV
Kg9VjoHCiMFiGG/+zM32kUAEWkunfG3AFn9uKXXk6MzU+sDkJkHu7SvOhgdVwUuU
UUK4H1+FtwKBgHzsTexJp6+bTQByuChxT4axWLDdIxVx3KuaWdUbd1fFoI+pO0EL
knI12DkOW5D06BKeVRIsoLy80zX2qq4485A5Ia2cTvdW/i1neLjgbQCzIBz0109H
+6MO6x8/0sb4zuu7+msDVIvdsV3lHuWZV3qm9LjW2899UbZNpWw1HizDAoGALxmo
uSjv3gh/CPqMlwIz0HpF01xz2RVaTr6HvYRSyF0mVdKi7fyM3khAc/7It8JfonWA
52bdcoj2wOfkrmfunneTaYTq1iBakPWu1YFjZ+zIOmoy9utdVEp0vvl2ltVYuOmW
UDx4wXiq4RTBy4QKMENvS/UG+GQb/hbpBmdeij0CgYEAozIy7sqdJ04Sz3KLDn2Y
/oH70gFEdYcwgBnEsWxnV7wL8ZqlesCxE7lB+yeNaO/6k6wXtdrFI6uL5TEBXHmT
l51lDCmhb976QpeDdTw0ePSwx1K2Z2eSaMNwrrUn0PBDqUmOJFE73nns6zVUmv7b
yYqbGNsiHbcQrQJ4bzCFj9c=
-----END PRIVATE KEY-----`

	block, _ := pem.Decode([]byte(testPrivKey))
	if block == nil {
		log.Fatal("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}

	var ok bool
	privateKey, ok = priv.(*rsa.PrivateKey)
	if !ok {
		log.Fatal("not an RSA private key")
	}
	publicKey = &privateKey.PublicKey
}

type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
}

type JSONWebKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func main() {
	targetStr := os.Getenv("PROXY_TARGET")
	if targetStr == "" {
		log.Fatal("PROXY_TARGET environment variable is required")
	}
	target, err := url.Parse(targetStr)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	http.HandleFunc("/.well-known/jwks.json", handleJWKS)
	http.HandleFunc("/mock/token", handleToken)
	http.HandleFunc("/dev-proxy/login", handleLogin)
	http.HandleFunc("/dev-proxy/auth", handleAuth)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Internal paths skip auth check
		if strings.HasPrefix(r.URL.Path, "/.well-known/") ||
			strings.HasPrefix(r.URL.Path, "/mock/") ||
			strings.HasPrefix(r.URL.Path, "/dev-proxy/") {
			w.Header().Set("X-Dev-Proxy", "internal")
		} else {
			// Auth check for proxied paths
			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				// Redirect to login UI
				loginURL := fmt.Sprintf("/dev-proxy/login?return_to=%s", url.QueryEscape(r.URL.String()))
				log.Printf("Unauthenticated request to %s, redirecting to %s", r.URL.Path, loginURL)
				http.Redirect(w, r, loginURL, http.StatusFound)
				return
			}

			// Add CF header from cookie value (JWT)
			r.Header.Set("Cf-Access-Jwt-Assertion", cookie.Value)
			log.Printf("Authenticated proxying (user from cookie): %s %s", r.Method, r.URL.Path)
		}

		r.Host = target.Host
		proxy.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Dev Proxy (Interactive Mode) starting on port %s", port)
	log.Printf("Proxy Target: %s", targetStr)
	log.Printf("Issuer: %s, Audience: %s", issuer, audience)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleJWKS(w http.ResponseWriter, r *http.Request) {
	log.Println("JWKS requested")
	n := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes())

	jwks := JWKS{
		Keys: []JSONWebKey{
			{
				Kty: "RSA",
				Kid: kid,
				Use: "sig",
				Alg: "RS256",
				N:   n,
				E:   e,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

func handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusLengthRequired)
		return
	}

	tokenStr, err := generateJWT(req.Email)
	if err != nil {
		http.Error(w, "Failed to sign token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenStr,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dev Proxy Login</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f4f7f9; }
        .card { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); width: 100%%; max-width: 400px; }
        h1 { font-size: 1.5rem; margin-bottom: 1rem; color: #333; }
        p { color: #666; font-size: 0.9rem; margin-bottom: 1.5rem; }
        input[type="email"] { width: 100%%; padding: 0.75rem; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; margin-bottom: 1rem; font-size: 1rem; }
        button { width: 100%%; padding: 0.75rem; background: #007aff; color: white; border: none; border-radius: 4px; font-size: 1rem; cursor: pointer; transition: background 0.2s; }
        button:hover { background: #005bb5; }
        .footer { margin-top: 1.5rem; font-size: 0.75rem; color: #999; text-align: center; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Dev Proxy 認証</h1>
        <p>開発用モック環境です。ログインに使用するメールアドレスを入力してください。</p>
        <form action="/dev-proxy/auth" method="POST">
            <input type="hidden" name="return_to" value="%s">
            <input type="email" name="email" placeholder="test@example.com" required autofocus>
            <button type="submit">ログイン</button>
        </form>
        <div class="footer">Cloudflare Access 挙動をシミュレートしています</div>
    </div>
</body>
</html>
`, returnTo)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.FormValue("email")
	returnTo := r.FormValue("return_to")
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	tokenStr, err := generateJWT(email)
	if err != nil {
		http.Error(w, "Failed to sign token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokenStr,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	if returnTo == "" {
		returnTo = "/"
	}
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func generateJWT(email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   issuer,
		"aud":   audience,
		"sub":   "mock-sub",
		"email": email,
		"exp":   now.Add(24 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	return token.SignedString(privateKey)
}
