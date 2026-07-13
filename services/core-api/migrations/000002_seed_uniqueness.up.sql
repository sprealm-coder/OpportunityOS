CREATE UNIQUE INDEX IF NOT EXISTS tenants_name_unique ON tenants(name);
CREATE UNIQUE INDEX IF NOT EXISTS brands_tenant_name_unique ON brands(tenant_id,name);

