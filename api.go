package poledit

import (
	"context"
	"github.com/oracle/oci-go-sdk/identity"
)

type Handler interface {
	Find(ctx context.Context, parent string, name string, iam identity.IdentityClient) (Handler, error)
	WithDescription(description *string) Handler
	WithName(name *string) Handler

	Edit() (Handler, error)
	JsonEdit() (Handler, error)

	ShouldCreate() bool

	DoCreate(iam identity.IdentityClient, compartment identity.Compartment) error
	DoUpdate(iam identity.IdentityClient) error
	DoDelete(iam identity.IdentityClient) error
}
