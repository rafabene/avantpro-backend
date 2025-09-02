# Services Documentation

This document provides comprehensive documentation for all service interfaces and their methods in the AvantPro Backend API.

## 📋 Table of Contents

1. [Organization Service](#organization-service)
2. [User Service](#user-service)
3. [Authentication Service](#authentication-service)
4. [Email Service](#email-service)

---

## 🏢 Organization Service

The `OrganizationService` provides business logic for organization management including CRUD operations, member management, and invitation handling.

### Interface: `OrganizationServiceInterface`

#### Organization CRUD Operations

##### `CreateOrganization`
Creates a new organization with the specified creator as admin.
- **Parameters:**
  - `req` - Organization creation request containing name and description
  - `creatorID` - UUID of the user creating the organization
- **Returns:** The created organization with all related data
- **Business Rules:** The creator is automatically added as an admin member

##### `GetOrganization`
Retrieves an organization by its ID with all related data.
- **Parameters:**
  - `id` - UUID of the organization to retrieve
- **Returns:** Organization with creator, members, and invites loaded

##### `GetUserOrganizations`
Retrieves all organizations created by a specific user.
- **Parameters:**
  - `userID` - UUID of the user whose organizations to retrieve
  - `limit` - Maximum number of results to return
  - `offset` - Number of results to skip (for pagination)
  - `sortBy` - Field to sort by (name, created_at, updated_at)
  - `sortOrder` - Sort direction (asc, desc)
- **Returns:** List of organizations and total count

##### `UpdateOrganization`
Updates an existing organization's details.
- **Parameters:**
  - `id` - UUID of the organization to update
  - `req` - Update request containing new name and/or description
  - `userID` - UUID of the user requesting the update (must be admin)
- **Returns:** The updated organization
- **Business Rules:** Only admin members can update organization information

##### `DeleteOrganization`
Soft-deletes an organization.
- **Parameters:**
  - `id` - UUID of the organization to delete
  - `userID` - UUID of the user requesting deletion (must be creator)
- **Business Rules:** Only the original creator can delete an organization

##### `ListOrganizations`
Retrieves all organizations (admin function).
- **Parameters:**
  - `limit` - Maximum number of results to return
  - `offset` - Number of results to skip (for pagination)
  - `sortBy` - Field to sort by (name, created_at, updated_at)
  - `sortOrder` - Sort direction (asc, desc)
- **Returns:** List of organizations and total count

#### Organization Member Management

##### `GetOrganizationMembers`
Retrieves all members of an organization.
- **Parameters:**
  - `orgID` - UUID of the organization
  - `userID` - UUID of the requesting user (must be a member)
  - Pagination and sorting parameters
- **Returns:** List of members with user details
- **Business Rules:** Only existing members can view the member list

##### `UpdateMemberRole`
Updates a member's role within an organization.
- **Parameters:**
  - `orgID` - UUID of the organization
  - `memberUserID` - UUID of the member whose role to update
  - `req` - Update request containing the new role (admin or user)
  - `requestorID` - UUID of the user making the request (must be admin)
- **Returns:** The updated member with new role
- **Business Rules:** 
  - Only admin members can change roles
  - Organization creator's role cannot be changed from admin

##### `RemoveMember`
Removes a member from an organization.
- **Parameters:**
  - `orgID` - UUID of the organization
  - `memberUserID` - UUID of the member to remove
  - `requestorID` - UUID of the user making the request
- **Business Rules:**
  - Admin members can remove any other member (except the creator)
  - Regular members can only remove themselves
  - The organization creator cannot be removed

##### `GetUserMemberships`
Retrieves all organizations a user is a member of.
- **Parameters:**
  - `userID` - UUID of the user whose memberships to retrieve
  - Pagination and sorting parameters
- **Returns:** List of memberships with organization details

#### Organization Invitation Management

##### `InviteUser`
Sends an invitation for a user to join an organization.
- **Parameters:**
  - `orgID` - UUID of the organization
  - `req` - Invitation request containing email and role for the invitee
  - `inviterID` - UUID of the user sending the invitation (must be admin)
- **Returns:** The created invitation with token and expiry
- **Business Rules:**
  - Only admin members can send invitations
  - Checks for existing members and pending invitations
  - Automatically sends email invitation

##### `GetOrganizationInvites`
Retrieves all pending invitations for an organization.
- **Parameters:**
  - `orgID` - UUID of the organization
  - `userID` - UUID of the requesting user (must be admin)
  - Pagination and sorting parameters
- **Returns:** List of invitations with details
- **Business Rules:** Only admin members can view invitations

##### `AcceptInvite`
Accepts an organization invitation using the invitation token.
- **Parameters:**
  - `token` - Unique invitation token from the email link
  - `userID` - UUID of the user accepting the invitation
- **Returns:** The created membership record
- **Business Rules:**
  - Invitation must be valid and not expired
  - Email must match the user's email
  - User becomes a member with the specified role

##### `RevokeInvite`
Cancels a pending organization invitation.
- **Parameters:**
  - `inviteID` - UUID of the invitation to revoke
  - `userID` - UUID of the user revoking the invitation (must be admin)
- **Business Rules:** Only admin members can revoke invitations

### Helper Methods

#### `isUserAdmin`
Checks if a user has administrative privileges in an organization.
- **Parameters:**
  - `org` - The organization to check admin status for
  - `userID` - UUID of the user to check
- **Returns:** `true` if user has admin privileges
- **Business Rules:**
  - Organization creators are always considered admins
  - Members with `OrganizationRoleAdmin` are considered admins
  - Non-members and regular users are not admins

---

## 👤 User Service

The `UserService` provides business logic for user management with proper validation, error handling, and business rule enforcement.

### Interface: `UserService`

##### `CreateUser`
Validates and creates a new user in the system.
- **Parameters:**
  - `req` - User creation request containing user details and optional profile
- **Returns:** Created user data (password excluded)
- **Business Rules:**
  - Email format validation and uniqueness check
  - Password strength requirements
  - Profile data validation if provided

##### `GetUserByID`
Retrieves a user by their unique identifier.
- **Parameters:**
  - `id` - UUID of the user to retrieve
- **Returns:** User data with profile (password excluded)

##### `GetUserByUsername`
Retrieves a user by their username (email address).
- **Parameters:**
  - `username` - Email address of the user to find
- **Returns:** User data with profile (password excluded)
- **Usage:** Commonly used for login and user lookup operations

##### `ListUsers`
Retrieves a paginated list of users with validation and sorting.
- **Parameters:**
  - `page` - Page number for pagination (1-based)
  - `limit` - Number of users per page (max 100)
  - `sortBy` - Field to sort by (name, username, created_at, updated_at)
  - `sortOrder` - Sort direction (asc, desc)
- **Returns:** Paginated user list with metadata

##### `UpdateUser`
Validates and updates an existing user's information.
- **Parameters:**
  - `id` - UUID of the user to update
  - `req` - Update request containing new user data
- **Returns:** Updated user data (password excluded)
- **Business Rules:** Allows partial updates, only provided fields are updated

##### `DeleteUser`
Removes a user by their unique identifier.
- **Parameters:**
  - `id` - UUID of the user to delete
- **Business Rules:** Performs soft delete, preserving data integrity

---

## 🔐 Authentication Service

The `AuthService` provides authentication and authorization services including user login, registration, password management, and JWT token generation.

### Interface: `AuthService`

##### `Login`
Authenticates a user with email and password.
- **Parameters:**
  - `req` - Login request containing email and password
- **Returns:** JWT token and user information
- **Business Rules:** Validates credentials and generates JWT token

##### `Register`
Creates a new user account and automatically logs them in.
- **Parameters:**
  - `req` - Registration request containing email, name, and password
- **Returns:** JWT token and user information
- **Business Rules:** Validates user data, creates account, and returns JWT token

##### `RequestPasswordReset`
Initiates the password reset process for a user.
- **Parameters:**
  - `email` - Email address of the user requesting password reset
- **Business Rules:** Generates reset token and sends password reset email

##### `ResetPassword`
Completes the password reset process using a reset token.
- **Parameters:**
  - `token` - Password reset token from email
  - `newPassword` - New password to set for the user
- **Business Rules:** Validates reset token and updates user's password

---

## 📧 Email Service

The `EmailService` provides email operations for organization management and user notifications. Currently implemented as a logging service for development.

### Interface: `EmailServiceInterface`

##### `SendOrganizationInvite`
Sends an invitation email to a user to join an organization.
- **Parameters:**
  - `invite` - Organization invitation containing recipient email, organization details, and token
  - `baseURL` - Base URL of the application for generating the invitation acceptance link
- **Business Rules:**
  - Creates unique invitation link
  - Includes organization name and role information
  - Shows expiration date and instructions

### Current Implementation
- **Development:** Logs email content instead of sending actual emails
- **Production Considerations:**
  - SMTP server configuration
  - HTML email templates
  - Email delivery tracking
  - Retry mechanisms for failed deliveries
  - Email queue processing for high volume

---

## 🔄 Service Dependencies

### Organization Service Dependencies
- `OrganizationRepositoryInterface` - For organization data operations
- `UserRepository` - For user data operations
- `EmailServiceInterface` - For sending email notifications

### User Service Dependencies
- `UserRepository` - For user data operations
- `validator.Validate` - For input validation

### Auth Service Dependencies
- `UserRepository` - For user data operations
- `jwtSecret` - For JWT token signing and validation

### Email Service Dependencies
- None (currently self-contained logging implementation)

---

## 🛡️ Security Considerations

### Permission Checks
- Organization creators have permanent admin privileges
- Admin role validation for sensitive operations
- Member validation for access to organization data
- Email verification for invitation acceptance

### Data Protection
- Passwords are never returned in API responses
- JWT tokens for authentication
- Soft deletes to preserve data integrity
- Input validation on all service methods

### Business Rules Enforcement
- Creator cannot be removed from organization
- Creator role cannot be changed
- Invitation email must match user email
- Only admins can manage organization settings

---

This documentation serves as a comprehensive guide for understanding the service layer architecture and business logic implementation in the AvantPro Backend API.