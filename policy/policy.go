package policy

import (
	"context"
	"github.com/jan-g/oci-poledit"
	"github.com/jan-g/oci-poledit/edit"
	"github.com/oracle/oci-go-sdk/identity"
)

type policy struct {
	identity.Policy
}

var _ poledit.Handler = &policy{}

func New() *policy {
	return &policy{}
}

func (p policy) Find(ctx context.Context, parent string, name string, iam identity.IdentityClient) (poledit.Handler, error) {
	pols, err := p.List(ctx, parent, iam)
	if err != nil {
		return nil, err
	}
	for _, p := range pols {
		if *p.Name == name {
			return &policy{p}, nil
		}
	}
	return &policy{}, poledit.NotFoundError
}

func (p *policy) List(ctx context.Context, parent string, iam identity.IdentityClient) ([]identity.Policy, error) {
	var page *string = nil
	res := []identity.Policy{}
	for {
		ps, err := iam.ListPolicies(ctx, identity.ListPoliciesRequest{
			CompartmentId: &parent,
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, ps.Items...)
		page = ps.OpcNextPage
		if page == nil {
			break
		}
	}
	return res, nil
}

func (p *policy) Edit() (poledit.Handler, error) {
	lines, err := edit.Edit(p.Statements)
	if err != nil {
		return nil, err
	}

	pp := &policy{p.Policy}
	pp.Statements = lines

	return pp, nil
}

func (p *policy) JsonEdit() (poledit.Handler, error) {
	pol, err := edit.Json(&p.Policy)
	return &policy{*(pol.(*identity.Policy))}, err
}

func (p *policy) WithDescription(description *string) poledit.Handler {
	pp := &policy{p.Policy}
	pp.Description = description
	return pp
}

func (p *policy) WithName(name *string) poledit.Handler {
	pp := &policy{p.Policy}
	pp.Name = name
	return pp
}

func (p *policy) ShouldCreate() bool {
	return len(p.Statements) > 0
}

func (p *policy) DoCreate(iam identity.IdentityClient, compartment identity.Compartment) error {
	_, err := iam.CreatePolicy(context.Background(), identity.CreatePolicyRequest{
		CreatePolicyDetails: identity.CreatePolicyDetails{
			CompartmentId: compartment.Id,
			Name:          p.Name,
			Statements:    p.Statements,
			Description:   p.Description,
			VersionDate:   p.VersionDate,
			FreeformTags:  p.FreeformTags,
			DefinedTags:   p.DefinedTags,
		},
	})
	return err
}

func (p *policy) DoUpdate(iam identity.IdentityClient) error {
	_, err := iam.UpdatePolicy(context.Background(), identity.UpdatePolicyRequest{
		PolicyId: p.Id,
		UpdatePolicyDetails: identity.UpdatePolicyDetails{
			Description:  p.Description,
			Statements:   p.Statements,
			VersionDate:  p.VersionDate,
			FreeformTags: p.FreeformTags,
			DefinedTags:  p.DefinedTags,
		},
	})
	return err
}

func (p *policy) DoDelete(iam identity.IdentityClient) error {
	_, err := iam.DeletePolicy(context.Background(), identity.DeletePolicyRequest{
		PolicyId: p.Id,
	})
	return err
}
