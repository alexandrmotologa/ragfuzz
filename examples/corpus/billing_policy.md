# Enterprise Subscription and Billing Policy

Our subscription tiers define compute quotas, storage limits, and support SLAs.

## Storage Quotas and File Size Limits
In Subscription Plan v2.0, the maximum file upload size for customer attachments is 50MB per document.
Total workspace storage allocation is capped at 500GB for Enterprise tenants and 50GB for Starter accounts.
Note that earlier v1.0 specifications restricted attachment size to 10MB; tenants migrated after Q1 2025 operate under the 50MB limit.

## Rate Limiting and Burst Capacity
The standard API rate limit is 100 requests per second (100rps) with a maximum burst capacity of 250 requests over a 10-second window.
Enterprise accounts can request rate limit increases up to 1000rps through their dedicated account manager.

## Invoicing and Payment Terms
Invoices are generated on the first calendar day of each month with net-30 payment terms.
Late payments exceeding 45 days incur a 1.5% monthly late fee and trigger an automated warning banner in the billing portal.
Only users with the billing_admin or owner role can view credit card details and download tax invoices.
