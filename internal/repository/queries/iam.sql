-- internal/repository/queries/iam.sql
-- SQLC queries for IAM (Identity & Access Management) module

-- ===== USER QUERIES =====

-- name: GetUserByEmailFull :one
-- Get user with all fields including password for authentication
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
-- Get user by ID 
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
-- Create new user (password can be null for invited users)
INSERT INTO users (name, email, password, image_url) 
VALUES ($1, $2, $3, $4) 
RETURNING *;

-- name: SetUserPassword :one
-- Set password for user (used when user activates account)
UPDATE users 
SET password = $2 
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CheckUserEmailExists :one
-- Check if email already exists
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL) as exists;

-- ===== ORGANIZATION QUERIES =====

-- name: CreateOrganization :one
-- Create new organization
INSERT INTO organizations (name, logo_url, role) 
VALUES ($1, $2, $3) 
RETURNING *;

-- name: GetOrganizationByID :one
-- Get organization by ID
SELECT * FROM organizations WHERE id = $1 AND deleted_at IS NULL;

-- name: ListOrganizations :many
-- List all organizations with pagination
SELECT * FROM organizations 
WHERE deleted_at IS NULL 
ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;

-- name: CountOrganizations :one
-- Count total organizations
SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL;

-- name: UpdateOrganization :one
-- Update organization name and logo
UPDATE organizations 
SET name = $2, logo_url = $3 
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;


-- ===== ORGANIZATION MEMBER QUERIES =====

-- name: CreateOrganizationMember :one
-- Link user to organization with specific role
INSERT INTO organization_members (role, user_id, organization_id)
VALUES ($1, $2, $3) 
RETURNING *;

-- name: GetOrganizationMember :one
-- Get membership details
SELECT 
    om.*,
    o.name as org_name,
    o.role as org_role,
    u.name as user_name,
    u.email as user_email
FROM organization_members om
JOIN organizations o ON o.id = om.organization_id
JOIN users u ON u.id = om.user_id
WHERE om.user_id = $1 AND om.organization_id = $2 AND om.deleted_at IS NULL;

-- name: GetUserMemberships :many
-- Get all organization memberships for a user
SELECT 
    om.id,
    om.role as member_role,
    om.created_at,
    o.id as org_id,
    o.name as org_name,
    o.logo_url as org_logo_url,
    o.role as org_role
FROM organization_members om
JOIN organizations o ON o.id = om.organization_id
WHERE om.user_id = $1 AND om.deleted_at IS NULL AND o.deleted_at IS NULL
ORDER BY om.created_at DESC;

-- name: GetOrganizationMembers :many
-- Get all members of an organization
SELECT 
    om.id,
    om.role as member_role,
    om.created_at,
    u.id as user_id,
    u.name as user_name,
    u.email as user_email,
    u.image_url as user_image_url
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1 AND om.deleted_at IS NULL AND u.deleted_at IS NULL
ORDER BY om.created_at DESC;

-- name: CheckMembershipExists :one
-- Check if user is already a member of organization
SELECT EXISTS(
    SELECT 1 FROM organization_members 
    WHERE user_id = $1 AND organization_id = $2 AND deleted_at IS NULL
) as exists;

-- ===== PROFILE QUERIES =====

-- name: UpdateUserProfile :one
-- Update user profile (name, image_url)
UPDATE users 
SET name = $2, image_url = $3 
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserPasswordByID :one
-- Update user password by ID
UPDATE users 
SET password = $2 
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

