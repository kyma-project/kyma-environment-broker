package runtime

import "strings"

type MachineTypeMultiError struct {
	errors []error
}

func (e *MachineTypeMultiError) Error() string {
	if len(e.errors) == 0 {
		return "no errors"
	}
	msg := ""
	if len(e.errors) > 1 {
		msg = "The following additionalWorkerPools have validation issues: "
	}
	errorMessages := []string{}
	for _, err := range e.errors {
		errorMessages = append(errorMessages, err.Error())
	}
	msg = msg + strings.Join(errorMessages, "; ")
	msg += ". You can update your virtual machine type only within the general-purpose machine types."
	return msg
}

func (e *MachineTypeMultiError) IsError() bool {
	return len(e.errors) > 0
}

func (e *MachineTypeMultiError) Append(err error) {
	if err != nil {
		e.errors = append(e.errors, err)
	}
}
