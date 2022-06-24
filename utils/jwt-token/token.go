package jwt_token

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

func JwtParse(tokenString string, jwtSecret string) (jwtData map[string]interface{}, err error) {
	jwtData = make(map[string]interface{})
	token, err := jwt.Parse(tokenString, secretFunc(jwtSecret))
	if err != nil {
		return
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		for k, v := range claims {
			jwtData[k] = v
		}
		return
	}
	return jwtData, fmt.Errorf("invalid token")
}

// example
//  jwt.MapClaims{
//		"user_id":   jwtData.UserId,
//		"user_name": jwtData.UserName,
//		"nbf":       time.Now().Unix(),
//		"iat":       time.Now().Unix(),
//		"exp":       time.Now().Add(duration).Unix(),
//	}
func JwtSign(jwtData map[string]interface{}, jwtSecret string) (tokenString string, err error) {
	jwtMap := jwt.MapClaims{}
	for k, v := range jwtData {
		jwtMap[k] = v
	}
	var token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwtMap)
	tokenString, err = token.SignedString([]byte(jwtSecret))
	return
}

func secretFunc(secret string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	}
}
