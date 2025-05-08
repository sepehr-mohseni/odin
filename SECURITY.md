# Security Policy

## Supported Versions

We currently support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |

## Reporting a Vulnerability

We take the security of Odin API Gateway seriously. If you believe you have found a security vulnerability, please follow these steps:

1. **Do NOT disclose the vulnerability publicly**
2. **Email us at [your-security-email@example.com]** with details about the vulnerability
3. Allow us time to investigate and address the vulnerability
4. We will coordinate with you on the disclosure timeline

## Security Best Practices

When deploying Odin API Gateway, we recommend the following security best practices:

1. Always use HTTPS in production
2. Use strong, randomly generated secrets for JWT signing
3. Regularly rotate credentials and secrets
4. Apply the principle of least privilege for service accounts
5. Keep the gateway and its dependencies updated
6. Configure proper rate limiting
7. Monitor logs for suspicious activities
8. Use network segmentation to limit access to the gateway

## Security Features

Odin API Gateway provides several security features:

- JWT-based authentication
- Role-based access control
- Request rate limiting
- Response validation
- Transport-level security
- Secure configuration storage
