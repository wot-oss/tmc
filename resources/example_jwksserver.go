package main

import (
	"bufio" // <-- New import for bufio
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv" // <-- New import for strconv to parse expiry
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	JWKSHost        = "127.0.0.1"
	JWKSPort        = "8100"
	JWKSPath        = "/.well-known/jwks.json"
	PrivateKeyFile  = "private_key.pem"
	PublicKeyFile   = "public_key.pem"
	JWKSFile        = "jwks.json"
	KeyID           = "my-test-key-id"
	DefaultAudience = "your_app_id"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	mu         sync.Mutex
)

func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating key pair: %v", err)
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	if err := os.WriteFile(PrivateKeyFile, privateKeyPEM, 0600); err != nil {
		return nil, nil, fmt.Errorf("error saving private key: %v", err)
	}
	fmt.Println("Private key saved to", PrivateKeyFile)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling public key: %v", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	if err := os.WriteFile(PublicKeyFile, publicKeyPEM, 0600); err != nil {
		return nil, nil, fmt.Errorf("error saving public key: %v", err)
	}
	fmt.Println("Public key saved to", PublicKeyFile)

	return privateKey, publicKey, nil
}

func LoadRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	mu.Lock()
	defer mu.Unlock()

	_, errPrivate := os.Stat(PrivateKeyFile)
	_, errPublic := os.Stat(PublicKeyFile)
	if os.IsNotExist(errPrivate) || os.IsNotExist(errPublic) {
		fmt.Println("Key files not found. Generating a new key pair...")
		return GenerateRSAKeyPair()
	}

	privateKeyPEM, err := os.ReadFile(PrivateKeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading private key file: %v", err)
	}

	privateKeyBlock, _ := pem.Decode(privateKeyPEM)
	if privateKeyBlock == nil || privateKeyBlock.Type != "RSA PRIVATE KEY" {
		return nil, nil, errors.New("invalid private key format")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing private key: %v", err)
	}

	publicKeyPEM, err := os.ReadFile(PublicKeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading public key file: %v", err)
	}

	publicKeyBlock, _ := pem.Decode(publicKeyPEM)
	if publicKeyBlock == nil || publicKeyBlock.Type != "PUBLIC KEY" {
		return nil, nil, errors.New("invalid public key format")
	}

	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing public key: %v", err)
	}

	fmt.Println("Loaded existing key pair.")
	return privateKey, publicKey.(*rsa.PublicKey), nil
}

func CreateJWKS(publicKey *rsa.PublicKey) error {
	n := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})

	jwk := map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"kid": KeyID,
		"alg": "RS256",
		"n":   n,
		"e":   e,
	}

	jwks := map[string]interface{}{
		"keys": []interface{}{jwk},
	}

	jwksJSON, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		return fmt.Errorf("error creating JWKS JSON: %v", err)
	}

	if err := os.WriteFile(JWKSFile, jwksJSON, 0600); err != nil {
		return fmt.Errorf("error saving JWKS: %v", err)
	}

	fmt.Println("JWKS saved to", JWKSFile)
	return nil
}

func GenerateJWT(privateKey *rsa.PrivateKey, scopes, issuer, audience string, expiryMinutes int) (string, error) {
	claims := jwt.MapClaims{
		"iss":                issuer,
		"aud":                audience,
		"sub":                "testuser@siemens.com",
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(time.Duration(expiryMinutes) * time.Minute).Unix(),
		"scope":              strings.Fields(scopes), // This part is correct for splitting space-separated scopes
		"preferred_username": "testuser",
		"email":              "testuser@siemens.com",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = KeyID

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("error signing JWT: %v", err)
	}

	return signedToken, nil
}

func jwksHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != JWKSPath {
		http.NotFound(w, r)
		return
	}

	data, err := os.ReadFile(JWKSFile)
	if err != nil {
		http.Error(w, "JWKS file not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func StartServer() {
	http.HandleFunc(JWKSPath, jwksHandler)
	fmt.Printf("Starting JWKS server on http://%s:%s%s\n", JWKSHost, JWKSPort, JWKSPath)
	http.ListenAndServe(fmt.Sprintf("%s:%s", JWKSHost, JWKSPort), nil)
}

func main() {
	var err error
	privateKey, publicKey, err = LoadRSAKeyPair()
	if err != nil {
		fmt.Println("Error loading key pair:", err)
		return
	}

	if err := CreateJWKS(publicKey); err != nil {
		fmt.Println("Error creating JWKS:", err)
		return
	}

	go StartServer()

	fmt.Println("\nJWKS server is running. Press Ctrl+C to stop.")
	fmt.Printf("JWKS URL: http://%s:%s%s\n", JWKSHost, JWKSPort, JWKSPath)

	// Initialize a new scanner for reading full lines
	scanner := bufio.NewScanner(os.Stdin)

	for {
		var scopes, issuer, audience string
		var expiry int = 60 // Set default expiry here

		fmt.Print("\nEnter custom scopes (space-separated, or 'q' to quit): ")
		scanner.Scan() // Read the entire line
		scopes = scanner.Text()
		if scopes == "q" {
			break
		}

		fmt.Printf("Enter issuer (default: http://%s:%s): ", JWKSHost, JWKSPort)
		scanner.Scan() // Read the entire line
		issuer = scanner.Text()
		if issuer == "" {
			issuer = fmt.Sprintf("http://%s:%s", JWKSHost, JWKSPort)
		}

		fmt.Printf("Enter audience (default: %s): ", DefaultAudience)
		scanner.Scan() // Read the entire line
		audience = scanner.Text()
		if audience == "" {
			audience = DefaultAudience
		}

		fmt.Print("Enter expiry (in minutes, default: 60): ")
		scanner.Scan() // Read the entire line
		expiryStr := scanner.Text()
		if expiryStr != "" {
			parsedExpiry, err := strconv.Atoi(expiryStr) // Convert string to int
			if err != nil {
				fmt.Println("Invalid expiry input. Using default (60 minutes).")
			} else {
				expiry = parsedExpiry
			}
		}

		token, err := GenerateJWT(privateKey, scopes, issuer, audience, expiry)
		if err != nil {
			fmt.Println("Error generating JWT:", err)
			continue
		}

		fmt.Println("\n--- Generated JWT ---")
		fmt.Println(token)
		fmt.Println("----------------------")
	}
}
