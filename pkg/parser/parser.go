package parser

import (
	"errors"
	"io"
	"net/http"

	"github.com/stakater/GitWebhookProxy/pkg/providers"
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

	if body, err := io.ReadAll(req.Body); err != nil {
		return nil, err
	} else {
		hook.Payload = body
	}

	hook.RequestMethod = req.Method

	return hook, nil
}
