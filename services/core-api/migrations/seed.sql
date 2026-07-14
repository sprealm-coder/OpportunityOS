WITH tenant AS (
  INSERT INTO tenants (name) VALUES ('Test Tenant') ON CONFLICT DO NOTHING RETURNING id
), selected_tenant AS (
  SELECT id FROM tenant UNION ALL SELECT id FROM tenants WHERE name='Test Tenant' LIMIT 1
), brand AS (
  INSERT INTO brands (tenant_id,name) SELECT id,'Test Brand' FROM selected_tenant ON CONFLICT DO NOTHING RETURNING id
), capability AS (
  INSERT INTO capabilities (tenant_id,name,definition) SELECT id,'Test Capability','{"fixture":true}'::jsonb FROM selected_tenant ON CONFLICT DO NOTHING RETURNING id
), provider AS (
  INSERT INTO providers (tenant_id,name,status) SELECT id,'Test Provider','active' FROM selected_tenant ON CONFLICT DO NOTHING RETURNING id
)
INSERT INTO feature_definitions(key,description,default_enabled) VALUES
('growth.proof','Generic proof workflows',true),
('marketplace.internal','Internal marketplace review flow',false),
('finance.settlement','Provider and reseller settlement',false)
ON CONFLICT (key) DO NOTHING;

INSERT INTO tenants (id, name)
VALUES ('00000000-0000-4000-8000-000000000001', 'OpportunityOS Development')
ON CONFLICT DO NOTHING;

INSERT INTO brands (id, tenant_id, name)
VALUES (
  '00000000-0000-4000-8000-000000000002',
  '00000000-0000-4000-8000-000000000001',
  'Development Brand'
)
ON CONFLICT DO NOTHING;
