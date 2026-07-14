package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/capability"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func newUUID(ctx context.Context, tx pgx.Tx) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&id)
	return id, err
}

func scanCapability(row rowScanner) (capability.Capability, error) {
	var item capability.Capability
	var definition []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Description, &definition)
	if err == nil {
		err = json.Unmarshal(definition, &item.Definition)
	}
	return item, err
}

func (s *Store) CreateCapability(ctx context.Context, scope tenancy.Scope, name, description string, definition map[string]any, key string) (capability.Capability, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return capability.Capability{}, platform.Invalid("invalid_name", "capability name is required")
	}
	if definition == nil {
		definition = map[string]any{}
	}
	encoded, err := json.Marshal(definition)
	if err != nil {
		return capability.Capability{}, err
	}
	return runCommand(ctx, s, scope, key, "capability.create", func(tx pgx.Tx) (capability.Capability, string, error) {
		item, err := scanCapability(tx.QueryRow(ctx, `INSERT INTO capabilities (tenant_id,name,description,definition) VALUES($1,$2,$3,$4) RETURNING id,tenant_id,name,description,definition`, scope.TenantID, name, description, encoded))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "capability.create", "capability", item.ID, key, nil); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "capability", item.ID, "capability.created", 1, map[string]any{"name": item.Name}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ListCapabilities(ctx context.Context, scope tenancy.Scope) ([]capability.Capability, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) ([]capability.Capability, error) {
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,name,description,definition FROM capabilities WHERE tenant_id=$1 ORDER BY name,id`, scope.TenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []capability.Capability{}
		for rows.Next() {
			item, err := scanCapability(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, rows.Err()
	})
}

func scanProvider(row rowScanner) (capability.Provider, error) {
	var item capability.Provider
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Status)
	item.Endpoints = []capability.ProviderEndpoint{}
	return item, err
}

func scanProviderEndpoint(row rowScanner) (capability.ProviderEndpoint, error) {
	var item capability.ProviderEndpoint
	err := row.Scan(&item.ID, &item.TenantID, &item.ProviderID, &item.CapabilityID, &item.AdapterType, &item.AdapterVersion, &item.Status)
	return item, err
}

func (s *Store) CreateProvider(ctx context.Context, scope tenancy.Scope, name, key string) (capability.Provider, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return capability.Provider{}, platform.Invalid("invalid_name", "provider name is required")
	}
	return runCommand(ctx, s, scope, key, "provider.create", func(tx pgx.Tx) (capability.Provider, string, error) {
		item, err := scanProvider(tx.QueryRow(ctx, `INSERT INTO providers (tenant_id,name,status) VALUES($1,$2,'active') RETURNING id,tenant_id,name,status`, scope.TenantID, name))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "provider.create", "provider", item.ID, key, nil); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "provider", item.ID, "provider.created", 1, map[string]any{"name": item.Name}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func loadProviderEndpoints(ctx context.Context, tx pgx.Tx, tenantID, providerID string) ([]capability.ProviderEndpoint, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,provider_id,capability_id,adapter_type,adapter_version,status FROM provider_endpoints WHERE tenant_id=$1 AND provider_id=$2 ORDER BY id`, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []capability.ProviderEndpoint{}
	for rows.Next() {
		item, err := scanProviderEndpoint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListProviders(ctx context.Context, scope tenancy.Scope) ([]capability.Provider, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) ([]capability.Provider, error) {
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,name,status FROM providers WHERE tenant_id=$1 ORDER BY name,id`, scope.TenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []capability.Provider{}
		for rows.Next() {
			item, err := scanProvider(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if err = rows.Err(); err != nil {
			return nil, err
		}
		rows.Close()
		for index := range items {
			items[index].Endpoints, err = loadProviderEndpoints(ctx, tx, scope.TenantID, items[index].ID)
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	})
}

func (s *Store) CreateProviderEndpoint(ctx context.Context, scope tenancy.Scope, providerID, capabilityID, adapterType, adapterVersion, key string) (capability.ProviderEndpoint, error) {
	adapterType, adapterVersion = strings.TrimSpace(adapterType), strings.TrimSpace(adapterVersion)
	if adapterType == "" || adapterVersion == "" {
		return capability.ProviderEndpoint{}, platform.Invalid("invalid_adapter", "adapter type and version are required")
	}
	return runCommand(ctx, s, scope, key, "provider.endpoint.create", func(tx pgx.Tx) (capability.ProviderEndpoint, string, error) {
		item, err := scanProviderEndpoint(tx.QueryRow(ctx, `INSERT INTO provider_endpoints (tenant_id,provider_id,capability_id,adapter_type,adapter_version,status) VALUES($1,$2,$3,$4,$5,'healthy') RETURNING id,tenant_id,provider_id,capability_id,adapter_type,adapter_version,status`, scope.TenantID, providerID, capabilityID, adapterType, adapterVersion))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "provider.endpoint.create", "provider_endpoint", item.ID, key, map[string]any{"provider_id": providerID, "capability_id": capabilityID}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "provider_endpoint", item.ID, "provider.endpoint_created", 1, map[string]any{"provider_id": providerID, "capability_id": capabilityID}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanProduct(row rowScanner) (catalog.Product, error) {
	var item catalog.Product
	err := row.Scan(&item.ID, &item.TenantID, &item.BlueprintID, &item.Name, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) CreateProduct(ctx context.Context, scope tenancy.Scope, blueprintID, name, key string) (catalog.Product, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return catalog.Product{}, platform.Invalid("invalid_name", "product name is required")
	}
	return runCommand(ctx, s, scope, key, "product.create", func(tx pgx.Tx) (catalog.Product, string, error) {
		var blueprintStatus string
		if err := tx.QueryRow(ctx, `SELECT status FROM business_blueprints WHERE tenant_id=$1 AND id=$2`, scope.TenantID, blueprintID).Scan(&blueprintStatus); err != nil {
			return catalog.Product{}, "", err
		}
		allowed := map[string]bool{"approved": true, "configuring": true, "ready": true, "launched": true}
		if !allowed[blueprintStatus] {
			return catalog.Product{}, "", platform.Invalid("product_blueprint_not_ready", "blueprint must be approved before product creation")
		}
		item, err := scanProduct(tx.QueryRow(ctx, `INSERT INTO products (tenant_id,blueprint_id,name,status,version,created_by) VALUES($1,$2,$3,'draft',1,$4) RETURNING id,tenant_id,blueprint_id,name,status,version,created_by,created_at,updated_at`, scope.TenantID, blueprintID, name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "product.create", "product", item.ID, key, map[string]any{"blueprint_id": blueprintID}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "product", item.ID, "product.created", item.Version, map[string]any{"blueprint_id": blueprintID}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ListProducts(ctx context.Context, scope tenancy.Scope) ([]catalog.Product, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) ([]catalog.Product, error) {
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,blueprint_id,name,status,version,created_by,created_at,updated_at FROM products WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scope.TenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []catalog.Product{}
		for rows.Next() {
			item, err := scanProduct(rows)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, rows.Err()
	})
}

func scanProductVersion(row rowScanner) (catalog.ProductVersion, error) {
	var item catalog.ProductVersion
	var definition, inputSchema, outputSchema []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ProductID, &item.Version, &definition, &inputSchema, &outputSchema, &item.CreatedBy, &item.CreatedAt)
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(definition, &item); err != nil {
		return item, err
	}
	if err = json.Unmarshal(inputSchema, &item.InputSchema); err != nil {
		return item, err
	}
	if err = json.Unmarshal(outputSchema, &item.OutputSchema); err != nil {
		return item, err
	}
	return item, nil
}

func loadProductVersion(ctx context.Context, tx pgx.Tx, tenantID, productID, versionID string) (catalog.ProductVersion, error) {
	return scanProductVersion(tx.QueryRow(ctx, `SELECT id,tenant_id,product_id,version,definition,input_schema,output_schema,created_by,created_at FROM product_versions WHERE tenant_id=$1 AND product_id=$2 AND id=$3`, tenantID, productID, versionID))
}

func loadProductVersions(ctx context.Context, tx pgx.Tx, tenantID, productID string) ([]catalog.ProductVersion, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,product_id,version,definition,input_schema,output_schema,created_by,created_at FROM product_versions WHERE tenant_id=$1 AND product_id=$2 ORDER BY version,id`, tenantID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []catalog.ProductVersion{}
	for rows.Next() {
		item, err := scanProductVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSKU(row rowScanner) (catalog.SKU, error) {
	var item catalog.SKU
	err := row.Scan(&item.ID, &item.TenantID, &item.ProductID, &item.Code, &item.Name, &item.Status, &item.CreatedBy, &item.CreatedAt)
	return item, err
}

func scanSKUVersion(row rowScanner) (catalog.SKUVersion, error) {
	var item catalog.SKUVersion
	var bindings []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.SKUID, &item.ProductVersionID, &item.Version, &bindings, &item.CreatedBy, &item.CreatedAt)
	if err == nil {
		err = json.Unmarshal(bindings, &item)
	}
	return item, err
}

func loadSKUs(ctx context.Context, tx pgx.Tx, tenantID, productID string) ([]catalog.SKUDetail, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,product_id,code,name,status,created_by,created_at FROM skus WHERE tenant_id=$1 AND product_id=$2 ORDER BY code,id`, tenantID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	skus := []catalog.SKU{}
	for rows.Next() {
		sku, err := scanSKU(rows)
		if err != nil {
			return nil, err
		}
		skus = append(skus, sku)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	items := make([]catalog.SKUDetail, 0, len(skus))
	for _, sku := range skus {
		versionRows, err := tx.Query(ctx, `SELECT id,tenant_id,sku_id,product_version_id,version,bindings,created_by,created_at FROM sku_versions WHERE tenant_id=$1 AND sku_id=$2 ORDER BY version,id`, tenantID, sku.ID)
		if err != nil {
			return nil, err
		}
		versions := []catalog.SKUVersion{}
		for versionRows.Next() {
			version, err := scanSKUVersion(versionRows)
			if err != nil {
				versionRows.Close()
				return nil, err
			}
			versions = append(versions, version)
		}
		err = versionRows.Err()
		versionRows.Close()
		if err != nil {
			return nil, err
		}
		items = append(items, catalog.SKUDetail{SKU: sku, Versions: versions})
	}
	return items, nil
}

func loadPublications(ctx context.Context, tx pgx.Tx, tenantID, productID string) ([]catalog.Publication, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,product_id,product_version_id,status,published_by,published_at FROM publications WHERE tenant_id=$1 AND product_id=$2 ORDER BY published_at,id`, tenantID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []catalog.Publication{}
	for rows.Next() {
		var item catalog.Publication
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ProductID, &item.ProductVersionID, &item.Status, &item.PublishedBy, &item.PublishedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getProductDetail(ctx context.Context, tx pgx.Tx, tenantID, productID string) (catalog.ProductDetail, error) {
	product, err := scanProduct(tx.QueryRow(ctx, `SELECT id,tenant_id,blueprint_id,name,status,version,created_by,created_at,updated_at FROM products WHERE tenant_id=$1 AND id=$2`, tenantID, productID))
	if err != nil {
		return catalog.ProductDetail{}, err
	}
	versions, err := loadProductVersions(ctx, tx, tenantID, productID)
	if err != nil {
		return catalog.ProductDetail{}, err
	}
	skus, err := loadSKUs(ctx, tx, tenantID, productID)
	if err != nil {
		return catalog.ProductDetail{}, err
	}
	publications, err := loadPublications(ctx, tx, tenantID, productID)
	if err != nil {
		return catalog.ProductDetail{}, err
	}
	return catalog.ProductDetail{Product: product, Versions: versions, SKUs: skus, Publications: publications}, nil
}

func (s *Store) GetProduct(ctx context.Context, scope tenancy.Scope, id string) (catalog.ProductDetail, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (catalog.ProductDetail, error) {
		return getProductDetail(ctx, tx, scope.TenantID, id)
	})
}

func (s *Store) CreateProductVersion(ctx context.Context, scope tenancy.Scope, productID string, input application.ProductVersionInput, key string) (catalog.ProductVersion, error) {
	normalized, err := application.NormalizeProductVersionInput(scope, input)
	if err != nil {
		return catalog.ProductVersion{}, err
	}
	return runCommand(ctx, s, scope, key, "product.version.create", func(tx pgx.Tx) (catalog.ProductVersion, string, error) {
		product, err := scanProduct(tx.QueryRow(ctx, `SELECT id,tenant_id,blueprint_id,name,status,version,created_by,created_at,updated_at FROM products WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, productID))
		if err != nil {
			return catalog.ProductVersion{}, "", err
		}
		if product.Status == "retired" {
			return catalog.ProductVersion{}, "", platform.Invalid("product_retired", "retired product cannot receive a new version")
		}
		var versionNumber int
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM product_versions WHERE tenant_id=$1 AND product_id=$2`, scope.TenantID, productID).Scan(&versionNumber); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		versionID, err := newUUID(ctx, tx)
		if err != nil {
			return catalog.ProductVersion{}, "", err
		}
		ids := make([]string, 5)
		for index := range ids {
			ids[index], err = newUUID(ctx, tx)
			if err != nil {
				return catalog.ProductVersion{}, "", err
			}
		}
		workflowID, meteringID, priceBookID, routePolicyID, complianceID := ids[0], ids[1], ids[2], ids[3], ids[4]
		normalized.Workflow.ID, normalized.Workflow.TenantID = workflowID, scope.TenantID
		if normalized.Workflow.Name == "" {
			normalized.Workflow.Name = product.Name + " Workflow"
		}
		normalized.Metering.ID, normalized.Metering.TenantID = meteringID, scope.TenantID
		if normalized.Metering.Name == "" {
			normalized.Metering.Name = product.Name + " Meter"
		}
		normalized.PriceBook.ID, normalized.PriceBook.TenantID = priceBookID, scope.TenantID
		normalized.RoutePolicy.ID, normalized.RoutePolicy.TenantID = routePolicyID, scope.TenantID
		if normalized.RoutePolicy.Name == "" {
			normalized.RoutePolicy.Name = product.Name + " Route"
		}
		workflowJSON, _ := json.Marshal(normalized.Workflow)
		meteringJSON, _ := json.Marshal(normalized.Metering)
		routeJSON, _ := json.Marshal(normalized.RoutePolicy)
		if _, err = tx.Exec(ctx, `INSERT INTO workflow_definitions (id,tenant_id,name,version,definition) VALUES($1,$2,$3,$4,$5)`, workflowID, scope.TenantID, normalized.Workflow.Name, normalized.Workflow.Version, workflowJSON); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO metering_definitions (id,tenant_id,name,version,definition) VALUES($1,$2,$3,$4,$5)`, meteringID, scope.TenantID, normalized.Metering.Name, normalized.Metering.Version, meteringJSON); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO price_books (id,tenant_id,currency,version,status) VALUES($1,$2,$3,$4,'draft')`, priceBookID, scope.TenantID, strings.ToUpper(normalized.PriceBook.Currency), normalized.PriceBook.Version); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		for index, rule := range normalized.PriceBook.Rules {
			ruleID, uuidErr := newUUID(ctx, tx)
			if uuidErr != nil {
				return catalog.ProductVersion{}, "", uuidErr
			}
			rule.ID = ruleID
			ruleJSON, _ := json.Marshal(rule)
			if _, err = tx.Exec(ctx, `INSERT INTO price_rules (id,tenant_id,price_book_id,priority,definition) VALUES($1,$2,$3,$4,$5)`, ruleID, scope.TenantID, priceBookID, index, ruleJSON); err != nil {
				return catalog.ProductVersion{}, "", err
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO route_policies (id,tenant_id,name,version,definition) VALUES($1,$2,$3,$4,$5)`, routePolicyID, scope.TenantID, normalized.RoutePolicy.Name, normalized.RoutePolicy.Version, routeJSON); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		version := catalog.ProductVersion{ID: versionID, TenantID: scope.TenantID, ProductID: productID, Version: versionNumber, InputSchema: normalized.InputSchema, OutputSchema: normalized.OutputSchema, FormSchema: normalized.FormSchema, CapabilityIDs: normalized.CapabilityIDs, Workflow: normalized.Workflow, MeteringID: meteringID, PriceBookID: priceBookID, RoutePolicyID: routePolicyID, DeliveryMode: normalized.DeliveryMode, ComplianceProfileID: complianceID, ComplianceProfile: normalized.ComplianceProfile, GrowthPlaybook: normalized.GrowthPlaybook, CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC()}
		definitionJSON, _ := json.Marshal(version)
		inputJSON, _ := json.Marshal(version.InputSchema)
		outputJSON, _ := json.Marshal(version.OutputSchema)
		if err = tx.QueryRow(ctx, `INSERT INTO product_versions (id,tenant_id,product_id,version,definition,input_schema,output_schema,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING created_at`, versionID, scope.TenantID, productID, versionNumber, definitionJSON, inputJSON, outputJSON, scope.ActorID).Scan(&version.CreatedAt); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		for _, capabilityID := range normalized.CapabilityIDs {
			if _, err = tx.Exec(ctx, `INSERT INTO product_capability_bindings (tenant_id,product_version_id,capability_id,required) VALUES($1,$2,$3,true)`, scope.TenantID, versionID, capabilityID); err != nil {
				return catalog.ProductVersion{}, "", err
			}
		}
		formJSON, _ := json.Marshal(version.FormSchema)
		growthJSON, _ := json.Marshal(version.GrowthPlaybook)
		complianceJSON, _ := json.Marshal(version.ComplianceProfile)
		bindingStatements := []struct {
			query string
			value any
		}{
			{`INSERT INTO product_workflow_bindings (tenant_id,product_version_id,workflow_definition_id) VALUES($1,$2,$3)`, workflowID},
			{`INSERT INTO product_metering_bindings (tenant_id,product_version_id,metering_definition_id) VALUES($1,$2,$3)`, meteringID},
			{`INSERT INTO product_pricing_bindings (tenant_id,product_version_id,price_book_id) VALUES($1,$2,$3)`, priceBookID},
			{`INSERT INTO product_routing_bindings (tenant_id,product_version_id,route_policy_id) VALUES($1,$2,$3)`, routePolicyID},
			{`INSERT INTO product_form_definitions (tenant_id,product_version_id,form_schema) VALUES($1,$2,$3)`, formJSON},
			{`INSERT INTO product_output_definitions (tenant_id,product_version_id,output_schema) VALUES($1,$2,$3)`, outputJSON},
			{`INSERT INTO product_growth_bindings (tenant_id,product_version_id,growth_playbook) VALUES($1,$2,$3)`, growthJSON},
			{`INSERT INTO product_compliance_bindings (id,tenant_id,product_version_id,compliance_profile) VALUES($3,$1,$2,$4)`, complianceID},
		}
		for index, binding := range bindingStatements {
			if index == len(bindingStatements)-1 {
				_, err = tx.Exec(ctx, binding.query, scope.TenantID, versionID, binding.value, complianceJSON)
			} else {
				_, err = tx.Exec(ctx, binding.query, scope.TenantID, versionID, binding.value)
			}
			if err != nil {
				return catalog.ProductVersion{}, "", err
			}
		}
		product.Version++
		if _, err = tx.Exec(ctx, `UPDATE products SET version=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, productID, product.Version); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		if err = appendAudit(ctx, tx, scope, "product.version.create", "product_version", version.ID, key, map[string]any{"product_id": productID, "version": versionNumber}); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		if err = appendEvent(ctx, tx, scope, "product", productID, "product.version_created", product.Version, map[string]any{"product_version_id": version.ID, "version": versionNumber}); err != nil {
			return catalog.ProductVersion{}, "", err
		}
		return version, version.ID, nil
	})
}

func (s *Store) CreateSKU(ctx context.Context, scope tenancy.Scope, productID, code, name, key string) (catalog.SKU, error) {
	code, name = strings.ToUpper(strings.TrimSpace(code)), strings.TrimSpace(name)
	if code == "" || name == "" {
		return catalog.SKU{}, platform.Invalid("invalid_sku", "SKU code and name are required")
	}
	return runCommand(ctx, s, scope, key, "sku.create", func(tx pgx.Tx) (catalog.SKU, string, error) {
		item, err := scanSKU(tx.QueryRow(ctx, `INSERT INTO skus (tenant_id,product_id,code,name,status,created_by) VALUES($1,$2,$3,$4,'draft',$5) RETURNING id,tenant_id,product_id,code,name,status,created_by,created_at`, scope.TenantID, productID, code, name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "sku.create", "sku", item.ID, key, map[string]any{"product_id": productID}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "sku", item.ID, "sku.created", 1, map[string]any{"product_id": productID}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateSKUVersion(ctx context.Context, scope tenancy.Scope, skuID string, input application.SKUVersionInput, key string) (catalog.SKUVersion, error) {
	if input.ProductVersionID == "" {
		return catalog.SKUVersion{}, platform.Invalid("invalid_reference", "product version is required")
	}
	if input.Entitlements == nil {
		input.Entitlements = map[string]any{}
	}
	return runCommand(ctx, s, scope, key, "sku.version.create", func(tx pgx.Tx) (catalog.SKUVersion, string, error) {
		var productID string
		if err := tx.QueryRow(ctx, `SELECT product_id FROM skus WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, skuID).Scan(&productID); err != nil {
			return catalog.SKUVersion{}, "", err
		}
		version, err := loadProductVersion(ctx, tx, scope.TenantID, productID, input.ProductVersionID)
		if err != nil {
			return catalog.SKUVersion{}, "", err
		}
		var versionNumber int
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM sku_versions WHERE tenant_id=$1 AND sku_id=$2`, scope.TenantID, skuID).Scan(&versionNumber); err != nil {
			return catalog.SKUVersion{}, "", err
		}
		item := catalog.SKUVersion{TenantID: scope.TenantID, SKUID: skuID, ProductVersionID: version.ID, WorkflowVersionID: version.Workflow.ID, MeteringVersionID: version.MeteringID, PricingVersionID: version.PriceBookID, RoutingVersionID: version.RoutePolicyID, Version: versionNumber, Entitlements: input.Entitlements, CreatedBy: scope.ActorID}
		bindings, _ := json.Marshal(item)
		err = tx.QueryRow(ctx, `INSERT INTO sku_versions (tenant_id,sku_id,product_version_id,version,bindings,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,created_at`, scope.TenantID, skuID, version.ID, versionNumber, bindings, scope.ActorID).Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return item, "", err
		}
		bindings, _ = json.Marshal(item)
		if _, err = tx.Exec(ctx, `UPDATE sku_versions SET bindings=$3 WHERE tenant_id=$1 AND id=$2`, scope.TenantID, item.ID, bindings); err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "sku.version.create", "sku_version", item.ID, key, map[string]any{"product_version_id": version.ID, "version": versionNumber}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "sku", skuID, "sku.version_created", versionNumber, map[string]any{"sku_version_id": item.ID, "product_version_id": version.ID}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) PublishProduct(ctx context.Context, scope tenancy.Scope, productID, productVersionID, key string) (catalog.Publication, error) {
	return runCommand(ctx, s, scope, key, "product.publish", func(tx pgx.Tx) (catalog.Publication, string, error) {
		product, err := scanProduct(tx.QueryRow(ctx, `SELECT id,tenant_id,blueprint_id,name,status,version,created_by,created_at,updated_at FROM products WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, productID))
		if err != nil {
			return catalog.Publication{}, "", err
		}
		version, err := loadProductVersion(ctx, tx, scope.TenantID, productID, productVersionID)
		if err != nil {
			return catalog.Publication{}, "", err
		}
		providers := map[string]bool{}
		for _, capabilityID := range version.CapabilityIDs {
			var available bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM provider_endpoints endpoint JOIN providers provider ON provider.id=endpoint.provider_id AND provider.tenant_id=endpoint.tenant_id WHERE endpoint.tenant_id=$1 AND endpoint.capability_id=$2 AND endpoint.status='healthy' AND provider.status='active')`, scope.TenantID, capabilityID).Scan(&available); err != nil {
				return catalog.Publication{}, "", err
			}
			providers[capabilityID] = available
		}
		var skuReady bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sku_versions version JOIN skus sku ON sku.id=version.sku_id AND sku.tenant_id=version.tenant_id WHERE version.tenant_id=$1 AND sku.product_id=$2 AND version.product_version_id=$3)`, scope.TenantID, productID, productVersionID).Scan(&skuReady); err != nil {
			return catalog.Publication{}, "", err
		}
		if !skuReady {
			return catalog.Publication{}, "", platform.Invalid("publication_not_ready", "at least one SKU version must bind the product version")
		}
		publication, err := catalog.Publish(&product, version, providers)
		if err != nil {
			return catalog.Publication{}, "", platform.Invalid("publication_not_ready", err.Error())
		}
		publication.PublishedBy = scope.ActorID
		if err = tx.QueryRow(ctx, `INSERT INTO publications (tenant_id,product_id,product_version_id,status,published_by) VALUES($1,$2,$3,'published',$4) RETURNING id,published_at`, scope.TenantID, productID, productVersionID, scope.ActorID).Scan(&publication.ID, &publication.PublishedAt); err != nil {
			return catalog.Publication{}, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE products SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, productID, product.Status, product.Version); err != nil {
			return catalog.Publication{}, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE price_books SET status='active' WHERE tenant_id=$1 AND id=$2`, scope.TenantID, version.PriceBookID); err != nil {
			return catalog.Publication{}, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE skus SET status='published' WHERE tenant_id=$1 AND product_id=$2 AND EXISTS(SELECT 1 FROM sku_versions WHERE sku_versions.tenant_id=$1 AND sku_versions.sku_id=skus.id AND sku_versions.product_version_id=$3)`, scope.TenantID, productID, productVersionID); err != nil {
			return catalog.Publication{}, "", err
		}
		if err = appendAudit(ctx, tx, scope, "product.publish", "publication", publication.ID, key, map[string]any{"product_id": productID, "product_version_id": productVersionID}); err != nil {
			return catalog.Publication{}, "", err
		}
		if err = appendEvent(ctx, tx, scope, "product", productID, "product.published", product.Version, map[string]any{"publication_id": publication.ID, "product_version_id": productVersionID}); err != nil {
			return catalog.Publication{}, "", err
		}
		return publication, publication.ID, nil
	})
}
