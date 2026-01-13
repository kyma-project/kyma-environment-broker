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
	sess := s.Factory.NewReadSession()
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

	sess := s.Factory.NewWriteSession()
	err = sess.InsertBinding(dto)

	switch {
	case dberr.IsAlreadyExists(err):
		return dberr.AlreadyExists("while saving binding with ID %s: %v", binding.ID, err)
	case err != nil:
		return fmt.Errorf("while saving binding with ID %s: %w", binding.ID, err)
	}

	return nil
}

func (s *Binding) Update(binding *internal.Binding) error {
	dto, err := s.toBindingDTO(binding)
	if err != nil {
		return err
	}

	sess := s.Factory.NewWriteSession()
	err = sess.UpdateBinding(dto)

	if err != nil {
		return fmt.Errorf("while updating binding with ID %s: %w", binding.ID, err)
	}

	return nil
}

func (s *Binding) Delete(instanceID, bindingID string) error {
	sess := s.Factory.NewWriteSession()
	return sess.DeleteBinding(instanceID, bindingID)
}

func (s *Binding) ListByInstanceID(instanceID string) ([]internal.Binding, error) {
	dtos, err := s.Factory.NewReadSession().ListBindings(instanceID)
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

func (s *Binding) ListExpired() ([]internal.Binding, error) {
	dtos, err := s.Factory.NewReadSession().ListExpiredBindings()
	if err != nil {
		return []internal.Binding{}, err
	}
	var bindings []internal.Binding
	for _, dto := range dtos {
		binding, err := s.toExpiredBinding(dto)
		if err != nil {
			return []internal.Binding{}, err
		}

		bindings = append(bindings, binding)
	}
	return bindings, err
}

func (s *Binding) GetStatistics() (internal.BindingStats, error) {
	sess := s.Factory.NewReadSession()
	dto, err := sess.GetBindingsStatistics()
	if err != nil {
		return internal.BindingStats{}, err
	}
	return internal.BindingStats{MinutesSinceEarliestExpiration: *dto.SecondsSinceEarliestExpiration / 60}, nil
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
		CreatedBy:         binding.CreatedBy,
		ExpiresAt:         binding.ExpiresAt,
	}, nil
}

func (s *Binding) toBinding(dto dbmodel.BindingDTO) (internal.Binding, error) {
	decrypted, err := s.cipher.DecryptUsingMode([]byte(dto.Kubeconfig))
	if err != nil {
		return internal.Binding{}, fmt.Errorf("while decrypting kubeconfig: %w", err)
	}

	return internal.Binding{
		Kubeconfig:        string(decrypted),
		ID:                dto.ID,
		InstanceID:        dto.InstanceID,
		CreatedAt:         dto.CreatedAt,
		ExpirationSeconds: dto.ExpirationSeconds,
		CreatedBy:         dto.CreatedBy,
		ExpiresAt:         dto.ExpiresAt,
	}, nil
}

func (s *Binding) toExpiredBinding(dto dbmodel.BindingDTO) (internal.Binding, error) {
	return internal.Binding{
		ID:         dto.ID,
		InstanceID: dto.InstanceID,
		ExpiresAt:  dto.ExpiresAt,
	}, nil
}
