// Code generated by go-swagger; DO NOT EDIT.

package branches

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"

	strfmt "github.com/go-openapi/strfmt"
)

// NewDiffBranchesParams creates a new DiffBranchesParams object
// no default values defined in spec.
func NewDiffBranchesParams() DiffBranchesParams {

	return DiffBranchesParams{}
}

// DiffBranchesParams contains all the bound params for the diff branches operation
// typically these are obtained from a http.Request
//
// swagger:parameters diffBranches
type DiffBranchesParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: path
	*/
	BranchID string
	/*
	  Required: true
	  In: path
	*/
	OtherBranchID string
	/*
	  Required: true
	  In: path
	*/
	RepositoryID string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewDiffBranchesParams() beforehand.
func (o *DiffBranchesParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	rBranchID, rhkBranchID, _ := route.Params.GetOK("branchId")
	if err := o.bindBranchID(rBranchID, rhkBranchID, route.Formats); err != nil {
		res = append(res, err)
	}

	rOtherBranchID, rhkOtherBranchID, _ := route.Params.GetOK("otherBranchId")
	if err := o.bindOtherBranchID(rOtherBranchID, rhkOtherBranchID, route.Formats); err != nil {
		res = append(res, err)
	}

	rRepositoryID, rhkRepositoryID, _ := route.Params.GetOK("repositoryId")
	if err := o.bindRepositoryID(rRepositoryID, rhkRepositoryID, route.Formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindBranchID binds and validates parameter BranchID from path.
func (o *DiffBranchesParams) bindBranchID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route

	o.BranchID = raw

	return nil
}

// bindOtherBranchID binds and validates parameter OtherBranchID from path.
func (o *DiffBranchesParams) bindOtherBranchID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route

	o.OtherBranchID = raw

	return nil
}

// bindRepositoryID binds and validates parameter RepositoryID from path.
func (o *DiffBranchesParams) bindRepositoryID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route

	o.RepositoryID = raw

	return nil
}