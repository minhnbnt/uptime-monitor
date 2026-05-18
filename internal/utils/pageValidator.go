package utils

import (
	"fmt"
)

type PageValidatorError error

type PageValidator struct {
	maxPageSize int
}

func NewPageValidator(maxPageSize int) *PageValidator {
	return &PageValidator{maxPageSize: maxPageSize}
}

func (v *PageValidator) validate(pageNumber, pageSize int) error {

	if pageNumber <= 0 {
		return fmt.Errorf("page number must be greater than 0")
	}

	if pageSize >= v.maxPageSize {
		return fmt.Errorf("page size must be less than %v", v.maxPageSize)
	}

	return nil
}

func (v *PageValidator) Validate(pageNumber, pageSize int) error {

	if err := v.validate(pageNumber, pageSize); err != nil {
		return PageValidatorError(err)
	}

	return nil
}
