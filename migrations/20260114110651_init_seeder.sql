-- +goose Up
-- +goose StatementBegin

-- 1. Insert Organization (BAPPENAS - GOV_SUPERVISOR)
-- ID: 11111111-1111-1111-1111-111111111111
INSERT INTO organizations (id, name, logo_url, role)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Badan Perencanaan Pembangunan Nasional (Bappenas)',
    'https://res.cloudinary.com/demo/image/upload/v1/bappenas_logo.png',
    'GOV_SUPERVISOR'
);

-- 2. Insert User (ADMIN BAPPENAS)
-- ID: 22222222-2222-2222-2222-222222222222
-- Password: "12345678" (Bcrypt Cost 10)
INSERT INTO users (id, name, email, password, image_url)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Super Admin Bappenas',
    'admin@bappenas.go.id',
    '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.', 
    'https://res.cloudinary.com/demo/image/upload/v1/user_avatar.png'
);

-- 3. Insert Organization Member (Link User to Org)
-- ID: 33333333-3333-3333-3333-333333333333
INSERT INTO organization_members (id, role, user_id, organization_id)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    'ADMIN',
    '22222222-2222-2222-2222-222222222222', -- User ID (Admin Bappenas)
    '11111111-1111-1111-1111-111111111111'  -- Org ID (Bappenas)
);

-- OPTIONAL: SEEDER UNTUK JICA & MRT (Supaya bisa langsung Create Project)
-- Uncomment jika ingin data ini langsung ada.


-- JICA (LOAN_PROVIDER)
INSERT INTO organizations (id, name, role) VALUES 
('44444444-4444-4444-4444-444444444444', 'Japan International Cooperation Agency (JICA)', 'LOAN_PROVIDER');

-- PT MRT JAKARTA (EXECUTING_AGENCY)
INSERT INTO organizations (id, name, role) VALUES 
('55555555-5555-5555-5555-555555555555', 'PT MRT Jakarta (Perseroda)', 'EXECUTING_AGENCY');

-- Admin JICA
INSERT INTO users (id, name, email, password) VALUES 
('66666666-6666-6666-6666-666666666666', 'Admin JICA', 'admin@jica.go.jp', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');

-- Admin MRT
INSERT INTO users (id, name, email, password) VALUES 
('77777777-7777-7777-7777-777777777777', 'Admin MRT', 'admin@jakartamrt.co.id', '$2a$10$.q6qwzesAfrCF8xw9TEnfOVo322LcznuqYThBZQx8CGOYAxUEU2e.');

-- Link Admin JICA -> JICA Org
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES 
('88888888-8888-8888-8888-888888888888', 'ADMIN', '66666666-6666-6666-6666-666666666666', '44444444-4444-4444-4444-444444444444');

-- Link Admin MRT -> MRT Org
INSERT INTO organization_members (id, role, user_id, organization_id) VALUES 
('99999999-9999-9999-9999-999999999999', 'ADMIN', '77777777-7777-7777-7777-777777777777', '55555555-5555-5555-5555-555555555555');


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Hapus data berdasarkan ID Hardcoded agar bersih saat Rollback
DELETE FROM organization_members WHERE id = '33333333-3333-3333-3333-333333333333';
DELETE FROM users WHERE id = '22222222-2222-2222-2222-222222222222';
DELETE FROM organizations WHERE id = '11111111-1111-1111-1111-111111111111';


-- Hapus Optional Data
DELETE FROM organization_members WHERE id IN ('88888888-8888-8888-8888-888888888888', '99999999-9999-9999-9999-999999999999');
DELETE FROM users WHERE id IN ('66666666-6666-6666-6666-666666666666', '77777777-7777-7777-7777-777777777777');
DELETE FROM organizations WHERE id IN ('44444444-4444-4444-4444-444444444444', '55555555-5555-5555-5555-555555555555');

-- +goose StatementEnd