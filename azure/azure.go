package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	oidc "github.com/coreos/go-oidc"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"golang.org/x/oauth2"
)

type AzureSettings struct {
	TenantID                string
	ClientID                string
	ActiveDirectoryEndpoint string
}

type azureProvider struct {
	oidcVerifier *oidc.IDTokenVerifier
	settings     *AzureSettings
	httpClient   *http.Client
}

type oidcDiscoveryInfo struct {
	Issuer  string `json:"issuer"`
	JWKSURL string `json:"jwks_uri"`
}

var claims struct {
	Name     string `json:"name"`
	UserName string `json:"preferred_username"`
}

func Azure(ADtoken string) (valid bool, err error) {
	validation := false
	settings := AzureSettings{
		TenantID:                "",
		ClientID:                "",
		ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
	
	}
	//ADtoken := GenerateToken()
	token, err := VerifyToken(&settings, ADtoken)
	if err != nil {
		log.Println("token validation failed:", err)
		return
	}
	validation = true
	log.Println("Issuer:", token.Issuer, "Audience:", token.Audience, "Expiry:", token.Expiry, "Claims:", token.Claims(&claims))
	return validation, err
}

func VerifyToken(settings *AzureSettings, rawIDToken string) (*oidc.IDToken, error) {
	provider, err := newAzureProvider(settings)
	if err != nil {
		return nil, err
	}
	//trimmedIDToken := strings.TrimPrefix(rawIDToken, "Bearer ")
	return provider.oidcVerifier.Verify(context.Background(), rawIDToken)
}

// copied from https://github.com/hashicorp/vault-plugin-auth-azure/blob/4c0b46069a2293d5a6ca7506c8d3e0c4a92f3dbc/azure.go#L58
func newAzureProvider(settings *AzureSettings) (*azureProvider, error) {
	httpClient := cleanhttp.DefaultClient()

	discoveryURL := fmt.Sprintf("%s%s/v2.0/.well-known/openid-configuration", settings.ActiveDirectoryEndpoint, settings.TenantID)
	req, err := http.NewRequest("GET", discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}
	var discoveryInfo oidcDiscoveryInfo
	if err := json.Unmarshal(body, &discoveryInfo); err != nil {
		return nil, fmt.Errorf("unable to unmarshal discovery url: %w", err)
	}

	//fmt.Printf("Found discoveryInfo %+v", discoveryInfo)

	// Create a remote key set from the discovery endpoint
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	remoteKeySet := oidc.NewRemoteKeySet(ctx, discoveryInfo.JWKSURL)

	verifierConfig := &oidc.Config{
		ClientID:             settings.ClientID,
		SupportedSigningAlgs: []string{oidc.RS256},
	}
	oidcVerifier := oidc.NewVerifier(discoveryInfo.Issuer, remoteKeySet, verifierConfig)

	return &azureProvider{
		settings:     settings,
		oidcVerifier: oidcVerifier,
		httpClient:   httpClient,
	}, nil
}

func userAgent() string {
	// latest chrome on linux
	return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Safari/537.36"
}
