package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	encryptionModeCFB = "AES-CFB"
	encryptionModeGCM = "AES-GCM"
)

func NewEncrypter(secretKey string) *Encrypter {
	return &Encrypter{key: []byte(secretKey)}
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

func (e *Encrypter) Encrypt(obj []byte) ([]byte, error) {
	if e.GetWriteGCMMode() {
		return e.encryptGCM(obj)
	} else {
		return e.encryptCFB(obj)
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

func (e *Encrypter) encryptCFB(obj []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(obj)
	bytes := make([]byte, aes.BlockSize+len(b))
	iv := bytes[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(bytes[aes.BlockSize:], []byte(b))

	return []byte(base64.StdEncoding.EncodeToString(bytes)), nil
}

func (e *Encrypter) encryptGCM(obj []byte) ([]byte, error) {
	panic("not implemented")
	return nil, nil
}

// Decryption

func (e *Encrypter) decryptCFB(obj []byte) ([]byte, error) {
	obj, err := base64.StdEncoding.DecodeString(string(obj))
	if err != nil {
		return nil, fmt.Errorf("while decoding input object: %w", err)
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	if len(obj) < aes.BlockSize {
		return nil, fmt.Errorf("cipher text is too short")
	}
	iv := obj[:aes.BlockSize]
	obj = obj[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(obj, obj)
	data, err := base64.StdEncoding.DecodeString(string(obj))
	if err != nil {
		return nil, fmt.Errorf("while decoding internal object: %w", err)
	}
	return data, nil
}

func (e *Encrypter) DecryptGCM(obj []byte) ([]byte, error) {
	panic("not implemented")
	return nil, nil
}

func (e *Encrypter) DecryptUsingMode(text []byte, encryptionMode string) ([]byte, error) {
	switch encryptionMode {
	case encryptionModeCFB:
		return e.decryptCFB(text)
	case encryptionModeGCM:
		return e.DecryptGCM(text)
	default:
		return e.decryptCFB(text)
	}
}

func (e *Encrypter) DecryptSMCredentialsUsingMode(provisioningParameters *internal.ProvisioningParameters, encryptionMode string) error {
	var err error
	switch encryptionMode {
	case encryptionModeCFB:
		err = e.decryptSMCredentialsCFB(provisioningParameters)
	case encryptionModeGCM:
		err = e.decryptSMCredentialsGCM(provisioningParameters)
	default:
		err = e.decryptSMCredentialsCFB(provisioningParameters)
	}
	return err
}

func (e *Encrypter) DecryptKubeconfigUsingMode(provisioningParameters *internal.ProvisioningParameters, encryptionMode string) error {
	var err error
	switch encryptionMode {
	case encryptionModeCFB:
		err = e.decryptKubeconfigCFB(provisioningParameters)
	case encryptionModeGCM:
		err = e.decryptKubeconfigGCM(provisioningParameters)
	default:
		err = e.decryptKubeconfigCFB(provisioningParameters)
	}
	return err
}

func (e *Encrypter) decryptSMCredentialsCFB(provisioningParameters *internal.ProvisioningParameters) error {
	if provisioningParameters.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	var clientID, clientSecret []byte

	creds := provisioningParameters.ErsContext.SMOperatorCredentials
	if creds.ClientID != "" {
		clientID, err = e.DecryptGCM([]byte(creds.ClientID))
		if err != nil {
			return fmt.Errorf("while decrypting ClientID: %w", err)
		}
	}
	if creds.ClientSecret != "" {
		clientSecret, err = e.decryptCFB([]byte(creds.ClientSecret))
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

func (e *Encrypter) decryptSMCredentialsGCM(provisioningParameters *internal.ProvisioningParameters) error {
	if provisioningParameters.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	var clientID, clientSecret []byte

	creds := provisioningParameters.ErsContext.SMOperatorCredentials
	if creds.ClientID != "" {
		clientID, err = e.DecryptGCM([]byte(creds.ClientID))
		if err != nil {
			return fmt.Errorf("while decrypting ClientID: %w", err)
		}
	}
	if creds.ClientSecret != "" {
		clientSecret, err = e.DecryptGCM([]byte(creds.ClientSecret))
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

func (e *Encrypter) decryptKubeconfigCFB(provisioningParameters *internal.ProvisioningParameters) error {
	if len(provisioningParameters.Parameters.Kubeconfig) == 0 {
		return nil
	}

	decryptedKubeconfig, err := e.decryptCFB([]byte(provisioningParameters.Parameters.Kubeconfig))
	if err != nil {
		return fmt.Errorf("while decrypting kubeconfig: %w", err)
	}
	provisioningParameters.Parameters.Kubeconfig = string(decryptedKubeconfig)
	return nil
}

func (e *Encrypter) decryptKubeconfigGCM(provisioningParameters *internal.ProvisioningParameters) error {
	if len(provisioningParameters.Parameters.Kubeconfig) == 0 {
		return nil
	}

	decryptedKubeconfig, err := e.DecryptGCM([]byte(provisioningParameters.Parameters.Kubeconfig))
	if err != nil {
		return fmt.Errorf("while decrypting kubeconfig: %w", err)
	}
	provisioningParameters.Parameters.Kubeconfig = string(decryptedKubeconfig)
	return nil
}
