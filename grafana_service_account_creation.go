package grafanasdkclistarter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type GrafanaAddOn struct {
	Url    string
	client http.Client
}

func NewGrafanaAddOn(url, username, password string) *GrafanaAddOn {
	client := http.Client{
		Transport: BasicAuthTransport{credentials: BasicAuth{Username: username, Password: password}},
	}

	return &GrafanaAddOn{client: client, Url: url}
}

func (g *GrafanaAddOn) CreateAPIKey(serviceAccountName, tokenName string) (string, error) {
	type ServiceAccountPayload struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}

	s := ServiceAccountPayload{Name: serviceAccountName, Role: "Admin"}

	var data bytes.Buffer

	err := json.NewEncoder(&data).Encode(s)
	if err != nil {
		return "", fmt.Errorf("unable to marshal ServiceAccountPayload: %w", err)
	}

	surl, err := url.JoinPath(g.Url, "/api/serviceaccounts")
	if err != nil {
		return "", fmt.Errorf("unable to serviceaccounts joinPath: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, surl, &data)
	if err != nil {
		return "", fmt.Errorf("unable to create NewRequest: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to get response: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("serviceaccount was not created, %v", resp.Body)
	}

	var sar ServiceAccountResponse
	err = json.NewDecoder(resp.Body).Decode(&sar)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal ServiceAccountResponse : %w", err)
	}

	tUrl, err := url.JoinPath(surl, fmt.Sprintf("%d/tokens", sar.ID))
	if err != nil {
		return "", fmt.Errorf("unable to token joinPath: %w", err)
	}

	type ServiceAccountTokenPayload struct {
		Name string `json:"name"`
	}
	var data2 bytes.Buffer
	satp := ServiceAccountTokenPayload{Name: tokenName}
	err = json.NewEncoder(&data2).Encode(satp)
	if err != nil {
		return "", fmt.Errorf("unable to marshal ServiceAccountTokenPayload: %w", err)
	}

	req2, err := http.NewRequest(http.MethodPost, tUrl, &data2)
	if err != nil {
		return "", fmt.Errorf("unable to create NewRequest for token: %w", err)
	}

	resp2, err := g.client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("unable to execute request for token: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token was not created, %v", resp.Body)
	}
	var satr ServiceAccountTokenResponse
	err = json.NewDecoder(resp2.Body).Decode(&satr)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal ServiceAccountTokenResponse : %w", err)
	}

	return satr.Key, nil
}

type ServiceAccountTokenResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ServiceAccountResponse struct {
	ID         int64         `json:"id"`
	Name       string        `json:"name"`
	Login      string        `json:"login"`
	OrgID      int64         `json:"orgId"`
	IsDisabled bool          `json:"isDisabled"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
	AvatarURL  string        `json:"avatarUrl"`
	Role       string        `json:"role"`
	Teams      []interface{} `json:"teams"`
}
