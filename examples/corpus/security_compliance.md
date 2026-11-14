# Security, Encryption, and Regulatory Compliance

This document outlines our information security controls, encryption specifications, and regulatory compliance standards.

## Encryption Standards
All customer data in transit is encrypted using TLS 1.3 with mandatory forward secrecy. TLS 1.0, 1.1, and unencrypted HTTP traffic are permanently rejected.
Data at rest is encrypted using AES-256 with keys managed in cloud Hardware Security Modules (HSM) conforming to FIPS 140-2 Level 3.

## Audit Logs and Retention
Audit logs record every authentication event, privilege escalation, and data export operation.
Logs are immutable and retained for a minimum duration of 365 days (1 year) to satisfy SOC 2 Type II and ISO 27001 audit mandates.

## Vulnerability Disclosure and Incident Response
Critical security vulnerabilities reported through our responsible disclosure program receive an initial triage response within 24 hours.
Remediation patches for severity rating 9.0+ CVEs are deployed within 72 hours of verification.
