package parser

import (
	"errors"
	"io"
	"net/http"

	"github.com/schwichtgit/GitWebhookProxy/pkg/providers"
)

func Parse(req *http.Request, provider providers.Provider) (*providers.Hook, error) {
	hook := &providers.Hook{
		Headers: make(map[string]string),
	}

	for _, header := range provider.GetHeaderKeys() {
		if req.Header.Get(header) != "" {
			hook.Headers[header] = req.Header.Get(header)
			continue
		}
		return nil, errors.New("Required header '" + header + "' not found in Request")
	}

	if optional, ok := provider.(providers.OptionalHeaderProvider); ok {
		for _, header := range optional.GetOptionalHeaderKeys() {
			if value := req.Header.Get(header); value != "" {
				hook.Headers[header] = value
			}
		}
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	hook.Payload = body

	hook.RequestMethod = req.Method

	return hook, nil
}
