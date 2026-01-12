package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

func NewEncrypter(secretKey string) *Encrypter {
	return &Encrypter{key: []byte(secretKey)}
}

type Encrypter struct {
	key []byte
}

func (e *Encrypter) Encrypt(data []byte) ([]byte, error) {
	return e.encryptGCM(data)
}

// Encryption
func (e *Encrypter) EncryptSMCredentials(provisioningParameters *internal.ProvisioningParameters) error {
	if provisioningParameters.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	encrypted := internal.ERSContext{}

	creds := provisioningParameters.ErsContext.SMOperatorCredentials
	var clientID, clientSecret []byte
	if creds.ClientID != "" {
		clientID, err = e.Encrypt([]byte(creds.ClientID))
		if err != nil {
			return fmt.Errorf("while encrypting ClientID: %w", err)
		}
	}
	if creds.ClientSecret != "" {
		clientSecret, err = e.Encrypt([]byte(creds.ClientSecret))
		if err != nil {
			return fmt.Errorf("while encrypting ClientSecret: %w", err)
		}
	}
	encrypted.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientID:          string(clientID),
		ClientSecret:      string(clientSecret),
		ServiceManagerURL: creds.ServiceManagerURL,
		URL:               creds.URL,
		XSAppName:         creds.XSAppName,
	}

	provisioningParameters.ErsContext.SMOperatorCredentials = encrypted.SMOperatorCredentials
	return nil
}

func (e *Encrypter) EncryptKubeconfig(provisioningParameters *internal.ProvisioningParameters) error {
	if len(provisioningParameters.Parameters.Kubeconfig) == 0 {
		return nil
	}
	encryptedKubeconfig, err := e.Encrypt([]byte(provisioningParameters.Parameters.Kubeconfig))
	if err != nil {
		return fmt.Errorf("while encrypting kubeconfig: %w", err)
	}
	provisioningParameters.Parameters.Kubeconfig = string(encryptedKubeconfig)
	return nil
}

func (e *Encrypter) encryptGCM(data []byte) ([]byte, error) {
	aes, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithRandomNonce(aes)
	if err != nil {
		return nil, err
	}
	encoded := gcm.Seal(nil, make([]byte, gcm.NonceSize()), data, nil)
	return []byte(base64.StdEncoding.EncodeToString(encoded)), nil
}

// Decryption
type DecryptFunc func(data []byte) ([]byte, error)

func (e *Encrypter) decryptGCM(ciphertext []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(ciphertext))
	aes, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCMWithRandomNonce(aes)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (e *Encrypter) DecryptUsingMode(data []byte) ([]byte, error) {
	return e.decryptGCM(data)
}

func (e *Encrypter) DecryptSMCredentialsUsingMode(provisioningParameters *internal.ProvisioningParameters) error {
	return e.decryptSMCredentials(provisioningParameters, e.decryptGCM)
}

func (e *Encrypter) decryptSMCredentials(provisioningParameters *internal.ProvisioningParameters, decryptFunc DecryptFunc) error {
	if provisioningParameters.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	var clientID, clientSecret []byte

	credentials := provisioningParameters.ErsContext.SMOperatorCredentials
	if credentials.ClientID != "" {
		clientID, err = decryptFunc([]byte(credentials.ClientID))
		if err != nil {
			return fmt.Errorf("while decrypting ClientID: %w", err)
		}
	}
	if credentials.ClientSecret != "" {
		clientSecret, err = decryptFunc([]byte(credentials.ClientSecret))
		if err != nil {
			return fmt.Errorf("while decrypting ClientSecret: %w", err)
		}
	}

	if len(clientID) != 0 {
		provisioningParameters.ErsContext.SMOperatorCredentials.ClientID = string(clientID)
	}
	if len(clientSecret) != 0 {
		provisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = string(clientSecret)
	}
	return nil
}

func (e *Encrypter) DecryptKubeconfigUsingMode(provisioningParameters *internal.ProvisioningParameters) error {
	return e.decryptKubeconfig(provisioningParameters, e.decryptGCM)
}

func (e *Encrypter) decryptKubeconfig(provisioningParameters *internal.ProvisioningParameters, decryptFunc DecryptFunc) error {
	if len(provisioningParameters.Parameters.Kubeconfig) == 0 {
		return nil
	}

	decryptedKubeconfig, err := decryptFunc([]byte(provisioningParameters.Parameters.Kubeconfig))
	if err != nil {
		return fmt.Errorf("while decrypting kubeconfig: %w", err)
	}
	provisioningParameters.Parameters.Kubeconfig = string(decryptedKubeconfig)
	return nil
}
