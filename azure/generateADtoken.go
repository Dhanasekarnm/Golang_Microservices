package azure

import (
	"context"
	"log"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
)

func GenerateToken() (token string) {
	// confidential clients have a credential, such as a secret or a certificate
	//cred, err := confidential.NewCredFromSecret("<SECRET>")
	cred, err := confidential.NewCredFromSecret("<SECRET>")
	if err != nil {
		log.Println(err)
		return
	}
	//
	confidentialClient, err := confidential.New("https://login.microsoftonline.com/<TENANT_ID>/", "<CLIENT_ID>", cred)
	if err != nil {
		log.Println(err)
		return
	}
	//scopes := []string{"https://graph.microsoft.com/.default"}
	scopes := []string{"https://<CLIENT_ID>/.default"}
	result, err := confidentialClient.AcquireTokenSilent(context.TODO(), scopes)
	if err != nil {
		// cache miss, authenticate with another AcquireToken... method
		result, err = confidentialClient.AcquireTokenByCredential(context.TODO(), scopes)
		if err != nil {
			log.Println(err)
			return
		}
	}
	accessToken := result.AccessToken
	log.Println(accessToken)
	return accessToken
}
