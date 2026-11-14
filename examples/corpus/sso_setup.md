# Enterprise Single Sign-On (SSO) Integration Guide

Single Sign-On (SSO) provides centralized identity federation across corporate enterprise directories using SAML 2.0 and OpenID Connect (OIDC).

## Identity Provider Support
We support Okta, Microsoft Azure Active Directory, and Google Workspace.
PingFederate and legacy Active Directory Federation Services (ADFS) are not supported on the Starter or Professional plans; they require an Enterprise Plus custom contract.

## Token Expiration and Session Timeouts
All session tokens generated via SAML authentication expire after exactly 3600 seconds (1 hour).
Refresh tokens are valid for up to 30 days unless revoked by an administrator. Idle sessions terminate automatically after 900 seconds of inactivity.

## Access Permissions and Roles
Only users assigned the role of administrator or security_auditor are permitted to upload identity provider metadata XML files or rotate X.509 signing certificates.
Users with the viewer role can inspect active directory mappings but cannot modify SAML attributes.
