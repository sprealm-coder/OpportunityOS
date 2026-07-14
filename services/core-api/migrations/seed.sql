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

INSERT INTO users (id, email, display_name, password_hash, status)
VALUES (
  '00000000-0000-4000-8000-000000000003',
  'admin@opportunity.local',
  'Development Admin',
  '$2a$10$7uJeFJFr1e60KnchZwY1L.Bxy1t.rGKhb75K9NS8Ail2kSMmqM.cy',
  'active'
)
ON CONFLICT (email) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  password_hash = EXCLUDED.password_hash,
  status = EXCLUDED.status;

INSERT INTO memberships (tenant_id, user_id, role)
VALUES (
  '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000003',
  'admin'
)
ON CONFLICT (tenant_id, user_id) DO UPDATE SET role = EXCLUDED.role;

INSERT INTO users (id, email, display_name, password_hash, status)
VALUES (
  '00000000-0000-4000-8000-000000000004',
  'reviewer@opportunity.local',
  'Development Reviewer',
  '$2a$10$7uJeFJFr1e60KnchZwY1L.Bxy1t.rGKhb75K9NS8Ail2kSMmqM.cy',
  'active'
)
ON CONFLICT (email) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  password_hash = EXCLUDED.password_hash,
  status = EXCLUDED.status;

INSERT INTO memberships (tenant_id, user_id, role)
VALUES (
  '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000004',
  'reviewer'
)
ON CONFLICT (tenant_id, user_id) DO UPDATE SET role = EXCLUDED.role;
