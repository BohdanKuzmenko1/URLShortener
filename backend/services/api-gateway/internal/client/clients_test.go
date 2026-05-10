package client_test

import (
	"github.com/BohdanKuzmenko1/URLShortener/services/api-gateway/internal/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClients_EmptyAddress(t *testing.T) {
	tests := []struct {
		name      string
		urlAddr   string
		authAddr  string
		statsAddr string
		expectErr string
	}{
		{
			name:      "empty url service address",
			urlAddr:   "",
			authAddr:  "localhost:50051",
			statsAddr: "localhost:50052",
			expectErr: "url service",
		},
		{
			name:      "empty auth service address",
			urlAddr:   "localhost:50050",
			authAddr:  "",
			statsAddr: "localhost:50052",
			expectErr: "auth service",
		},
		{
			name:      "empty stats service address",
			urlAddr:   "localhost:50050",
			authAddr:  "localhost:50051",
			statsAddr: "",
			expectErr: "stats service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clients, err := client.NewClients(tt.urlAddr, tt.authAddr, tt.statsAddr)
			assert.Nil(t, clients)
			assert.ErrorContains(t, err, tt.expectErr)
		})
	}
}
