package postsql

import "github.com/kyma-project/kyma-environment-broker/internal"

type Cipher interface {
	Encrypt(text []byte) ([]byte, error)
	DecryptUsingMode(text []byte) ([]byte, error)

	// methods used to encrypt/decrypt SM credentials
	EncryptSMCredentials(pp *internal.ProvisioningParameters) error
	DecryptSMCredentialsUsingMode(pp *internal.ProvisioningParameters) error

	// methods used to encrypt/decrypt kubeconfig
	EncryptKubeconfig(pp *internal.ProvisioningParameters) error
	DecryptKubeconfigUsingMode(pp *internal.ProvisioningParameters) error
}
