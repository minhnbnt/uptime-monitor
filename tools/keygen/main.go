// Command keygen generates the ES256 JWK keypair GoTrue uses to sign
// tokens, writes it to .env.gotrue-es256, and can mint a service_role JWT
// for calling the GoTrue Admin API (used by notification-service).
//
// Usage:
//
//	go run ./tools/keygen                  # write .env.gotrue-es256
//	go run ./tools/keygen --service-token # print a service_role JWT (reads .env.gotrue-es256)
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// envPath is .env.gotrue-es256 at the repo root (parent of this tool's dir),
// where compose.yml references it.
var envPath = filepath.Join("..", "..", ".env.gotrue-es256")

type jwk struct {
	Kty    string   `json:"kty"`
	Crv    string   `json:"crv"`
	X      string   `json:"x"`
	Y      string   `json:"y"`
	D      string   `json:"d,omitempty"`
	Kid    string   `json:"kid"`
	Alg    string   `json:"alg"`
	Use    string   `json:"use"`
	KeyOps []string `json:"key_ops"`
}

func b64enc(buf []byte) string { return base64.RawURLEncoding.EncodeToString(buf) }

func b64dec(s string) []byte {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		log.Fatalf("base64 decode: %v", err)
	}
	return b
}

//	--exp 315360000  # 10 years (default for service tokens)
func main() {
	serviceToken := flag.Bool("service-token", false, "print a service_role JWT instead of generating keys")
	exp := flag.Int64("exp", 315360000, "service token lifetime in seconds (default 10y)")
	flag.Parse()

	if *serviceToken {
		printServiceToken(*exp)
		return
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	pub, err := key.PublicKey.Bytes()
	if err != nil {
		log.Fatal(err)
	}
	x := pub[1:33]
	y := pub[33:65]
	d, err := key.Bytes()
	if err != nil {
		log.Fatal(err)
	}

	kid := fmt.Sprintf("%x", sha256.Sum256(append(x, y...)))[:16]

	full := jwk{
		Kty: "EC", Crv: "P-256",
		X: b64enc(x), Y: b64enc(y), D: b64enc(d),
		Kid: kid, Alg: "ES256", Use: "sig",
		KeyOps: []string{"sign", "verify"},
	}
	pubJWK := jwk{
		Kty: "EC", Crv: "P-256",
		X: b64enc(x), Y: b64enc(y),
		Kid: kid, Alg: "ES256", Use: "sig",
		KeyOps: []string{"verify"},
	}

	keysJSON, _ := json.Marshal([]jwk{full})
	pubJSON, _ := json.Marshal(pubJWK)

	now := time.Now().Format("2006-01-02 15:04:05.000000")
	out := fmt.Sprintf("# Generated %s\n", now)
	out += fmt.Sprintf("GOTRUE_JWT_KEYS='%s'\n", string(keysJSON))
	out += "GOTRUE_JWT_VALID_METHODS=ES256\n"
	out += fmt.Sprintf("\n# Public JWK (kid=%s)\n", kid)
	out += fmt.Sprintf("# %s\n", string(pubJSON))

	if err := os.WriteFile(envPath, []byte(out), 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Print(out)
}

func printServiceToken(exp int64) {
	raw, err := os.ReadFile(envPath)
	if err != nil {
		log.Fatalf("read .env.gotrue-es256: %v (run keygen first)", err)
	}
	key := parseKeys(string(raw))

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(b64dec(key.X)),
			Y:     new(big.Int).SetBytes(b64dec(key.Y)),
		},
		D: new(big.Int).SetBytes(b64dec(key.D)),
	}

	header := fmt.Sprintf(`{"alg":"ES256","kid":"%s","typ":"JWT"}`, key.Kid)

	now := time.Now().Unix()
	claims := fmt.Sprintf(
		`{"iss":"http://gotrue:9999","sub":"","aud":"uptime-monitor","role":"service_role","iat":%d,"exp":%d}`,
		now, now+exp,
	)

	h := b64enc([]byte(header))
	c := b64enc([]byte(claims))
	payload := h + "." + c

	hash := sha256.Sum256([]byte(payload))

	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		log.Fatalf("sign: %v", err)
	}

	token := payload + "." + b64enc(append(r.Bytes(), s.Bytes()...))
	fmt.Println(token)
}

func parseKeys(env string) jwk {
	for _, line := range strings.Split(env, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "GOTRUE_JWT_KEYS=") {
			continue
		}
		val := strings.TrimPrefix(line, "GOTRUE_JWT_KEYS=")
		val = strings.Trim(val, "'\"")
		var keys []jwk
		if err := json.Unmarshal([]byte(val), &keys); err != nil {
			log.Fatalf("parse JWK: %v", err)
		}
		if len(keys) == 0 {
			log.Fatal("no keys found")
		}
		return keys[0]
	}
	log.Fatal("GOTRUE_JWT_KEYS not found in .env.gotrue-es256")
	return jwk{}
}
