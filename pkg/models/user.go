package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	tyderrors "github.com/thankyoudiscord/api/pkg/errors"
)

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
	PublicFlags   int    `json:"public_flags"`
	Flags         int    `json:"flags"`
	Banner        string `json:"banner"`
	BannerColor   string `json:"banner_color"`
	AccentColor   int    `json:"accent_color"`
	Locale        string `json:"locale"`
	MFAEnabled    bool   `json:"mfa_enabled"`
	PremiumType   int    `json:"premium_type"`
}

func GetUser(at string) (*DiscordUser, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+at)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		bdy, _ := io.ReadAll(res.Body)
		fmt.Println("discord responded with non-200 status code:", res.StatusCode, string(bdy))

		if res.StatusCode == http.StatusUnauthorized {
			return nil, tyderrors.DiscordAPIUnauthorized
		}

		return nil, tyderrors.DiscordAPIError
	}

	var user DiscordUser
	err = json.NewDecoder(res.Body).Decode(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
