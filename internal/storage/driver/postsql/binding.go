package postsql

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
)

type Binding struct {
	postsql.Factory
	cipher Cipher
}

func NewBinding(sess postsql.Factory, cipher Cipher) *Binding {
	return &Binding{
		Factory: sess,
		cipher:  cipher,
	}
}

func (s *Binding) Get(instanceID string, bindingID string) (*internal.Binding, error) {
	sess := s.NewReadSession()
	bindingDTO := dbmodel.BindingDTO{}
	bindingDTO, dbErr := sess.GetBinding(instanceID, bindingID)
	if dbErr != nil {
		if dberr.IsNotFound(dbErr) {
			return nil, dberr.NotFound("Binding with id %s does not exist", bindingID)
		}

		return nil, fmt.Errorf("while getting bindingDTO by ID %s: %w", bindingID, dbErr)
	}

	binding, err := s.toBinding(bindingDTO)
	if err != nil {
		return nil, err
	}

	return &binding, nil
}

func (s *Binding) Insert(binding *internal.Binding) error {
	dto, err := s.toBindingDTO(binding)
	if err != nil {
		return err
	}

	sess := s.NewWriteSession()
	err = sess.InsertBinding(dto)

	if err != nil {
		return fmt.Errorf("while saving binding with ID %s: %w", binding.ID, err)
	}

	return nil
}

func (s *Binding) DeleteByBindingID(ID string) error {
	sess := s.NewWriteSession()
	return sess.DeleteBinding(ID)
}

func (s *Binding) ListByInstanceID(instanceID string) ([]internal.Binding, error) {
	dtos, err := s.NewReadSession().ListBindings(instanceID)
	if err != nil {
		return []internal.Binding{}, err
	}
	var bindings []internal.Binding
	for _, dto := range dtos {
		instance, err := s.toBinding(dto)
		if err != nil {
			return []internal.Binding{}, err
		}

		bindings = append(bindings, instance)
	}
	return bindings, err
}

func (s *Binding) toBindingDTO(binding *internal.Binding) (dbmodel.BindingDTO, error) {
	encrypted, err := s.cipher.Encrypt([]byte(binding.Kubeconfig))
	if err != nil {
		return dbmodel.BindingDTO{}, fmt.Errorf("while encrypting kubeconfig: %w", err)
	}

	return dbmodel.BindingDTO{
		Kubeconfig:        string(encrypted),
		ID:                binding.ID,
		InstanceID:        binding.InstanceID,
		CreatedAt:         binding.CreatedAt,
		ExpirationSeconds: binding.ExpirationSeconds,
	}, nil
}

func (s *Binding) toBinding(dto dbmodel.BindingDTO) (internal.Binding, error) {
	decrypted, err := s.cipher.Decrypt([]byte(dto.Kubeconfig))
	if err != nil {
		return internal.Binding{}, fmt.Errorf("while decrypting kubeconfig: %w", err)
	}

	return internal.Binding{
		Kubeconfig:        string(decrypted),
		ID:                dto.ID,
		InstanceID:        dto.InstanceID,
		CreatedAt:         dto.CreatedAt,
		ExpirationSeconds: dto.ExpirationSeconds,
	}, nil
}
