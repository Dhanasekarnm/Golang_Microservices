package auth

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"

	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"pythonrpa/elasticsearch"

	"github.com/elastic/go-elasticsearch/esapi"
)

var (
	ErrWrongFormatKey          = errors.New("the key has a wrong format")
	ErrUnexpectedBytesPayload  = errors.New("unexpected bytes payload")
	ErrUnexpectedPayloadLength = errors.New("unexpected payload length")
	paddingBytesSequence       = []byte{0xff, 0}
)

func GenerateRSAkey() (private string, public string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Println("Error reading privatekey: ", err)
		return
	}
	publicKey := &privateKey.PublicKey
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	byteSliceprv := []byte(privateKeyPEM)
	prvencoded := base64.StdEncoding.EncodeToString(byteSliceprv)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Println("Error reading publickey: ", err)
		return
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	byteSlicepub := []byte(publicKeyPEM)
	pubencoded := base64.StdEncoding.EncodeToString(byteSlicepub)

	return prvencoded, pubencoded

	// privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	// if err != nil {
	// 	panic(err)
	// }
	// publicKey := &privateKey.PublicKey
	// //println("Publickey:", publicKey)
	// privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	// privateKeyPEM := pem.EncodeToMemory(&pem.Block{
	// 	Type:  "RSA PRIVATE KEY",
	// 	Bytes: privateKeyBytes,
	// })
	// err = os.WriteFile("private.pem", privateKeyPEM, 0644)
	// if err != nil {
	// 	panic(err)
	// }
	// publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	// if err != nil {
	// 	panic(err)
	// }
	// publicKeyPEM := pem.EncodeToMemory(&pem.Block{
	// 	Type:  "RSA PUBLIC KEY",
	// 	Bytes: publicKeyBytes,
	// })
	// println(publicKeyPEM)
	// err = os.WriteFile("public.pem", publicKeyPEM, 0644)
	// if err != nil {
	// 	panic(err)
	// }
	// //--------------------------------------------
	// randomText := "test"
	// plaintext := []byte(randomText)
	// privateKeyPEM1, err := os.ReadFile("private.pem")
	// if err != nil {
	// 	panic(err)
	// }
	// prvkey, err := ParsePrivate(privateKeyPEM1)
	// if err != nil {
	// 	fmt.Println("Error Parsing Privatekey:", err)
	// }
	// chiphertext, err := PrivateEncrypt(plaintext, prvkey)
	// if err != nil {
	// 	fmt.Println("Error Ecrypting Privatekey:", err)
	// }
	// fmt.Printf("Chiphertext: %x\n", chiphertext)
	// byteSlice := []byte(chiphertext)
	// encoded := base64.StdEncoding.EncodeToString(byteSlice)
	// fmt.Printf("chipertext encoded: %s\n", encoded)
	// //--------------------------------------------------------
	// publicKeyPEM1, err := os.ReadFile("public.pem")
	// if err != nil {
	// 	panic(err)
	// }
	// pubkey, err := ParsePublic(publicKeyPEM1)
	// if err != nil {
	// 	fmt.Println("Error Parsing Publickey:", err)
	// }
	// decryptedtxt, err := PublicDecrypt(chiphertext, pubkey)
	// if err != nil {
	// 	fmt.Println("Error Decrypting Publickey:", err)
	// }
	// fmt.Printf("Decrypted: %s\n", decryptedtxt)
	// decoded := string(decryptedtxt)
	// fmt.Printf("chipertext decoded: %s\n", decoded)
}

func GenerateChipherText(randomText string, privatekey string) (chiphertxt string) {
	plaintext := []byte(randomText)
	privateKeyPEM1, err := base64.StdEncoding.DecodeString(privatekey)
	if err != nil {
		log.Println("Error reading privatekey: ", err)
		return
	}
	//log.Printf("%q\n", privateKeyPEM1)
	prvkey, err := ParsePrivate(privateKeyPEM1)
	if err != nil {
		log.Println("Error Parsing Privatekey:", err)
		return
	}
	chiphertext, err := PrivateEncrypt(plaintext, prvkey)
	if err != nil {
		log.Println("Error Ecrypting Privatekey:", err)
		return
	}
	//log.Printf("Chiphertext: %x\n", chiphertext)
	byteSlice := []byte(chiphertext)
	encoded := base64.StdEncoding.EncodeToString(byteSlice)
	//log.Printf("chipertext encoded: %s\n", encoded)
	return encoded
}

func CheckAuth(worker_id string, ct string) (mystring string) {
	worker_id_prim := worker_id
	chiphertxt := ct
	chiphertext, err := base64.StdEncoding.DecodeString(chiphertxt)
	if err != nil {
		log.Println("Error in decoding chipher text:", err)
		return
	}
	publickey := Getpublickey(worker_id_prim)
	publicKeyPEM1, err := base64.StdEncoding.DecodeString(publickey)
	if err != nil {
		log.Println("Error in decoding publick key:", err)
		return
	}
	pubkey, err := ParsePublic(publicKeyPEM1)
	if err != nil {
		log.Println("Error Parsing Publickey:", err)
		return
	}
	decryptedtxt, err := PublicDecrypt(chiphertext, pubkey)
	if err != nil {
		log.Println("Error Decrypting Publickey:", err)
		return
	}
	randomtext := string(decryptedtxt[:])
	return randomtext
}

func Getpublickey(worker_id_prim string) string {
	var publickey string
	client, err := elasticsearch.ESConnect()
	if err != nil {
		log.Fatalln("Elasticsearch connection error:", err)
	}
	query := `{"query":{
		"bool": {
			  "filter": [
				  { "term": { "_id": "` + worker_id_prim + `" }}
				  ]
			  }
		}}`
	searchresp, err := client.Search(
		client.Search.WithIndex("automate-orch-workers"),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Fatalln("Search failed, Check search query", err)
	}
	docs := elasticsearch.ESparsehits((*esapi.Response)(searchresp))

	if len(docs) == 1 {
		publickey = fmt.Sprint(docs[0]["_source"].(map[string]interface{})["publicKey"])
	}
	defer searchresp.Body.Close()
	return publickey
}

// PrivateEncrypt encrypts a payload with private key and return encrypted bytes or error.
func PrivateEncrypt(payload []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if len(payload) < 1 {
		return nil, ErrUnexpectedPayloadLength
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.Hash(0), payload) // Note: crypto.Hash(0), unhashed payload
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return signature, nil
}

// PublicDecrypt decrypts a payload with public key and then return decrypted bytes.
func PublicDecrypt(payload []byte, pubKey *rsa.PublicKey) ([]byte, error) {
	if len(payload) < 1 {
		return nil, ErrUnexpectedPayloadLength
	}

	m := new(big.Int).SetBytes(payload)
	e := big.NewInt(int64(pubKey.E))
	c := new(big.Int).Exp(m, e, pubKey.N)

	bbytes := bytes.Split(c.Bytes(), paddingBytesSequence)
	if len(bbytes) != 2 {
		return nil, ErrUnexpectedBytesPayload
	}

	return bbytes[1], nil
}

// ParsePublic parses *rsa.PublicKey from the raw bytes data.
func ParsePublic(publicKey []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKey)

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	puk, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, ErrWrongFormatKey
	}

	return puk, nil
}

// ParsePrivate parses *rsa.PrivateKey from the raw bytes data.
func ParsePrivate(privateKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKey)

	parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes) //ParsePKCS8PrivateKey(block.Bytes) //
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return parsed, nil
}
