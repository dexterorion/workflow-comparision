package security

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
)

const (
	AVENUE_KMS_LOCATION = "AVENUE_KMS_LOCATION"
	AVENUE_KMS_RING     = "AVENUE_KMS_RING"
	AVENUE_KMS_KEY      = "AVENUE_KMS_KEY"
)

var (
	log, _  = zap.NewDevelopmentConfig().Build()
	locId   = os.Getenv(AVENUE_KMS_LOCATION)
	keyId   = os.Getenv(AVENUE_KMS_KEY)
	keyRing = os.Getenv(AVENUE_KMS_RING)
)

func DecryptIf(projectID, cipherText string) string {
	if keyRing == "" {
		return cipherText
	}

	return Decrypt(projectID, cipherText)
}

func Decrypt(projectID, cipherText string) string {
	c, _ := base64.StdEncoding.DecodeString(cipherText)
	s, err := decrypt(projectID, locId, keyRing, keyId, []byte(c))

	if err != nil {
		log.Sugar().Panicw("failed to decrypt",
			"err", err,
			"projectID", projectID,
			"key_ring", keyRing,
			"key", keyId,
			"cypher", cipherText) //for debugging
	}

	return string(s)
}

func decrypt(projectID, locationID, keyRingID, cryptoKeyID string, ciphertext []byte) ([]byte, error) {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudkms.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	cloudkmsService, err := cloudkms.New(client)
	if err != nil {
		return nil, err
	}

	parentName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		projectID, locationID, keyRingID, cryptoKeyID)

	req := &cloudkms.DecryptRequest{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	resp, err := cloudkmsService.Projects.Locations.KeyRings.CryptoKeys.Decrypt(parentName, req).Do()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(resp.Plaintext)
}
