package auth

import (
	"encoding/json"
	"github.com/golang-jwt/jwt"
	"net/http"
	"time"
)

// JwtProvider is the representation of decoded JWT
type JwtProvider struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// Create a struct to read the username and password from the request body
type credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

const tokenName = "souin-authorization-token"
const lifetime = time.Hour * 24 * 7

func signJWT(security *SecurityAPI, w http.ResponseWriter, r *http.Request) {
	var creds credentials
	// Get the JSON body and decode into credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the expected password from our in memory map
	expectedPassword, ok := security.users[creds.Username]

	// If a password exists for the given user
	// AND, if it is the same as the password we received, the we can move ahead
	// if NOT, then we return an "Unauthorized" status
	if !ok || expectedPassword != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Create the JWT claims, which includes the username and expiry time
	claims := &JwtProvider{
		Username:       creds.Username,
		StandardClaims: jwt.StandardClaims{},
	}

	setCookie(w, claims, security.secret)
}

func refresh(security *SecurityAPI, w http.ResponseWriter, r *http.Request) {
	claims, err := CheckToken(security, w, r)
	if err != nil {
		return
	}

	// We ensure that a new token is not issued until enough time has elapsed
	// In this case, a new token will only be issued if the old token is within
	// 30 seconds of expiry. Otherwise, return a bad request status
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 24*time.Hour {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	setCookie(w, claims, security.secret)
}

// CheckToken will return if token is valid or not
func CheckToken(security *SecurityAPI, w http.ResponseWriter, r *http.Request) (*JwtProvider, error) {
	c, err := r.Cookie(tokenName)
	if err != nil {
		if err.Error() == http.ErrNoCookie.Error() {
			w.WriteHeader(http.StatusUnauthorized)
			return nil, &tokenError{found: false}
		}
		w.WriteHeader(http.StatusBadRequest)
		return nil, &tokenError{found: true}
	}
	tknStr := c.Value
	claims := &JwtProvider{}
	tkn, e := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return security.secret, nil
	})
	if e != nil {
		if jwt.ErrSignatureInvalid.Error() == e.Error() {
			w.WriteHeader(http.StatusUnauthorized)
			return claims, &signatureError{}
		}
		w.WriteHeader(http.StatusBadRequest)
		return claims, &signatureError{}
	}
	if !tkn.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return claims, &signatureError{}
	}

	return claims, nil
}

func setCookie(w http.ResponseWriter, claims *JwtProvider, secret []byte) {
	expirationTime := time.Now().Add(lifetime)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    tokenName,
		Path:    "/",
		Value:   tokenString,
		Expires: expirationTime,
	})
}
