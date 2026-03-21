package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        = "mock-key-id-1"
	issuer     = os.Getenv("CLOUDFLARE_JWT_ISSUER")
	audience   = os.Getenv("CLOUDFLARE_JWT_AUDIENCE")
)

func init() {
	if issuer == "" {
		issuer = "https://pechka.cloudflareaccess.com"
	}
	if audience == "" {
		audience = "mock-audience"
	}

	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
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
		targetStr = "http://cilium-gateway-cilium-gateway.default.svc.cluster.local:80"
	}
	target, err := url.Parse(targetStr)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	http.HandleFunc("/.well-known/jwks.json", handleJWKS)
	http.HandleFunc("/mock/token", handleToken)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying: %s %s", r.Method, r.URL.Path)
		r.Host = target.Host
		proxy.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Mock Auth & Proxy starting on port %s", port)
	log.Printf("Issuer: %s, Audience: %s", issuer, audience)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleJWKS(w http.ResponseWriter, r *http.Request) {
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

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   issuer,
		"aud":   audience,
		"sub":   "mock-sub",
		"email": req.Email,
		"exp":   now.Add(24 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	tokenStr, err := token.SignedString(privateKey)
	if err != nil {
		http.Error(w, "Failed to sign token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenStr,
	})
}
