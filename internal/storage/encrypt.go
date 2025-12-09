package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	EncryptionModeCFB = "AES-CFB"
	EncryptionModeGCM = "AES-GCM"
)

func NewEncrypter(secretKey string, encodeGCM bool) *Encrypter {
	return &Encrypter{key: []byte(secretKey), encodeGCM: encodeGCM}
}

type Encrypter struct {
	key       []byte
	encodeGCM bool
}

func (e *Encrypter) SetWriteGCMMode(mode bool) {
	e.encodeGCM = mode
}

func (e *Encrypter) GetWriteGCMMode() bool {
	return e.encodeGCM
}

func (e *Encrypter) GetEncryptionMode() string {
	if e.GetWriteGCMMode() {
		return EncryptionModeGCM
	} else {
		return EncryptionModeCFB
	}
}

func (e *Encrypter) Encrypt(data []byte) ([]byte, error) {
	if e.GetWriteGCMMode() {
		return e.encryptGCM(data)
	} else {
		return e.encryptCFB(data)
	}
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

func (e *Encrypter) encryptCFB(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(data)
	bytes := make([]byte, aes.BlockSize+len(b))
	iv := bytes[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(bytes[aes.BlockSize:], []byte(b))

	return []byte(base64.StdEncoding.EncodeToString(bytes)), nil
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

func (e *Encrypter) decryptCFB(data []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("while decoding input object: %w", err)
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("cipher text is too short")
	}
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(data, data)
	decryptedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("while decoding internal object: %w", err)
	}
	return decryptedData, nil
}

func (e *Encrypter) decryptGCM(ciphertext []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(ciphertext))
	aes, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(aes)
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

func (e *Encrypter) DecryptUsingMode(data []byte, encryptionMode string) ([]byte, error) {
	switch strings.ToUpper(encryptionMode) {
	case EncryptionModeCFB:
		return e.decryptCFB(data)
	case EncryptionModeGCM:
		return e.decryptGCM(data)
	default:
		return e.decryptCFB(data)
	}
}

func (e *Encrypter) DecryptSMCredentialsUsingMode(provisioningParameters *internal.ProvisioningParameters, encryptionMode string) error {
	var err error
	switch strings.ToUpper(encryptionMode) {
	case EncryptionModeCFB:
		err = e.decryptSMCredentials(provisioningParameters, e.decryptCFB)
	case EncryptionModeGCM:
		err = e.decryptSMCredentials(provisioningParameters, e.decryptGCM)
	default:
		err = e.decryptSMCredentials(provisioningParameters, e.decryptCFB)
	}
	return err
}

func (e *Encrypter) decryptSMCredentials(provisioningParameters *internal.ProvisioningParameters, decryptFunc DecryptFunc) error {
	if provisioningParameters.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	var clientID, clientSecret []byte

	creds := provisioningParameters.ErsContext.SMOperatorCredentials
	if creds.ClientID != "" {
		clientID, err = decryptFunc([]byte(creds.ClientID))
		if err != nil {
			return fmt.Errorf("while decrypting ClientID: %w", err)
		}
	}
	if creds.ClientSecret != "" {
		clientSecret, err = decryptFunc([]byte(creds.ClientSecret))
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

func (e *Encrypter) DecryptKubeconfigUsingMode(provisioningParameters *internal.ProvisioningParameters, encryptionMode string) error {
	var err error
	switch encryptionMode {
	case EncryptionModeCFB:
		err = e.decryptKubeconfig(provisioningParameters, e.decryptCFB)
	case EncryptionModeGCM:
		err = e.decryptKubeconfig(provisioningParameters, e.decryptGCM)
	default:
		err = e.decryptKubeconfig(provisioningParameters, e.decryptCFB)
	}
	return err
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
