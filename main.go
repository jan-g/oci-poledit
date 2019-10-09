package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jan-g/oci-poledit/edit"

	"github.com/mitchellh/go-homedir"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
)

// Usage: poledit compartment-name:compartment-name:policy-name

var (
	config      = flag.String("config", "~/.oci/config", "OCI configuration file")
	profile     = flag.String("profile", "DEFAULT", "profile to use")
	new         = flag.Bool("create", false, "create new policy")
	description = flag.String("description", "", "description for new policy")
)

func main() {
	flag.Parse()
	config, err := homedir.Expand(*config)
	if err != nil {
		panic(err)
	}
	if p, ok := os.LookupEnv("OCI_CLI_PROFILE"); ok && *profile == "DEFAULT" {
		profile = &p
	}
	provider, err := common.ConfigurationProviderFromFileWithProfile(config, *profile, "")
	if err != nil {
		panic(err)
	}
	tenancy, err := provider.TenancyOCID()
	if err != nil {
		panic(err)
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Specify one path to a policy object")
	}
	iam, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		panic(err)
	}

	cp := CompartmentPath(flag.Arg(0))
	compartments := cp[:len(cp)-1]
	policyName := cp[len(cp)-1]

	compartment, err := ChainCompartmentLookup(context.Background(), iam, tenancy, compartments)
	if err != nil {
		panic(err)
	}

	policy, err := FindPolicy(context.Background(), *compartment.Id, policyName, iam)
	if err == nil {
		if *new {
			panic(errors.New("policy already exists"))
		}
		policy, err = editPolicy(policy)
		if err != nil {
			panic(err)
		}
		if *description != "" {
			policy.Description = description
		}

		if len(policy.Statements) > 0 {
			_, err := iam.UpdatePolicy(context.Background(), identity.UpdatePolicyRequest{
				PolicyId: policy.Id,
				UpdatePolicyDetails: identity.UpdatePolicyDetails{
					Description:  policy.Description,
					Statements:   policy.Statements,
					VersionDate:  policy.VersionDate,
					FreeformTags: policy.FreeformTags,
					DefinedTags:  policy.DefinedTags,
				},
			})
			if err != nil {
				panic(err)
			}
		} else {
			// Delete
			_, err := iam.DeletePolicy(context.Background(), identity.DeletePolicyRequest{
				PolicyId: policy.Id,
			})
			if err != nil {
				panic(err)
			}
		}
	} else if err == NotFoundError {
		if !*new {
			panic(errors.New("policy does not exist; use -create"))
		}
		if *description == "" {
			panic(errors.New("use -description to specify a description"))
		}
		policy, err = editPolicy(policy)
		if err != nil {
			panic(err)
		}

		if len(policy.Statements) > 0 {
			_, err := iam.CreatePolicy(context.Background(), identity.CreatePolicyRequest{
				CreatePolicyDetails: identity.CreatePolicyDetails{
					CompartmentId: compartment.Id,
					Name:          &policyName,
					Statements:    policy.Statements,
					Description:   description,
				},
			})
			if err != nil {
				panic(err)
			}
		}
	} else {
		panic(err)
	}
}

func CompartmentPath(path string) []string {
	cp := strings.Split(path, ":")
	for i, c := range cp {
		cp[i] = strings.Trim(c, " ")
	}
	return cp
}

func ChainCompartmentLookup(ctx context.Context, iam identity.IdentityClient, start string, rest []string) (identity.Compartment, error) {
	// Look up the top-level compartment
	t, err := iam.GetCompartment(ctx, identity.GetCompartmentRequest{
		CompartmentId: &start,
	})
	if err != nil {
		return identity.Compartment{}, err
	}

	comp := t.Compartment
	for _, part := range rest {
		c, err := FindCompartment(ctx, *comp.Id, part, iam)
		if err != nil {
			return identity.Compartment{}, err
		}
		comp = c
	}
	return comp, nil
}

var (
	NotFoundError = errors.New("not found")
)

func FindCompartment(ctx context.Context, parent string, name string, iam identity.IdentityClient) (identity.Compartment, error) {
	comps, err := ListCompartments(ctx, parent, iam)
	if err != nil {
		return identity.Compartment{}, err
	}
	for _, c := range comps {
		if *c.Name == name {
			return c, nil
		}
	}
	return identity.Compartment{}, NotFoundError
}

func FindPolicy(ctx context.Context, parent string, name string, iam identity.IdentityClient) (identity.Policy, error) {
	pols, err := ListPolicies(ctx, parent, iam)
	if err != nil {
		return identity.Policy{}, err
	}
	for _, p := range pols {
		if *p.Name == name {
			return p, nil
		}
	}
	return identity.Policy{}, NotFoundError
}

func ListCompartments(ctx context.Context, parent string, iam identity.IdentityClient) ([]identity.Compartment, error) {
	var page *string = nil
	res := []identity.Compartment{}
	for {
		cs, err := iam.ListCompartments(ctx, identity.ListCompartmentsRequest{
			CompartmentId: &parent,
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, cs.Items...)
		page = cs.OpcNextPage
		if page == nil {
			break
		}
	}
	return res, nil
}

func ListPolicies(ctx context.Context, parent string, iam identity.IdentityClient) ([]identity.Policy, error) {
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

func editPolicy(policy identity.Policy) (identity.Policy, error) {
	lines, err := edit.Edit(policy.Statements)
	if err != nil {
		return policy, err
	}

	policy.Statements = lines

	return policy, nil
}
