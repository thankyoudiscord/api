package errors

import (
	"errors"
)

// TODO: come up with a better system for this
var DiscordAPIError = errors.New("discord api error")
var DiscordAPIUnauthorized = errors.New("duscird api unauthorized request")
