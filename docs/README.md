# Documentation Index

- [Development guide](development.md): local setup, commands, verification, and generated artifacts
- [Architecture overview](architecture.md): repo layout, request flow, and shipped frontend/backend boundaries
- [API and route behavior](api.md): browser pages, JSON contracts, auth fallback submits, and CSRF expectations
- [Frontend migration inventory](frontend-migration-inventory.md): historical route inventory and the final migration state
- [Security notes](security.md): current baseline security posture and obvious next hardening steps
- [Deployment notes](deployment.md): app/server configuration guidance
- [Ubuntu deployment walkthrough](ubuntu-deployment.md): concrete deployment example
- [Example YAML config](config.example.yaml): reference config structure

## Notes

- Generated SQLC output and the shipped `web/dist` frontend are checked in.
- Run `mage generate` after schema or SQL query changes.
- Run the frontend checks and build after changing `web/`.
