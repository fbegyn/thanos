// Copyright (c) The Thanos Authors.
// Licensed under the Apache License 2.0.

package tenancy

import (
	"net/http"
	"path"

	"github.com/pkg/errors"
)

const (
	// DefaultTenantHeader is the default header used to designate the tenant making a request.
	DefaultTenantHeader = "THANOS-TENANT"
	// DefaultTenant is the default value used for when no tenant is passed via the tenant header.
	DefaultTenant = "default-tenant"
	// DefaultTenantLabel is the default label-name with which the tenant is announced in stored metrics.
	DefaultTenantLabel = "tenant_id"
)

// Allowed fields in client certificates.
const (
	CertificateFieldOrganization       = "organization"
	CertificateFieldOrganizationalUnit = "organizationalUnit"
	CertificateFieldCommonName         = "commonName"
)

func IsTenantValid(tenant string) error {
	if tenant != path.Base(tenant) {
		return errors.New("Tenant name not valid")
	}
	return nil
}

// GetTenantFromHTTP extracts the tenant from a http.Request object.
func GetTenantFromHTTP(r *http.Request, tenantHeader string, defaultTenantID string, certTenantField string) (string, error) {
	var err error
	tenant := r.Header.Get(tenantHeader)
	if tenant == "" {
		tenant = defaultTenantID
	}

	if certTenantField != "" {
		tenant, err = getTenantFromCertificate(r, certTenantField)
		if err != nil {
			// This must hard fail to ensure hard tenancy when feature is enabled.
			return "", err
		}
	}

	err = IsTenantValid(tenant)
	if err != nil {
		return "", err
	}
	return tenant, nil
}

// getTenantFromCertificate extracts the tenant value from a client's presented certificate. The x509 field to use as
// value can be configured with Options.TenantField. An error is returned when the extraction has not succeeded.
func getTenantFromCertificate(r *http.Request, certTenantField string) (string, error) {
	var tenant string

	if len(r.TLS.PeerCertificates) == 0 {
		return "", errors.New("could not get required certificate field from client cert")
	}

	// First cert is the leaf authenticated against.
	cert := r.TLS.PeerCertificates[0]

	switch certTenantField {

	case CertificateFieldOrganization:
		if len(cert.Subject.Organization) == 0 {
			return "", errors.New("could not get organization field from client cert")
		}
		tenant = cert.Subject.Organization[0]

	case CertificateFieldOrganizationalUnit:
		if len(cert.Subject.OrganizationalUnit) == 0 {
			return "", errors.New("could not get organizationalUnit field from client cert")
		}
		tenant = cert.Subject.OrganizationalUnit[0]

	case CertificateFieldCommonName:
		if cert.Subject.CommonName == "" {
			return "", errors.New("could not get commonName field from client cert")
		}
		tenant = cert.Subject.CommonName

	default:
		return "", errors.New("tls client cert field requested is not supported")
	}

	return tenant, nil
}
