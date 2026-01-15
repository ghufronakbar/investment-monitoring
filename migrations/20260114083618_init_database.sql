-- +goose Up
-- +goose StatementBegin

-- 1. Setup Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 2. Setup Trigger Function for updated_at
-- Fungsi ini akan dipanggil oleh trigger di setiap tabel
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 3. Create ENUM Types
CREATE TYPE organization_role_enum AS ENUM (
    'GOV_SUPERVISOR',
    'LOAN_PROVIDER',
    'EXECUTING_AGENCY',
    'CONTRACTOR',
    'INDEPENDENT_AUDITOR',
    'GOV_PARTNER'
);

CREATE TYPE member_role_enum AS ENUM (
    'ADMIN',
    'MEMBER'
);

CREATE TYPE ticket_status_enum AS ENUM (
    'OPEN',
    'ASSIGNED',
    'IN_PROGRESS',
    'RESOLVED',
    'CLOSED'
);

CREATE TYPE ticket_log_action_enum AS ENUM (
    'CREATE',
    'ASSIGNED',
    'COMMENTED',
    'STATUS_CHANGED',
    'RESOLVED'
);

CREATE TYPE payment_request_status_enum AS ENUM (
    'SUBMITTED',
    'AUDITOR_APPROVED',
    'AUDITOR_REJECTED',
    'AGENCY_APPROVED',
    'AGENCY_REJECTED',
    'PAID'
);

-- 4. Create Tables & Triggers

-- Table: organizations
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    logo_url TEXT,
    role organization_role_enum NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_organizations_modtime BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_url TEXT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_users_modtime BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_urls TEXT[], 
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    contract_file_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_projects_modtime BEFORE UPDATE ON projects FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: organization_members
CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role member_role_enum NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_organization_members_modtime BEFORE UPDATE ON organization_members FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: project_settings
CREATE TABLE project_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    start_date TIMESTAMPTZ NOT NULL,
    baseline_deadline TIMESTAMPTZ NOT NULL,
    total_budget BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'IDR',
    project_id UUID NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_project_settings_modtime BEFORE UPDATE ON project_settings FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: contracts
CREATE TABLE contracts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    description TEXT NOT NULL,
    project_id UUID NOT NULL UNIQUE REFERENCES projects(id) ON DELETE RESTRICT,
    
    contract_value BIGINT,
    contract_doc_url TEXT,
    
    -- Roles Mapping
    gov_supervisor_participant_id UUID NOT NULL REFERENCES organizations(id),
    loan_provider_participant_id UUID NOT NULL REFERENCES organizations(id),
    executing_agency_participant_id UUID REFERENCES organizations(id),
    
    -- Winner / Vendor
    vendor_participant_id UUID REFERENCES organizations(id),
    vendor_participant_selected_at TIMESTAMPTZ,
    
    -- Auditor
    auditor_participant_id UUID REFERENCES organizations(id),
    auditor_participant_selected_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_contracts_modtime BEFORE UPDATE ON contracts FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: gov_participants (Pivot)
CREATE TABLE gov_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    gov_participant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_gov_participants_modtime BEFORE UPDATE ON gov_participants FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: task_items
CREATE TABLE task_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    deadline TIMESTAMPTZ NOT NULL,
    weight INT NOT NULL,
    parent_id UUID REFERENCES task_items(id) ON DELETE SET NULL, 
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_task_items_modtime BEFORE UPDATE ON task_items FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: payment_requests
CREATE TABLE payment_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    amount BIGINT NOT NULL,
    status payment_request_status_enum NOT NULL,
    invoice_file_url TEXT NOT NULL,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE RESTRICT,
    
    -- Approval Chain
    auditor_org_id UUID REFERENCES organizations(id),
    auditor_user_id UUID REFERENCES users(id),
    auditor_at TIMESTAMPTZ,
    
    executing_agency_org_id UUID REFERENCES organizations(id),
    executing_agency_user_id UUID REFERENCES users(id),
    executing_agency_at TIMESTAMPTZ,
    
    loan_provider_org_id UUID REFERENCES organizations(id),
    loan_provider_user_id UUID REFERENCES users(id),
    loan_provider_at TIMESTAMPTZ,
    payment_proof_image_urls TEXT[] NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_payment_requests_modtime BEFORE UPDATE ON payment_requests FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: tickets
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    location TEXT NOT NULL,
    evidence_image_urls TEXT[], 
    task_item_id UUID REFERENCES task_items(id) ON DELETE SET NULL,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE RESTRICT,
    
    -- Creator
    ticket_assigner_contractor_id UUID NOT NULL REFERENCES organizations(id),
    ticket_assigner_user_id UUID NOT NULL REFERENCES users(id),
    
    -- Assignee
    ticket_assign_to_id UUID REFERENCES organizations(id),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_tickets_modtime BEFORE UPDATE ON tickets FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: progress_logs
CREATE TABLE progress_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report TEXT NOT NULL,
    progress_percentage INT NOT NULL,
    evidence_image_urls TEXT[],
    
    -- Snapshot
    snapshot_title TEXT NOT NULL,
    snapshot_description TEXT NOT NULL,
    snapshot_weight INT NOT NULL,
    
    task_item_id UUID NOT NULL REFERENCES task_items(id) ON DELETE RESTRICT,
    vendor_id UUID NOT NULL REFERENCES organizations(id),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_progress_logs_modtime BEFORE UPDATE ON progress_logs FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: ticket_logs
CREATE TABLE ticket_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action ticket_log_action_enum NOT NULL,
    description TEXT NOT NULL,
    attachment_url TEXT,
    previous_status ticket_status_enum,
    new_status ticket_status_enum NOT NULL,
    
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_ticket_logs_modtime BEFORE UPDATE ON ticket_logs FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Table: progress_validations
CREATE TABLE progress_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    is_accepted BOOLEAN NOT NULL,
    comment TEXT,
    real_progress_percentage INT NOT NULL,
    progress_log_id UUID NOT NULL UNIQUE REFERENCES progress_logs(id) ON DELETE CASCADE,
    auditor_id UUID NOT NULL REFERENCES organizations(id),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_progress_validations_modtime BEFORE UPDATE ON progress_validations FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- EXAMPLE TABLE
CREATE TABLE examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_examples_modtime BEFORE UPDATE ON examples FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- EXAMPLE TABLE
DROP TABLE IF EXISTS examples;

-- Drop Tables (Trigger will be dropped automatically with table)
DROP TABLE IF EXISTS progress_validations;
DROP TABLE IF EXISTS ticket_logs;
DROP TABLE IF EXISTS progress_logs;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS payment_requests;
DROP TABLE IF EXISTS task_items;
DROP TABLE IF EXISTS gov_participants;
DROP TABLE IF EXISTS contracts;
DROP TABLE IF EXISTS project_settings;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

-- Drop Function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop Enums
DROP TYPE IF EXISTS payment_request_status_enum;
DROP TYPE IF EXISTS ticket_log_action_enum;
DROP TYPE IF EXISTS ticket_status_enum;
DROP TYPE IF EXISTS member_role_enum;
DROP TYPE IF EXISTS organization_role_enum;

-- +goose StatementEnd