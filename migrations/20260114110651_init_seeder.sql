-- +goose Up
-- +goose StatementBegin

-- =================================================================
-- 1. UTAMA: BAPPENAS, JICA, MRT (INITIATORS)
-- =================================================================

-- BAPPENAS (GOV_SUPERVISOR)
INSERT INTO organizations (id, name, logo_url, role) VALUES 
('11111111-1111-1111-1111-111111111111', 'Badan Perencanaan Pembangunan Nasional (Bappenas)', 'https://res.cloudinary.com/demo/image/upload/v1/bappenas_logo.png', 'GOV_SUPERVISOR');

INSERT INTO users (id, name, email, password, image_url) VALUES 
('22222222-2222-2222-2222-222222222222', 'Super Admin Bappenas', 'admin@bappenas.go.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.', 'https://res.cloudinary.com/demo/image/upload/v1/user_avatar.png');

INSERT INTO organization_members (id, role, user_id, organization_id) VALUES 
('33333333-3333-3333-3333-333333333333', 'ADMIN', '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111');

-- JICA (LOAN_PROVIDER)
INSERT INTO organizations (id, name, role) VALUES 
('44444444-4444-4444-4444-444444444444', 'Japan International Cooperation Agency (JICA)', 'LOAN_PROVIDER');

INSERT INTO users (id, name, email, password) VALUES 
('66666666-6666-6666-6666-666666666666', 'Admin JICA', 'admin@jica.go.jp', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');

INSERT INTO organization_members (id, role, user_id, organization_id) VALUES 
('88888888-8888-8888-8888-888888888888', 'ADMIN', '66666666-6666-6666-6666-666666666666', '44444444-4444-4444-4444-444444444444');

-- PT MRT JAKARTA (EXECUTING_AGENCY)
INSERT INTO organizations (id, name, role) VALUES 
('55555555-5555-5555-5555-555555555555', 'PT MRT Jakarta (Perseroda)', 'EXECUTING_AGENCY');

INSERT INTO users (id, name, email, password) VALUES 
('77777777-7777-7777-7777-777777777777', 'Admin MRT', 'admin@jakartamrt.co.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');

INSERT INTO organization_members (id, role, user_id, organization_id) VALUES 
('99999999-9999-9999-9999-999999999999', 'ADMIN', '77777777-7777-7777-7777-777777777777', '55555555-5555-5555-5555-555555555555');


-- =================================================================
-- 2. CONTRACTORS (Pelaksana Konstruksi) - ID Prefix: cc...
-- =================================================================

-- Contractor 1: Shimizu - Adhi Karya JO (Main CP201)
INSERT INTO organizations (id, name, role) VALUES ('cc000000-cccc-cccc-cccc-000000000001', 'Shimizu - Adhi Karya JO', 'CONTRACTOR');
INSERT INTO users (id, name, email, password) VALUES ('cc000000-cccc-cccc-cccc-000000000002', 'Admin Shimizu', 'admin@shimizu.co.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('cc000000-cccc-cccc-cccc-000000000003', 'ADMIN', 'cc000000-cccc-cccc-cccc-000000000002', 'cc000000-cccc-cccc-cccc-000000000001');

-- Contractor 2: Sumitomo Mitsui Construction Co. (CP203)
INSERT INTO organizations (id, name, role) VALUES ('cc000000-cccc-cccc-cccc-000000000004', 'Sumitomo Mitsui Construction Co.', 'CONTRACTOR');
INSERT INTO users (id, name, email, password) VALUES ('cc000000-cccc-cccc-cccc-000000000005', 'Admin Sumitomo', 'admin@sumitomo.co.jp', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('cc000000-cccc-cccc-cccc-000000000006', 'ADMIN', 'cc000000-cccc-cccc-cccc-000000000005', 'cc000000-cccc-cccc-cccc-000000000004');

-- Contractor 3: PT Waskita Karya (Persero) Tbk (Local Partner)
INSERT INTO organizations (id, name, role) VALUES ('cc000000-cccc-cccc-cccc-000000000007', 'PT Waskita Karya (Persero) Tbk', 'CONTRACTOR');
INSERT INTO users (id, name, email, password) VALUES ('cc000000-cccc-cccc-cccc-000000000008', 'Admin Waskita', 'admin@waskita.co.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('cc000000-cccc-cccc-cccc-000000000009', 'ADMIN', 'cc000000-cccc-cccc-cccc-000000000008', 'cc000000-cccc-cccc-cccc-000000000007');


-- =================================================================
-- 3. INDEPENDENT AUDITORS (Pengawas) - ID Prefix: aa...
-- =================================================================

-- Auditor 1: Nippon Koei Co., Ltd.
INSERT INTO organizations (id, name, role) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000001', 'Nippon Koei Co., Ltd.', 'INDEPENDENT_AUDITOR');
INSERT INTO users (id, name, email, password) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000002', 'Admin Nippon Koei', 'admin@nipponkoei.co.jp', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000003', 'ADMIN', 'aa000000-aaaa-aaaa-aaaa-000000000002', 'aa000000-aaaa-aaaa-aaaa-000000000001');

-- Auditor 2: Oriental Consultants Global
INSERT INTO organizations (id, name, role) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000004', 'Oriental Consultants Global', 'INDEPENDENT_AUDITOR');
INSERT INTO users (id, name, email, password) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000005', 'Admin OCG', 'admin@ocg.jp', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000006', 'ADMIN', 'aa000000-aaaa-aaaa-aaaa-000000000005', 'aa000000-aaaa-aaaa-aaaa-000000000004');

-- Auditor 3: PT Virama Karya (Persero)
INSERT INTO organizations (id, name, role) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000007', 'PT Virama Karya (Persero)', 'INDEPENDENT_AUDITOR');
INSERT INTO users (id, name, email, password) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000008', 'Admin Virama', 'admin@viramakarya.co.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('aa000000-aaaa-aaaa-aaaa-000000000009', 'ADMIN', 'aa000000-aaaa-aaaa-aaaa-000000000008', 'aa000000-aaaa-aaaa-aaaa-000000000007');


-- =================================================================
-- 4. GOV PARTNERS (Instansi Terkait) - ID Prefix: ee...
-- =================================================================

-- Gov Partner 1: BPN Jakarta Pusat (Urusan Lahan)
INSERT INTO organizations (id, name, role) VALUES ('ee000000-eeee-eeee-eeee-000000000001', 'Kantor Pertanahan (BPN) Jakarta Pusat', 'GOV_PARTNER');
INSERT INTO users (id, name, email, password) VALUES ('ee000000-eeee-eeee-eeee-000000000002', 'Admin BPN Jakpus', 'admin@bpn.jakpus.go.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('ee000000-eeee-eeee-eeee-000000000003', 'ADMIN', 'ee000000-eeee-eeee-eeee-000000000002', 'ee000000-eeee-eeee-eeee-000000000001');

-- Gov Partner 2: PLN UID Jakarta Raya (Urusan Utilitas Listrik)
INSERT INTO organizations (id, name, role) VALUES ('ee000000-eeee-eeee-eeee-000000000004', 'PLN UID Jakarta Raya', 'GOV_PARTNER');
INSERT INTO users (id, name, email, password) VALUES ('ee000000-eeee-eeee-eeee-000000000005', 'Admin PLN', 'admin@pln.co.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('ee000000-eeee-eeee-eeee-000000000006', 'ADMIN', 'ee000000-eeee-eeee-eeee-000000000005', 'ee000000-eeee-eeee-eeee-000000000004');

-- Gov Partner 3: Dinas Perhubungan DKI Jakarta (Urusan Rekayasa Lalin)
INSERT INTO organizations (id, name, role) VALUES ('ee000000-eeee-eeee-eeee-000000000007', 'Dinas Perhubungan DKI Jakarta', 'GOV_PARTNER');
INSERT INTO users (id, name, email, password) VALUES ('ee000000-eeee-eeee-eeee-000000000008', 'Admin Dishub', 'admin@dishub.dki.go.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES ('ee000000-eeee-eeee-eeee-000000000009', 'ADMIN', 'ee000000-eeee-eeee-eeee-000000000008', 'ee000000-eeee-eeee-eeee-000000000007');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 1. DELETE GOV PARTNERS
DELETE FROM organization_members WHERE id LIKE 'ee000000%';
DELETE FROM users WHERE id LIKE 'ee000000%';
DELETE FROM organizations WHERE id LIKE 'ee000000%';

-- 2. DELETE AUDITORS
DELETE FROM organization_members WHERE id LIKE 'aa000000%';
DELETE FROM users WHERE id LIKE 'aa000000%';
DELETE FROM organizations WHERE id LIKE 'aa000000%';

-- 3. DELETE CONTRACTORS
DELETE FROM organization_members WHERE id LIKE 'cc000000%';
DELETE FROM users WHERE id LIKE 'cc000000%';
DELETE FROM organizations WHERE id LIKE 'cc000000%';

-- 4. DELETE INITIAL DATA
DELETE FROM organization_members WHERE id IN ('33333333-3333-3333-3333-333333333333', '88888888-8888-8888-8888-888888888888', '99999999-9999-9999-9999-999999999999');
DELETE FROM users WHERE id IN ('22222222-2222-2222-2222-222222222222', '66666666-6666-6666-6666-666666666666', '77777777-7777-7777-7777-777777777777');
DELETE FROM organizations WHERE id IN ('11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444444', '55555555-5555-5555-5555-555555555555');

-- +goose StatementEnd